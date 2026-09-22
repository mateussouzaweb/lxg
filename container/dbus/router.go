package dbus

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mateussouzaweb/lxg/command"
)

// Router manages routing between container clients, HostSocket, and ContainerSocket
type Router struct {
	UID             string
	RouterSocket    string
	HostSocket      string
	ContainerSocket string
	listener        net.Listener
	closed          atomic.Bool
	serialCounter   atomic.Uint32
}

// NewRouter creates a new D-Bus router instance
func NewRouter(ctx *command.Context) *Router {
	uid := ctx.UID
	return &Router{
		UID:             uid,
		RouterSocket:    fmt.Sprintf("/run/user/%s/lxg.router", uid),
		HostSocket:      fmt.Sprintf("/run/user/%s/lxg.bus", uid),
		ContainerSocket: fmt.Sprintf("/run/user/%s/bus", uid),
	}
}

// NextSerial generates an incrementing serial number
func (r *Router) NextSerial() uint32 {
	return r.serialCounter.Add(1)
}

// Start listens on RouterSocket and dispatches connections
func (r *Router) Start(ctx context.Context) error {

	// Clean up previous socket if exists
	err := os.Remove(r.RouterSocket)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old router socket error: %w", err)
	}

	// Ensure runtime directory exists
	err = os.MkdirAll(filepath.Dir(r.RouterSocket), 0700)
	if err != nil {
		return fmt.Errorf("create router directory error: %w", err)
	}

	listener, err := net.Listen("unix", r.RouterSocket)
	if err != nil {
		return fmt.Errorf("listen router socket error: %w", err)
	}

	r.listener = listener
	defer listener.Close()
	defer os.Remove(r.RouterSocket)

	// Set socket permissions
	err = os.Chmod(r.RouterSocket, 0600)
	if err != nil {
		return fmt.Errorf("chmod router socket error: %w", err)
	}

	fmt.Printf("LXG D-Bus Router listening on %s\n", r.RouterSocket)

	// Close listener on context cancel
	go func() {
		<-ctx.Done()
		r.closed.Store(true)
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if r.closed.Load() || errors.Is(err, net.ErrClosed) {
				break
			}
			fmt.Printf("Router accept error: %s\n", err)
			continue
		}

		unixConn, ok := conn.(*net.UnixConn)
		if !ok {
			conn.Close()
			continue
		}

		go r.handleClient(unixConn)
	}

	return nil
}

// ClientSession manages a single client connection and its upstream links
type ClientSession struct {
	router        *Router
	clientConn    *net.UnixConn
	hostConn      *net.UnixConn
	containerConn *net.UnixConn
	clientWriteMu sync.Mutex
	closed        atomic.Bool
}

func (r *Router) handleClient(clientConn *net.UnixConn) {

	dateTime := time.Now()
	fmt.Printf("Received new request: %s\n", dateTime.Format(time.RFC3339))

	// Initiate session
	session := &ClientSession{
		router:     r,
		clientConn: clientConn,
	}

	defer session.Close()

	// Authenticate client
	extraBytes, err := AuthenticateClient(clientConn)
	if err != nil {
		fmt.Printf("Client SASL auth failed: %s\n", err)
		return
	}

	// Connect to ContainerSocket (native container bus)
	cConn, err := net.Dial("unix", r.ContainerSocket)
	if err != nil {
		fmt.Printf("Dial ContainerSocket (%s) error: %s\n", r.ContainerSocket, err)
		return
	}

	session.containerConn = cConn.(*net.UnixConn)
	err = AuthenticateUpstream(session.containerConn)
	if err != nil {
		fmt.Printf("ContainerSocket SASL auth failed: %s\n", err)
		return
	}

	// Connect to HostSocket (host proxy) if available
	hConn, err := net.Dial("unix", r.HostSocket)
	if err == nil {
		session.hostConn = hConn.(*net.UnixConn)
		err = AuthenticateUpstream(session.hostConn)
		if err != nil {
			fmt.Printf("HostSocket SASL auth failed: %s\n", err)
			session.hostConn.Close()
			session.hostConn = nil
		}
	} else {
		// Non-fatal: container apps will still work locally
		session.hostConn = nil
	}

	// Set up message readers
	clientReader := NewMessageReader(clientConn)
	if len(extraBytes) > 0 {
		clientReader.buf = append(clientReader.buf, extraBytes...)
	}

	containerReader := NewMessageReader(session.containerConn)
	hostReader := &MessageReader{}

	if session.hostConn != nil {
		hostReader = NewMessageReader(session.hostConn)
	}

	// Start upstream forwarders to client
	var wg sync.WaitGroup

	// Container -> Client forwarder
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer session.Close()
		for {
			msg, err := containerReader.ReadMessage()
			if err != nil {
				break
			}
			err = session.SendToClient(msg)
			msg.CloseFDs()
			if err != nil {
				break
			}
		}
	}()

	// Host -> Client forwarder
	if hostReader != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer session.Close()
			for {
				msg, err := hostReader.ReadMessage()
				if err != nil {
					break
				}
				err = session.SendToClient(msg)
				msg.CloseFDs()
				if err != nil {
					break
				}
			}
		}()
	}

	// Client -> Upstream dispatcher loop
	for {
		msg, err := clientReader.ReadMessage()
		if err != nil {
			break
		}

		err = session.DispatchClientMessage(msg)
		msg.CloseFDs()
		if err != nil {
			break
		}
	}

	session.Close()
	wg.Wait()
}

// SendToClient writes a message safely to the client connection
func (s *ClientSession) SendToClient(msg *Message) error {
	s.clientWriteMu.Lock()
	defer s.clientWriteMu.Unlock()
	return WriteMessage(s.clientConn, msg)
}

// DispatchClientMessage routes an incoming client message to the appropriate upstream
func (s *ClientSession) DispatchClientMessage(msg *Message) error {

	// Special handling for bus management calls on org.freedesktop.DBus
	if msg.Destination == "org.freedesktop.DBus" || msg.Destination == "" {
		return s.handleDBusManagement(msg)
	}

	// Normal method calls: check route target
	target := RouteDestination(msg.Destination)
	if target == TargetHost {
		if s.hostConn != nil {
			return WriteMessage(s.hostConn, msg)
		}

		// Host proxy is not available: return error
		errMsg := NewErrorMessage(msg, "org.freedesktop.DBus.Error.ServiceUnknown",
			fmt.Sprintf("The name %s is a host service but host proxy is unavailable", msg.Destination),
			s.router.NextSerial())

		return s.SendToClient(errMsg)
	}

	// Default fallback: ContainerSocket
	return WriteMessage(s.containerConn, msg)
}

// handleDBusManagement handles calls addressed to org.freedesktop.DBus
func (s *ClientSession) handleDBusManagement(msg *Message) error {
	switch msg.Member {
	case "AddMatch", "RemoveMatch":
		// Check if match rule targets host
		rule := extractFirstStringArg(msg)
		isHostSignal := MatchSignalRuleHost(rule)

		if isHostSignal && s.hostConn != nil {
			// Duplicate FDs and raw bytes for host so there are no shared references
			dupFDs, _ := CopyFDs(msg.FDs)
			rawCopy := make([]byte, len(msg.Raw))
			copy(rawCopy, msg.Raw)
			hostMsg := &Message{Raw: rawCopy, FDs: dupFDs}
			_ = WriteMessage(s.hostConn, hostMsg)
		}

		// Also register on container (container will send the reply back to client)
		return WriteMessage(s.containerConn, msg)

	case "RequestName", "ReleaseName":
		name := extractFirstStringArg(msg)
		target := RouteOwnership(name)

		if target == TargetHost && s.hostConn != nil {
			return WriteMessage(s.hostConn, msg)
		}
		return WriteMessage(s.containerConn, msg)

	case "GetNameOwner":
		name := extractFirstStringArg(msg)
		target := RouteDestination(name)

		if target == TargetHost && s.hostConn != nil {
			return WriteMessage(s.hostConn, msg)
		}
		return WriteMessage(s.containerConn, msg)

	default:
		// Send Hello and other methods to ContainerSocket.
		// When Hello is sent, container bus returns canonical unique name (:1.x).
		// Also register on HostSocket so the host proxy tracks this connection.
		if msg.Member == "Hello" && s.hostConn != nil {
			dupFDs, _ := CopyFDs(msg.FDs)
			rawCopy := make([]byte, len(msg.Raw))
			copy(rawCopy, msg.Raw)
			hostMsg := &Message{Raw: rawCopy, FDs: dupFDs}
			_ = WriteMessage(s.hostConn, hostMsg)
		}

		return WriteMessage(s.containerConn, msg)
	}
}

// Close closes all connections in this session
func (s *ClientSession) Close() {
	if s.closed.CompareAndSwap(false, true) {
		if s.clientConn != nil {
			s.clientConn.Close()
		}
		if s.hostConn != nil {
			s.hostConn.Close()
		}
		if s.containerConn != nil {
			s.containerConn.Close()
		}
	}
}

// Helper: extract the first string argument from the message body
func extractFirstStringArg(msg *Message) string {

	// In D-Bus wire format:
	// The body starts after HeaderLength
	if int(msg.HeaderLength)+4 > len(msg.Raw) {
		return ""
	}

	body := msg.Raw[msg.HeaderLength:]
	var order binary.ByteOrder
	if msg.Endianness == 'l' {
		order = binary.LittleEndian
	} else {
		order = binary.BigEndian
	}

	if len(body) < 4 {
		return ""
	}

	strLen := int(order.Uint32(body[:4]))
	if len(body) < 4+strLen {
		return ""
	}

	return string(body[4 : 4+strLen])
}

// NewErrorMessage constructs a D-Bus Error response message
func NewErrorMessage(replyTo *Message, errorName string, errorText string, serial uint32) *Message {

	// Error body is signature 's'
	bodyBytes := make([]byte, 4+len(errorText)+1)
	binary.LittleEndian.PutUint32(bodyBytes[:4], uint32(len(errorText)))
	copy(bodyBytes[4:], errorText)
	bodyBytes[4+len(errorText)] = 0
	bodyLen := uint32(len(bodyBytes))

	// Build header fields:
	// 4: ErrorName (signature 's')
	// 5: ReplySerial (signature 'u')
	// 7: Destination (signature 's', if replyTo.Sender != "")
	// 8: Signature (signature 'g') = "s"
	var fields []byte

	// Field 4: ErrorName
	fields = appendFieldString(fields, HeaderErrorName, errorName)
	// Field 5: ReplySerial
	fields = appendFieldUint32(fields, HeaderReplySerial, replyTo.Serial)
	// Field 8: Signature "s"
	fields = appendFieldSignature(fields, "s")

	headerFieldsLen := uint32(len(fields))
	paddedHeaderLen := 16 + headerFieldsLen
	if rem := paddedHeaderLen % 8; rem != 0 {
		paddedHeaderLen += (8 - rem)
	}

	header := make([]byte, paddedHeaderLen)
	header[0] = 'l'       // Little endian
	header[1] = TypeError // Type = Error
	header[2] = 0x01      // No reply expected
	header[3] = 1         // Protocol version
	binary.LittleEndian.PutUint32(header[4:8], bodyLen)
	binary.LittleEndian.PutUint32(header[8:12], serial)
	binary.LittleEndian.PutUint32(header[12:16], headerFieldsLen)

	copy(header[16:], fields)
	raw := append(header, bodyBytes...)

	return &Message{
		Endianness:   'l',
		Type:         TypeError,
		Flags:        0x01,
		Version:      1,
		BodyLength:   bodyLen,
		Serial:       serial,
		HeaderLength: paddedHeaderLen,
		TotalLength:  uint32(len(raw)),
		ErrorName:    errorName,
		ReplySerial:  replyTo.Serial,
		Raw:          raw,
	}
}

func appendFieldString(buf []byte, code byte, val string) []byte {

	// Align struct to 8 bytes relative to offset 16
	offset := 16 + len(buf)
	if rem := offset % 8; rem != 0 {
		pad := 8 - rem
		for i := 0; i < pad; i++ {
			buf = append(buf, 0)
		}
	}

	buf = append(buf, code)      // field code
	buf = append(buf, 1, 's', 0) // variant signature "s\0"

	// String length (aligned to 4 bytes)
	offset = 16 + len(buf)
	if rem := offset % 4; rem != 0 {
		pad := 4 - rem
		for i := 0; i < pad; i++ {
			buf = append(buf, 0)
		}
	}

	valBytes := make([]byte, 4+len(val)+1)
	binary.LittleEndian.PutUint32(valBytes[:4], uint32(len(val)))
	copy(valBytes[4:], val)
	valBytes[4+len(val)] = 0

	return append(buf, valBytes...)
}

func appendFieldUint32(buf []byte, code byte, val uint32) []byte {

	offset := 16 + len(buf)
	if rem := offset % 8; rem != 0 {
		pad := 8 - rem
		for i := 0; i < pad; i++ {
			buf = append(buf, 0)
		}
	}

	buf = append(buf, code)
	buf = append(buf, 1, 'u', 0) // variant signature "u\0"

	offset = 16 + len(buf)
	if rem := offset % 4; rem != 0 {
		pad := 4 - rem
		for i := 0; i < pad; i++ {
			buf = append(buf, 0)
		}
	}

	valBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(valBytes, val)

	return append(buf, valBytes...)
}

func appendFieldSignature(buf []byte, sig string) []byte {

	offset := 16 + len(buf)
	if rem := offset % 8; rem != 0 {
		pad := 8 - rem
		for i := 0; i < pad; i++ {
			buf = append(buf, 0)
		}
	}

	buf = append(buf, HeaderSignature)
	buf = append(buf, 1, 'g', 0) // variant signature "g\0"
	buf = append(buf, byte(len(sig)))
	buf = append(buf, sig...)
	buf = append(buf, 0)

	return buf
}
