package dbus

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
)

const serverGUID = "10000000000000000000000000000000"

const (
	saslRejected      = "REJECTED EXTERNAL\r\n"
	saslOK            = "OK " + serverGUID + "\r\n"
	saslDataChallenge = "DATA\r\n"
	saslAgreeUnixFD   = "AGREE_UNIX_FD\r\n"
)

func writeSASL(conn *net.UnixConn, line string) error {
	_, err := conn.Write([]byte(line))
	return err
}

// AuthenticateClient performs the SASL handshake with an incoming client connection.
// It returns any unconsumed bytes read from the client that belong to the binary stream.
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

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read sasl line error: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")
		parts := strings.Fields(line)
		if len(parts) == 0 {
			if err := writeSASL(conn, saslRejected); err != nil {
				return nil, err
			}
			continue
		}

		switch parts[0] {
		case "AUTH":
			// Bare AUTH is a probe: advertise supported mechanisms.
			// GDBus (gdbus, gio) always starts this way and expects
			// "REJECTED EXTERNAL", not OK.
			if len(parts) == 1 {
				if err := writeSASL(conn, saslRejected); err != nil {
					return nil, err
				}
				continue
			}

			if parts[1] != "EXTERNAL" {
				if err := writeSASL(conn, saslRejected); err != nil {
					return nil, err
				}
				continue
			}

			// AUTH EXTERNAL <identity>: accept immediately.
			// AUTH EXTERNAL with no initial response: send an empty
			// DATA challenge so the client can reply with DATA.
			if len(parts) >= 3 {
				if err := writeSASL(conn, saslOK); err != nil {
					return nil, err
				}
			} else {
				if err := writeSASL(conn, saslDataChallenge); err != nil {
					return nil, err
				}
			}

		case "DATA":
			if err := writeSASL(conn, saslOK); err != nil {
				return nil, err
			}

		case "NEGOTIATE_UNIX_FD":
			if err := writeSASL(conn, saslAgreeUnixFD); err != nil {
				return nil, err
			}

		case "BEGIN":
			var extraBytes []byte
			buffered := reader.Buffered()
			if buffered > 0 {
				extraBytes = make([]byte, buffered)
				_, err = reader.Read(extraBytes)
				if err != nil {
					return nil, err
				}
			}
			return extraBytes, nil

		case "CANCEL", "ERROR":
			if err := writeSASL(conn, saslRejected); err != nil {
				return nil, err
			}

		default:
			if err := writeSASL(conn, saslRejected); err != nil {
				return nil, err
			}
		}
	}
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

	// Some older D-Bus daemon versions may not agree
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
