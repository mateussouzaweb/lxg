package dbus

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
)

// AuthenticateClient performs the SASL handshake with an incoming client connection
// It returns any unconsumed bytes read from the client that belong to the binary stream
func AuthenticateClient(conn *net.UnixConn) ([]byte, error) {

	// First byte in D-Bus Unix domain socket must be a null byte \0
	firstByte := make([]byte, 1)
	_, err := conn.Read(firstByte)
	if err != nil {
		return nil, fmt.Errorf("read initial null byte error: %w", err)
	}
	if firstByte[0] != 0 {
		return nil, fmt.Errorf("expected null byte, got: %x", firstByte[0])
	}

	reader := bufio.NewReader(conn)
	var extraBytes []byte

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read sasl line error: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")

		if strings.HasPrefix(line, "AUTH") {
			// Accept any AUTH (typically AUTH EXTERNAL)
			guid := "10000000000000000000000000000000"
			_, err = conn.Write(fmt.Appendf(nil, "OK %s\r\n", guid))
			if err != nil {
				return nil, err
			}

		} else if strings.HasPrefix(line, "NEGOTIATE_UNIX_FD") {
			// Agree to FD passing
			_, err = conn.Write([]byte("AGREE_UNIX_FD\r\n"))
			if err != nil {
				return nil, err
			}

		} else if strings.HasPrefix(line, "BEGIN") {
			// Handshake is finished! Check if there are buffered bytes remaining
			buffered := reader.Buffered()
			if buffered > 0 {
				extraBytes = make([]byte, buffered)
				_, err = reader.Read(extraBytes)
				if err != nil {
					return nil, err
				}
			}

			break
		} else {
			// Reject unknown commands
			_, err = conn.Write([]byte("REJECTED EXTERNAL\r\n"))
			if err != nil {
				return nil, err
			}
		}
	}

	return extraBytes, nil
}

// AuthenticateUpstream performs the SASL handshake as a client to an upstream D-Bus daemon
func AuthenticateUpstream(conn *net.UnixConn) error {
	uid := os.Getuid()
	uidHex := hex.EncodeToString(fmt.Appendf(nil, "%d", uid))

	// Send initial null byte and AUTH command
	authCmd := fmt.Sprintf("\x00AUTH EXTERNAL %s\r\n", uidHex)
	_, err := conn.Write([]byte(authCmd))
	if err != nil {
		return fmt.Errorf("send auth error: %w", err)
	}

	reader := bufio.NewReader(conn)

	// Read response: expect OK
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read auth response error: %w", err)
	}
	if !strings.HasPrefix(response, "OK") {
		return fmt.Errorf("auth rejected by upstream: %s", strings.TrimSpace(response))
	}

	// Negotiate Unix FD passing
	_, err = conn.Write([]byte("NEGOTIATE_UNIX_FD\r\n"))
	if err != nil {
		return fmt.Errorf("send negotiate unix fd error: %w", err)
	}

	response, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read negotiate response error: %w", err)
	}

	// Some older dbus-daemon versions may not agree
	// non-fatal, FD passing will simply not work
	// The router continues without FD forwarding on this upstream if not agreed
	// Send BEGIN
	_, err = conn.Write([]byte("BEGIN\r\n"))
	if err != nil {
		return fmt.Errorf("send begin error: %w", err)
	}

	_ = response
	return nil
}
