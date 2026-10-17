package deviceconfig

// The NETCONF dry run (ADR-0035): the candidate is loaded into the device's
// candidate datastore under a lock, validated, read back and discarded. The
// diff is between what the device holds as running and what it would hold as
// candidate, both rendered by the device, so the device's own defaults and
// ordering are in it rather than Seed's guess at them.

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Capabilities a push depends on (RFC 6241 §8.3, §8.4, §8.6). Apply commits
// with a confirmed commit that a new session confirms, which needs 1.1.
const (
	netconfCapCandidate       = "urn:ietf:params:netconf:capability:candidate:1.0"
	netconfCapConfirmedCommit = "urn:ietf:params:netconf:capability:confirmed-commit:1.1"
	netconfCapValidate10      = "urn:ietf:params:netconf:capability:validate:1.0"
	netconfCapValidate11      = "urn:ietf:params:netconf:capability:validate:1.1"
)

// netconfDryRun previews candidate on the session's netconf subsystem.
func netconfDryRun(session *ssh.Session, mode PushMode, candidate string) (DryRunResult, error) {
	stdin, err := session.StdinPipe()
	if err != nil {
		return DryRunResult{}, fmt.Errorf("%w: %w", ErrCommand, err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return DryRunResult{}, fmt.Errorf("%w: %w", ErrCommand, err)
	}
	if err = session.RequestSubsystem("netconf"); err != nil {
		return DryRunResult{}, fmt.Errorf("%w: the device refused the netconf subsystem (is NETCONF enabled?): %w",
			ErrCommand, err)
	}

	nc := &netconfSession{r: bufio.NewReader(stdout), w: stdin}
	if err = nc.hello(); err != nil {
		return DryRunResult{}, err
	}
	for _, c := range []string{netconfCapCandidate, netconfCapConfirmedCommit} {
		if !nc.capabilities[c] {
			return DryRunResult{}, &PlanRefusedError{
				Reason: "the NETCONF server does not advertise " + c + ", which a push with automatic revert needs",
			}
		}
	}
	rpc := &netconfRPC{nc: nc}
	// A device does not grant the candidate lock while another session has
	// uncommitted changes in it, so the lock is also what keeps a plan from
	// discarding someone else's work.
	if _, err = rpc.call("lock", `<lock><target><candidate/></target></lock>`); err != nil {
		return DryRunResult{}, err
	}
	res, err := previewCandidate(rpc, mode, candidate)
	_, discardErr := rpc.call("discard-changes", `<discard-changes/>`)
	_, unlockErr := rpc.call("unlock", `<unlock><target><candidate/></target></unlock>`)
	// The reply is not awaited, as in a backup.
	_ = rpc.send("close-session", `<close-session/>`)
	if err != nil {
		return DryRunResult{}, err
	}
	if err = errors.Join(discardErr, unlockErr); err != nil {
		return DryRunResult{}, err
	}
	return res, nil
}

func previewCandidate(rpc *netconfRPC, mode PushMode, candidate string) (DryRunResult, error) {
	running, err := rpc.getConfig("running")
	if err != nil {
		return DryRunResult{}, err
	}
	if _, err = rpc.call("edit-config", `<edit-config><target><candidate/></target>`+
		`<default-operation>`+string(mode)+`</default-operation>`+
		`<config>`+candidate+`</config></edit-config>`); err != nil {
		return DryRunResult{}, err
	}
	if rpc.nc.capabilities[netconfCapValidate10] || rpc.nc.capabilities[netconfCapValidate11] {
		if _, err = rpc.call("validate", `<validate><source><candidate/></source></validate>`); err != nil {
			return DryRunResult{}, err
		}
	}
	after, err := rpc.getConfig("candidate")
	if err != nil {
		return DryRunResult{}, err
	}
	return lineDiff("running", indentXML(running), "candidate", indentXML(after))
}

// netconfRPC numbers the operations of one session.
type netconfRPC struct {
	nc     *netconfSession
	nextID int
}

func (r *netconfRPC) send(op, body string) error {
	r.nextID++
	msg := `<rpc message-id="` + strconv.Itoa(r.nextID) + `" xmlns="` + netconfNS + `">` + body + `</rpc>`
	if err := r.nc.send(msg); err != nil {
		return fmt.Errorf("%w: send %s: %w", ErrCommand, op, err)
	}
	return nil
}

func (r *netconfRPC) call(op, body string) (rpcReply, error) {
	if err := r.send(op, body); err != nil {
		return rpcReply{}, err
	}
	msg, err := r.nc.receive()
	if err != nil {
		return rpcReply{}, fmt.Errorf("%w: read %s reply: %w", ErrCommand, op, err)
	}
	return parseReply(msg, op)
}

func (r *netconfRPC) getConfig(datastore string) (string, error) {
	reply, err := r.call("get-config", `<get-config><source><`+datastore+`/></source></get-config>`)
	if err != nil {
		return "", err
	}
	if reply.Data == nil {
		return "", fmt.Errorf("%w: the get-config reply carries no data", ErrCommand)
	}
	return reply.Data.Inner, nil
}

// checkXMLCandidate accepts a NETCONF candidate: one or more well-formed
// elements and nothing else at the top level. It goes inside <config> in an
// <edit-config>, so anything else could close that element and send the
// device an operation of the caller's choosing.
func checkXMLCandidate(candidate string) error {
	if strings.Contains(candidate, netconfEOM) {
		return errors.New("candidate contains the NETCONF end-of-message marker")
	}
	d := xml.NewDecoder(strings.NewReader(candidate))
	depth, elements := 0, 0
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("candidate is not well-formed XML: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 0 {
				elements++
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(t)) != "" {
				return errors.New("candidate has text outside its elements")
			}
		case xml.ProcInst, xml.Directive:
			return errors.New("candidate must hold configuration elements only, without an XML declaration or DTD")
		case xml.Comment:
		}
	}
	if elements == 0 {
		return errors.New("candidate holds no configuration element")
	}
	return nil
}

// indentXML renders an element sequence one element per line, so a diff of
// two configurations is a diff of their leaves however the device spaced
// them. Prefixes are kept as written. Text that is not XML is returned as it
// came.
func indentXML(s string) string {
	d := xml.NewDecoder(strings.NewReader(s))
	var x xmlIndenter
	for {
		tok, err := d.RawToken()
		if errors.Is(err, io.EOF) {
			return x.b.String()
		}
		if err != nil {
			return s
		}
		switch t := tok.(type) {
		case xml.StartElement:
			x.start(t)
		case xml.CharData:
			x.text(strings.TrimSpace(string(t)))
		case xml.EndElement:
			x.end(t)
		}
	}
}

// xmlIndenter holds back each start tag until it knows whether the element is
// empty, a leaf with text, or a parent, so a leaf renders on one line.
type xmlIndenter struct {
	b       strings.Builder
	depth   int
	pending *xml.StartElement
	leaf    string
}

func (x *xmlIndenter) line(indent int, parts ...string) {
	x.b.WriteString(strings.Repeat("  ", indent))
	for _, p := range parts {
		x.b.WriteString(p)
	}
	x.b.WriteByte('\n')
}

// flush writes a held start tag, and its text, as the opening of a parent.
func (x *xmlIndenter) flush() {
	if x.pending == nil {
		return
	}
	x.line(x.depth-1, "<", xmlStartTag(*x.pending), ">")
	if x.leaf != "" {
		x.line(x.depth, escapeXML(x.leaf))
	}
	x.pending, x.leaf = nil, ""
}

func (x *xmlIndenter) start(t xml.StartElement) {
	x.flush()
	start := t.Copy()
	x.pending = &start
	x.depth++
}

func (x *xmlIndenter) text(text string) {
	switch {
	case text == "":
	case x.pending != nil && x.leaf == "":
		x.leaf = text
	default:
		x.flush()
		x.line(x.depth, escapeXML(text))
	}
}

func (x *xmlIndenter) end(t xml.EndElement) {
	x.depth--
	switch {
	case x.pending == nil:
		x.line(x.depth, "</", xmlName(t.Name), ">")
	case x.leaf == "":
		x.line(x.depth, "<", xmlStartTag(*x.pending), "/>")
	default:
		x.line(x.depth, "<", xmlStartTag(*x.pending), ">", escapeXML(x.leaf), "</", xmlName(t.Name), ">")
	}
	x.pending, x.leaf = nil, ""
}

func xmlStartTag(e xml.StartElement) string {
	var b strings.Builder
	b.WriteString(xmlName(e.Name))
	for _, a := range e.Attr {
		b.WriteString(" " + xmlName(a.Name) + `="` + escapeXML(a.Value) + `"`)
	}
	return b.String()
}

func xmlName(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}

func escapeXML(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
