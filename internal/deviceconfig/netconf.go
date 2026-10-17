package deviceconfig

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// NETCONF over SSH (RFC 6241, RFC 6242). Both peers send a <hello> framed by
// the end-of-message marker; if both advertise base:1.1, every later message
// uses chunked framing, otherwise the marker framing continues.
const (
	netconfNS          = "urn:ietf:params:xml:ns:netconf:base:1.0"
	netconfBase10      = "urn:ietf:params:netconf:base:1.0"
	netconfBase11      = "urn:ietf:params:netconf:base:1.1"
	netconfEOM         = "]]>]]>"
	netconfClientHello = `<?xml version="1.0" encoding="UTF-8"?>` +
		`<hello xmlns="` + netconfNS + `"><capabilities>` +
		`<capability>` + netconfBase10 + `</capability>` +
		`<capability>` + netconfBase11 + `</capability>` +
		`</capabilities></hello>`
	netconfGetConfig = `<rpc message-id="1" xmlns="` + netconfNS + `">` +
		`<get-config><source><running/></source></get-config></rpc>`
	netconfCloseSession = `<rpc message-id="2" xmlns="` + netconfNS + `"><close-session/></rpc>`
	// maxChunkHeaderDigits is the width of the largest chunk size RFC 6242
	// allows, 4294967295.
	maxChunkHeaderDigits = 10
)

var errNetconfFraming = errors.New("malformed NETCONF framing")

// getRunningConfig reads the running datastore over the session's netconf
// subsystem and returns the contents of the reply's <data> element.
func getRunningConfig(session *ssh.Session) (string, error) {
	stdin, err := session.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrCommand, err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrCommand, err)
	}
	if err = session.RequestSubsystem("netconf"); err != nil {
		return "", fmt.Errorf("%w: the device refused the netconf subsystem (is NETCONF enabled?): %w",
			ErrCommand, err)
	}

	nc := &netconfSession{r: bufio.NewReader(stdout), w: stdin}
	if err = nc.hello(); err != nil {
		return "", err
	}
	if err = nc.send(netconfGetConfig); err != nil {
		return "", fmt.Errorf("%w: send get-config: %w", ErrCommand, err)
	}
	msg, err := nc.receive()
	if err != nil {
		return "", fmt.Errorf("%w: read get-config reply: %w", ErrCommand, err)
	}
	config, err := parseGetConfigReply(msg)
	if err != nil {
		return "", err
	}
	// The session ends when the channel closes either way; close-session
	// only spares the device logging an abnormal termination. Its reply is
	// not awaited: a device that never sends one would hold the backup until
	// the device deadline and turn it into a failure.
	_ = nc.send(netconfCloseSession)
	return config, nil
}

type netconfSession struct {
	r       *bufio.Reader
	w       io.Writer
	chunked bool
	// capabilities is what the device's hello advertised, each without its
	// query parameters.
	capabilities map[string]bool
}

// hello exchanges capabilities and settles the framing for the rest of the
// session.
func (nc *netconfSession) hello() error {
	if err := nc.send(netconfClientHello); err != nil {
		return fmt.Errorf("%w: send hello: %w", ErrCommand, err)
	}
	msg, err := nc.receive()
	if err != nil {
		return fmt.Errorf("%w: read hello: %w", ErrCommand, err)
	}
	var hello struct {
		XMLName      xml.Name `xml:"hello"`
		Capabilities []string `xml:"capabilities>capability"`
	}
	if err = xml.Unmarshal(msg, &hello); err != nil {
		return fmt.Errorf("%w: the device's hello is not NETCONF: %w", ErrCommand, err)
	}
	nc.capabilities = make(map[string]bool, len(hello.Capabilities))
	for _, c := range hello.Capabilities {
		uri, _, _ := strings.Cut(strings.TrimSpace(c), "?")
		nc.capabilities[uri] = true
	}
	nc.chunked = nc.capabilities[netconfBase11]
	if !nc.capabilities[netconfBase10] && !nc.chunked {
		return fmt.Errorf("%w: the device advertises no NETCONF base capability", ErrCommand)
	}
	return nil
}

func (nc *netconfSession) send(msg string) error {
	var err error
	if nc.chunked {
		_, err = fmt.Fprintf(nc.w, "\n#%d\n%s\n##\n", len(msg), msg)
	} else {
		_, err = io.WriteString(nc.w, msg+netconfEOM)
	}
	return err
}

// receive reads one message, refusing one larger than maxConfigBytes.
func (nc *netconfSession) receive() ([]byte, error) {
	if nc.chunked {
		return readChunked(nc.r)
	}
	return readEOM(nc.r)
}

func readEOM(r *bufio.Reader) ([]byte, error) {
	var msg []byte
	for {
		part, err := r.ReadSlice('>')
		msg = append(msg, part...)
		if len(msg) > maxConfigBytes+len(netconfEOM) {
			return nil, fmt.Errorf("message exceeds %d bytes", maxConfigBytes)
		}
		if err != nil && !errors.Is(err, bufio.ErrBufferFull) {
			return nil, err
		}
		if bytes.HasSuffix(msg, []byte(netconfEOM)) {
			return msg[:len(msg)-len(netconfEOM)], nil
		}
	}
}

// readChunked reads one chunked message: one or more "\n#<size>\n<data>"
// chunks, then "\n##\n".
func readChunked(r *bufio.Reader) ([]byte, error) {
	var msg []byte
	for {
		if err := skipToChunk(r); err != nil {
			return nil, err
		}
		size, err := readChunkHeader(r)
		if err != nil {
			return nil, err
		}
		if size == 0 {
			if len(msg) == 0 {
				return nil, fmt.Errorf("%w: empty message", errNetconfFraming)
			}
			return msg, nil
		}
		if len(msg)+size > maxConfigBytes {
			return nil, fmt.Errorf("message exceeds %d bytes", maxConfigBytes)
		}
		start := len(msg)
		msg = append(msg, make([]byte, size)...)
		if _, err = io.ReadFull(r, msg[start:]); err != nil {
			return nil, err
		}
	}
}

// readChunkHeader reads what follows "\n#": a chunk size and "\n", or the
// "#\n" that ends the message, for which it returns 0.
func readChunkHeader(r *bufio.Reader) (int, error) {
	var digits []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		switch {
		case b == '#' && len(digits) == 0:
			return 0, expect(r, "\n")
		case b == '\n' && len(digits) > 0:
			n, parseErr := strconv.Atoi(string(digits))
			if parseErr != nil || n == 0 || digits[0] == '0' {
				return 0, fmt.Errorf("%w: chunk size %q", errNetconfFraming, digits)
			}
			return n, nil
		case b >= '0' && b <= '9' && len(digits) < maxChunkHeaderDigits:
			digits = append(digits, b)
		default:
			return 0, fmt.Errorf("%w: unexpected byte %q in a chunk header", errNetconfFraming, b)
		}
	}
}

// skipToChunk consumes the "\n#" that opens a chunk. It accepts any run of
// whitespace before the '#': some devices end their marker-framed hello with
// a newline, which then precedes the first chunked message.
func skipToChunk(r *bufio.Reader) error {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		switch b {
		case '#':
			return nil
		case '\n', '\r', ' ', '\t':
		default:
			return fmt.Errorf("%w: unexpected byte %q before a chunk", errNetconfFraming, b)
		}
	}
}

func expect(r *bufio.Reader, want string) error {
	for i := range len(want) {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		if b != want[i] {
			return fmt.Errorf("%w: got %q, want %q", errNetconfFraming, b, want[i])
		}
	}
	return nil
}

// rpcReply is an <rpc-reply>: errors, and the <data> of a retrieval.
type rpcReply struct {
	XMLName xml.Name `xml:"rpc-reply"`
	Errors  []struct {
		Severity string `xml:"error-severity"`
		Tag      string `xml:"error-tag"`
		Message  string `xml:"error-message"`
	} `xml:"rpc-error"`
	Data *struct {
		Inner string `xml:",innerxml"`
	} `xml:"data"`
}

// parseReply decodes an operation's reply and returns the device's error, if
// it sent one. Warnings do not fail an operation.
func parseReply(msg []byte, op string) (rpcReply, error) {
	var reply rpcReply
	if err := xml.Unmarshal(msg, &reply); err != nil {
		return rpcReply{}, fmt.Errorf("%w: the %s reply is not an rpc-reply: %w", ErrCommand, op, err)
	}
	for _, e := range reply.Errors {
		if strings.TrimSpace(e.Severity) == "warning" {
			continue
		}
		detail := strings.TrimSpace(e.Message)
		if detail == "" {
			detail = strings.TrimSpace(e.Tag)
		}
		return rpcReply{}, fmt.Errorf("%w: %s: %s", ErrCommand, op, detail)
	}
	return reply, nil
}

// parseGetConfigReply returns the contents of a <get-config> reply's <data>,
// or the device's error.
func parseGetConfigReply(msg []byte) (string, error) {
	reply, err := parseReply(msg, "get-config")
	if err != nil {
		return "", err
	}
	if reply.Data == nil {
		return "", fmt.Errorf("%w: the get-config reply carries no data", ErrCommand)
	}
	return strings.TrimSpace(reply.Data.Inner) + "\n", nil
}
