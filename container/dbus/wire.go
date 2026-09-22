package dbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"syscall"
)

const (
	TypeMethodCall   byte = 1
	TypeMethodReturn byte = 2
	TypeError        byte = 3
	TypeSignal       byte = 4

	HeaderPath        byte = 1
	HeaderInterface   byte = 2
	HeaderMember      byte = 3
	HeaderErrorName   byte = 4
	HeaderReplySerial byte = 5
	HeaderDestination byte = 6
	HeaderSender      byte = 7
	HeaderSignature   byte = 8
	HeaderUnixFDs     byte = 9

	MaxMessageSize = 134217728 // 128 MB (standard D-Bus limit)
)

// Message represents a framed D-Bus message
type Message struct {
	Endianness   byte
	Type         byte
	Flags        byte
	Version      byte
	BodyLength   uint32
	Serial       uint32
	HeaderLength uint32
	TotalLength  uint32

	// Parsed header fields
	Path        string
	Interface   string
	Member      string
	ErrorName   string
	ReplySerial uint32
	Destination string
	Sender      string
	UnixFDs     uint32

	// Raw message bytes (header + body)
	Raw []byte

	// Associated Unix file descriptors passed with this message
	FDs []int
}

// Close closes any file descriptors held by this message
func (m *Message) CloseFDs() {
	for _, fd := range m.FDs {
		if fd >= 0 {
			syscall.Close(fd)
		}
	}
	m.FDs = nil
}

// MessageReader reads D-Bus messages from a Unix connection
type MessageReader struct {
	conn       *net.UnixConn
	buf        []byte
	pendingFDs []int
}

// NewMessageReader creates a new reader for a Unix socket connection
func NewMessageReader(conn *net.UnixConn) *MessageReader {
	return &MessageReader{
		conn:       conn,
		buf:        make([]byte, 0, 4096),
		pendingFDs: make([]int, 0),
	}
}

// ReadMessage reads the next complete D-Bus message
func (r *MessageReader) ReadMessage() (*Message, error) {

	// Reuse read buffers
	tmp := make([]byte, 65536)
	oob := make([]byte, 4096)
	readMore := func() error {
		for {
			n, oobn, _, _, err := r.conn.ReadMsgUnix(tmp, oob)
			if oobn > 0 {
				fds, parseErr := parseFDs(oob[:oobn])
				if parseErr == nil && len(fds) > 0 {
					r.pendingFDs = append(r.pendingFDs, fds...)
				}
			}
			if n > 0 {
				r.buf = append(r.buf, tmp[:n]...)
				return nil
			}
			if err != nil {
				return err
			}
			// n==0, oobn==0, no error: should not happen on blocking Unix socket
			// but guard against infinite loop
		}
	}

	// Read until we have at least the 16-byte fixed header
	for len(r.buf) < 16 {
		if err := readMore(); err != nil {
			return nil, err
		}
	}

	endian := r.buf[0]
	var order binary.ByteOrder
	switch endian {
	case 'l':
		order = binary.LittleEndian
	case 'B':
		order = binary.BigEndian
	default:
		return nil, fmt.Errorf("invalid dbus endianness: %c", endian)
	}

	bodyLen := order.Uint32(r.buf[4:8])
	headerFieldsLen := order.Uint32(r.buf[12:16])

	// Structs in header array are padded to 8-byte boundary
	paddedHeaderLen := 16 + headerFieldsLen
	if rem := paddedHeaderLen % 8; rem != 0 {
		paddedHeaderLen += (8 - rem)
	}

	totalLen := paddedHeaderLen + bodyLen
	if totalLen > MaxMessageSize {
		return nil, fmt.Errorf("message size exceeds limit: %d", totalLen)
	}

	// Read until we have the full message
	for len(r.buf) < int(totalLen) {
		if err := readMore(); err != nil {
			return nil, err
		}
	}

	raw := make([]byte, totalLen)
	copy(raw, r.buf[:totalLen])
	r.buf = r.buf[totalLen:]

	msg, err := ParseHeader(raw)
	if err != nil {
		return nil, err
	}

	// Attach file descriptors if present
	if msg.UnixFDs > 0 && len(r.pendingFDs) >= int(msg.UnixFDs) {
		msg.FDs = append([]int(nil), r.pendingFDs[:msg.UnixFDs]...)
		r.pendingFDs = r.pendingFDs[msg.UnixFDs:]
	}

	return msg, nil
}

// ParseHeader parses D-Bus message header fields from raw bytes
func ParseHeader(raw []byte) (*Message, error) {

	if len(raw) < 16 {
		return nil, errors.New("message too short for header")
	}

	endian := raw[0]
	var order binary.ByteOrder
	switch endian {
	case 'l':
		order = binary.LittleEndian
	case 'B':
		order = binary.BigEndian
	default:
		return nil, fmt.Errorf("invalid endianness: %c", endian)
	}

	msgType := raw[1]
	flags := raw[2]
	version := raw[3]
	bodyLen := order.Uint32(raw[4:8])
	serial := order.Uint32(raw[8:12])
	headerFieldsLen := order.Uint32(raw[12:16])

	paddedHeaderLen := 16 + headerFieldsLen
	if rem := paddedHeaderLen % 8; rem != 0 {
		paddedHeaderLen += (8 - rem)
	}

	msg := &Message{
		Endianness:   endian,
		Type:         msgType,
		Flags:        flags,
		Version:      version,
		BodyLength:   bodyLen,
		Serial:       serial,
		HeaderLength: paddedHeaderLen,
		TotalLength:  paddedHeaderLen + bodyLen,
		Raw:          raw,
	}

	// Parse header fields array `a(yv)`
	pos := 16
	end := 16 + int(headerFieldsLen)
	if end > len(raw) {
		return nil, errors.New("header fields length exceeds message size")
	}

	for pos < end {
		// Align struct to 8 bytes
		pos = align(pos, 8)
		if pos >= end {
			break
		}

		code := raw[pos]
		pos++
		if pos >= end {
			break
		}

		// Variant signature length
		sigLen := int(raw[pos])
		pos++
		if pos+sigLen >= end {
			break
		}

		sig := string(raw[pos : pos+sigLen])
		pos += sigLen + 1 // skip signature and null byte

		// Value decoding based on type
		switch sig {
		case "s", "o": // String or Object Path
			pos = align(pos, 4)
			if pos+4 > end {
				break
			}

			strLen := int(order.Uint32(raw[pos : pos+4]))
			pos += 4
			if pos+strLen > len(raw) {
				break
			}

			strVal := string(raw[pos : pos+strLen])
			pos += strLen + 1 // skip string and null byte

			switch code {
			case HeaderPath:
				msg.Path = strVal
			case HeaderInterface:
				msg.Interface = strVal
			case HeaderMember:
				msg.Member = strVal
			case HeaderErrorName:
				msg.ErrorName = strVal
			case HeaderDestination:
				msg.Destination = strVal
			case HeaderSender:
				msg.Sender = strVal
			}

		case "u": // UINT32
			pos = align(pos, 4)
			if pos+4 > len(raw) {
				break
			}

			uVal := order.Uint32(raw[pos : pos+4])
			pos += 4

			switch code {
			case HeaderReplySerial:
				msg.ReplySerial = uVal
			case HeaderUnixFDs:
				msg.UnixFDs = uVal
			}

		case "g": // Signature
			if pos >= end {
				break
			}

			sLen := int(raw[pos])
			pos++
			pos += sLen + 1

		default:
			// Unknown or unsupported variant; skip to end
			pos = end
		}
	}

	return msg, nil
}

// WriteMessage writes a message with any file descriptors to a Unix socket
func WriteMessage(conn *net.UnixConn, msg *Message) error {

	if len(msg.FDs) > 0 {
		oob := syscall.UnixRights(msg.FDs...)
		n, oobn, err := conn.WriteMsgUnix(msg.Raw, oob, nil)
		if err != nil {
			return err
		}
		// Write remaining bytes if not all written in the first call
		if n < len(msg.Raw) {
			_, err = conn.Write(msg.Raw[n:])
			if err != nil {
				return err
			}
		}
		_ = oobn
		return nil
	}

	_, err := conn.Write(msg.Raw)
	return err
}

// Helper: align offset to n bytes
func align(offset int, n int) int {
	if rem := offset % n; rem != 0 {
		return offset + (n - rem)
	}

	return offset
}

// Helper: parse FDs from socket control message
func parseFDs(oob []byte) ([]int, error) {

	scms, err := syscall.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, err
	}

	var fds []int
	for _, scm := range scms {
		scmFDs, err := syscall.ParseUnixRights(&scm)
		if err == nil {
			fds = append(fds, scmFDs...)
		}
	}

	return fds, nil
}

// CopyFDs creates duplicates of the given file descriptors
func CopyFDs(fds []int) ([]int, error) {

	if len(fds) == 0 {
		return nil, nil
	}

	dups := make([]int, len(fds))
	for i, fd := range fds {
		dup, err := syscall.Dup(fd)
		if err != nil {
			// Close already duplicated ones on failure
			for j := 0; j < i; j++ {
				syscall.Close(dups[j])
			}
			return nil, err
		}
		dups[i] = dup
	}

	return dups, nil
}

// DrainFDs ensures any remaining pending file descriptors in reader are closed
func (r *MessageReader) Close() {
	for _, fd := range r.pendingFDs {
		syscall.Close(fd)
	}

	r.pendingFDs = nil
}
