package dbus

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mateussouzaweb/lxg/command"
)

const (
	hostSwallow byte = iota
	hostForward
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

// PidFile is the path of the daemon pid file next to the router socket
func (r *Router) PidFile() string {
	return r.RouterSocket + ".pid"
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

	err = os.WriteFile(r.PidFile(), []byte(fmt.Sprintf("%d\n", os.Getpid())), 0600)
	if err != nil {
		return fmt.Errorf("write router pid file error: %w", err)
	}
	defer os.Remove(r.PidFile())

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

type hostPending struct {
	kind   byte
	member string
}

type mergeState struct {
	member  string
	hostMsg *Message
	contMsg *Message
}

// ClientSession manages a single client connection and its upstream links
type ClientSession struct {
	router          *Router
	clientConn      *net.UnixConn
	hostConn        *net.UnixConn
	containerConn   *net.UnixConn
	clientWriteMu   sync.Mutex
	mu              sync.Mutex
	closed          atomic.Bool
	hostPending     map[uint32]hostPending
	merges          map[uint32]*mergeState
	hostUniqueNames map[string]struct{}
	hostUniqueName  string
}

func (r *Router) handleClient(clientConn *net.UnixConn) {

	dateTime := time.Now()
	fmt.Printf("Received new request: %s\n", dateTime.Format(time.RFC3339))

	session := &ClientSession{
		router:          r,
		clientConn:      clientConn,
		hostPending:     make(map[uint32]hostPending),
		merges:          make(map[uint32]*mergeState),
		hostUniqueNames: make(map[string]struct{}),
	}

	var (
		clientReader    *MessageReader
		containerReader *MessageReader
		hostReader      *MessageReader
		wg              sync.WaitGroup
	)

	defer func() {
		session.Close()
		wg.Wait()
		if clientReader != nil {
			clientReader.Close()
		}
		if containerReader != nil {
			containerReader.Close()
		}
		if hostReader != nil {
			hostReader.Close()
		}
	}()

	extraBytes, err := AuthenticateClient(clientConn)
	if err != nil {
		if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			fmt.Printf("Client SASL auth failed: %s\n", err)
		}
		return
	}

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
		session.hostConn = nil
	}

	clientReader = NewMessageReader(clientConn)
	if len(extraBytes) > 0 {
		clientReader.buf = append(clientReader.buf, extraBytes...)
	}

	containerReader = NewMessageReader(session.containerConn)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer session.Close()
		for {
			msg, err := containerReader.ReadMessage()
			if err != nil {
				break
			}
			err = session.handleUpstream(TargetContainer, msg)
			if err != nil {
				msg.CloseFDs()
				break
			}
		}
	}()

	if session.hostConn != nil {
		hostReader = NewMessageReader(session.hostConn)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer session.Close()
			for {
				msg, err := hostReader.ReadMessage()
				if err != nil {
					break
				}
				err = session.handleUpstream(TargetHost, msg)
				if err != nil {
					msg.CloseFDs()
					break
				}
			}
		}()
	}

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
}

// SendToClient writes a message safely to the client connection
func (s *ClientSession) SendToClient(msg *Message) error {
	s.clientWriteMu.Lock()
	defer s.clientWriteMu.Unlock()
	return WriteMessage(s.clientConn, msg)
}

func (s *ClientSession) noteHostPending(serial uint32, kind byte, member string) {
	s.mu.Lock()
	s.hostPending[serial] = hostPending{kind: kind, member: member}
	s.mu.Unlock()
}

func (s *ClientSession) rememberHostUnique(name string) {
	if name == "" || !strings.HasPrefix(name, ":") {
		return
	}
	s.mu.Lock()
	s.hostUniqueNames[name] = struct{}{}
	s.mu.Unlock()
}

func (s *ClientSession) routeDestination(destination string) Target {
	if destination == "" {
		return TargetContainer
	}
	if RouteDestination(destination) == TargetHost {
		return TargetHost
	}
	if RouteOwnership(destination) == TargetHost {
		return TargetHost
	}
	s.mu.Lock()
	_, known := s.hostUniqueNames[destination]
	s.mu.Unlock()
	if known {
		return TargetHost
	}
	return TargetContainer
}

func (s *ClientSession) writeHostCopy(msg *Message) error {
	dupFDs, _ := CopyFDs(msg.FDs)
	rawCopy := make([]byte, len(msg.Raw))
	copy(rawCopy, msg.Raw)
	hostMsg := &Message{Raw: rawCopy, FDs: dupFDs}
	err := WriteMessage(s.hostConn, hostMsg)
	hostMsg.CloseFDs()
	return err
}

func (s *ClientSession) sendToHost(msg *Message, kind byte) error {
	if msg.Type == TypeMethodCall {
		s.noteHostPending(msg.Serial, kind, msg.Member)
	}
	err := WriteMessage(s.hostConn, msg)
	if err != nil && msg.Type == TypeMethodCall {
		s.mu.Lock()
		delete(s.hostPending, msg.Serial)
		s.mu.Unlock()
	}
	return err
}

// DispatchClientMessage routes an incoming client message to the appropriate upstream
func (s *ClientSession) DispatchClientMessage(msg *Message) error {
	if msg.Type == TypeSignal {
		return s.dispatchSignal(msg)
	}

	if msg.Destination == "org.freedesktop.DBus" {
		return s.handleDBusManagement(msg)
	}

	target := s.routeDestination(msg.Destination)
	if target == TargetHost {
		if s.hostConn != nil {
			return s.sendToHost(msg, hostForward)
		}

		errMsg := NewErrorMessage(msg, "org.freedesktop.DBus.Error.ServiceUnknown",
			fmt.Sprintf("The name %s is a host service but host proxy is unavailable", msg.Destination),
			s.router.NextSerial())

		return s.SendToClient(errMsg)
	}

	return WriteMessage(s.containerConn, msg)
}

func (s *ClientSession) dispatchSignal(msg *Message) error {
	if msg.Destination != "" {
		if msg.Destination == "org.freedesktop.DBus" {
			return WriteMessage(s.containerConn, msg)
		}
		if s.routeDestination(msg.Destination) == TargetHost && s.hostConn != nil {
			return s.sendToHost(msg, hostForward)
		}
		return WriteMessage(s.containerConn, msg)
	}

	// Broadcast: deliver on the container bus, and also on the host bus so
	// well-known names owned there (e.g. MPRIS) still emit to host listeners.
	if s.hostConn != nil && msg.Interface != "org.freedesktop.DBus" {
		_ = s.writeHostCopy(msg)
	}
	return WriteMessage(s.containerConn, msg)
}

func (s *ClientSession) handleDBusManagement(msg *Message) error {
	switch msg.Member {
	case "Hello":
		if s.hostConn != nil {
			s.noteHostPending(msg.Serial, hostSwallow, msg.Member)
			if err := s.writeHostCopy(msg); err != nil {
				s.mu.Lock()
				delete(s.hostPending, msg.Serial)
				s.mu.Unlock()
			}
		}
		return WriteMessage(s.containerConn, msg)

	case "AddMatch", "RemoveMatch":
		rule := extractFirstStringArg(msg)
		if MatchSignalRuleHost(rule) && s.hostConn != nil {
			s.noteHostPending(msg.Serial, hostSwallow, msg.Member)
			if err := s.writeHostCopy(msg); err != nil {
				s.mu.Lock()
				delete(s.hostPending, msg.Serial)
				s.mu.Unlock()
			}
		}
		return WriteMessage(s.containerConn, msg)

	case "RequestName", "ReleaseName":
		name := extractFirstStringArg(msg)
		if RouteOwnership(name) == TargetHost && s.hostConn != nil {
			return s.sendToHost(msg, hostForward)
		}
		return WriteMessage(s.containerConn, msg)

	case "GetNameOwner", "NameHasOwner", "StartServiceByName":
		name := extractFirstStringArg(msg)
		if s.routeDestination(name) == TargetHost && s.hostConn != nil {
			return s.sendToHost(msg, hostForward)
		}
		return WriteMessage(s.containerConn, msg)

	case "ListNames", "ListActivatableNames":
		if s.hostConn != nil {
			s.mu.Lock()
			s.merges[msg.Serial] = &mergeState{member: msg.Member}
			s.mu.Unlock()
			if err := s.writeHostCopy(msg); err != nil {
				s.mu.Lock()
				delete(s.merges, msg.Serial)
				s.mu.Unlock()
				return WriteMessage(s.containerConn, msg)
			}
		}
		return WriteMessage(s.containerConn, msg)

	case "GetConnectionUnixProcessID", "GetConnectionUnixUser",
		"GetConnectionCredentials", "GetAdtAuditSessionData",
		"GetConnectionSELinuxSecurityContext":
		name := extractFirstStringArg(msg)
		if s.routeDestination(name) == TargetHost && s.hostConn != nil {
			return s.sendToHost(msg, hostForward)
		}
		return WriteMessage(s.containerConn, msg)

	default:
		return WriteMessage(s.containerConn, msg)
	}
}

func (s *ClientSession) handleUpstream(from Target, msg *Message) error {
	if msg.Type == TypeMethodReturn || msg.Type == TypeError {
		if from == TargetContainer {
			if s.takeMerge(from, msg) {
				return nil
			}
			err := s.SendToClient(msg)
			msg.CloseFDs()
			return err
		}

		// Host replies: only deliver those we asked for.
		if s.takeMerge(from, msg) {
			return nil
		}

		s.mu.Lock()
		pending, ok := s.hostPending[msg.ReplySerial]
		if ok {
			delete(s.hostPending, msg.ReplySerial)
		}
		s.mu.Unlock()

		if !ok {
			msg.CloseFDs()
			return nil
		}

		if pending.kind == hostSwallow {
			if pending.member == "Hello" && msg.Type == TypeMethodReturn {
				name := extractFirstStringArg(msg)
				s.mu.Lock()
				s.hostUniqueName = name
				s.mu.Unlock()
				s.rememberHostUnique(name)
			}
			msg.CloseFDs()
			return nil
		}

		if pending.member == "GetNameOwner" && msg.Type == TypeMethodReturn {
			s.rememberHostUnique(extractFirstStringArg(msg))
		}

		err := s.SendToClient(msg)
		msg.CloseFDs()
		return err
	}

	if from == TargetHost && !s.shouldForwardHostSignal(msg) {
		msg.CloseFDs()
		return nil
	}

	err := s.SendToClient(msg)
	msg.CloseFDs()
	return err
}

func (s *ClientSession) takeMerge(from Target, msg *Message) bool {
	s.mu.Lock()
	state, ok := s.merges[msg.ReplySerial]
	if !ok {
		s.mu.Unlock()
		return false
	}
	if from == TargetHost {
		state.hostMsg = msg
	} else {
		state.contMsg = msg
	}
	ready := state.hostMsg != nil && state.contMsg != nil
	if ready {
		delete(s.merges, msg.ReplySerial)
	}
	s.mu.Unlock()

	if !ready {
		return true
	}

	merged := mergeNameListReplies(state.contMsg, state.hostMsg)
	state.hostMsg.CloseFDs()
	state.contMsg.CloseFDs()
	_ = s.SendToClient(merged)
	return true
}

func (s *ClientSession) shouldForwardHostSignal(msg *Message) bool {
	if msg.Type != TypeSignal {
		return false
	}

	if msg.Sender == "org.freedesktop.DBus" || msg.Interface == "org.freedesktop.DBus" {
		name := extractFirstStringArg(msg)
		if msg.Member != "NameOwnerChanged" && msg.Member != "NameAcquired" && msg.Member != "NameLost" {
			return false
		}
		if RouteDestination(name) != TargetHost && RouteOwnership(name) != TargetHost {
			return false
		}
		if msg.Member == "NameOwnerChanged" {
			stringsInBody := extractBodyStrings(msg)
			if len(stringsInBody) >= 3 {
				s.rememberHostUnique(stringsInBody[2])
			}
		}
		return true
	}

	return true
}

func mergeNameListReplies(containerMsg, hostMsg *Message) *Message {
	base := containerMsg
	if containerMsg.Type != TypeMethodReturn {
		if hostMsg.Type == TypeMethodReturn {
			base = hostMsg
		} else {
			return containerMsg
		}
	}

	var names []string
	if containerMsg.Type == TypeMethodReturn {
		if parsed, err := ParseStringArray(containerMsg); err == nil {
			names = append(names, parsed...)
		}
	}
	if hostMsg.Type == TypeMethodReturn {
		if parsed, err := ParseStringArray(hostMsg); err == nil {
			names = append(names, parsed...)
		}
	}

	return ReplaceBody(base, EncodeStringArray(base.byteOrder(), uniqueStrings(names)))
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
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

func extractFirstStringArg(msg *Message) string {
	stringsInBody := extractBodyStrings(msg)
	if len(stringsInBody) == 0 {
		return ""
	}
	return stringsInBody[0]
}

func extractBodyStrings(msg *Message) []string {
	if msg == nil || int(msg.HeaderLength)+4 > len(msg.Raw) {
		return nil
	}

	body := msg.Raw[msg.HeaderLength:]
	order := msg.byteOrder()
	var out []string
	pos := 0
	for pos+4 <= len(body) {
		pos = align(pos, 4)
		if pos+4 > len(body) {
			break
		}
		strLen := int(order.Uint32(body[pos : pos+4]))
		pos += 4
		if strLen < 0 || pos+strLen+1 > len(body) {
			break
		}
		out = append(out, string(body[pos:pos+strLen]))
		pos += strLen + 1
	}
	return out
}

// NewErrorMessage constructs a D-Bus Error response message
func NewErrorMessage(replyTo *Message, errorName string, errorText string, serial uint32) *Message {

	bodyBytes := make([]byte, 4+len(errorText)+1)
	binary.LittleEndian.PutUint32(bodyBytes[:4], uint32(len(errorText)))
	copy(bodyBytes[4:], errorText)
	bodyBytes[4+len(errorText)] = 0
	bodyLen := uint32(len(bodyBytes))

	var fields []byte
	fields = appendFieldString(fields, HeaderErrorName, errorName)
	fields = appendFieldUint32(fields, HeaderReplySerial, replyTo.Serial)
	if replyTo.Sender != "" {
		fields = appendFieldString(fields, HeaderDestination, replyTo.Sender)
	}
	fields = appendFieldSignature(fields, "s")

	headerFieldsLen := uint32(len(fields))
	paddedHeaderLen := 16 + headerFieldsLen
	if rem := paddedHeaderLen % 8; rem != 0 {
		paddedHeaderLen += (8 - rem)
	}

	header := make([]byte, paddedHeaderLen)
	header[0] = 'l'
	header[1] = TypeError
	header[2] = 0x01
	header[3] = 1
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

	offset := 16 + len(buf)
	if rem := offset % 8; rem != 0 {
		pad := 8 - rem
		for i := 0; i < pad; i++ {
			buf = append(buf, 0)
		}
	}

	buf = append(buf, code)
	buf = append(buf, 1, 's', 0)

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
	buf = append(buf, 1, 'u', 0)

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
	buf = append(buf, 1, 'g', 0)
	buf = append(buf, byte(len(sig)))
	buf = append(buf, sig...)
	buf = append(buf, 0)

	return buf
}
