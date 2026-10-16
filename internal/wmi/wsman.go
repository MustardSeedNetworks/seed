// Package wmi reads Windows inventory from WMI over WS-Management (P-D11,
// ADR-0034): WQL SELECT queries in root\cimv2, sent as WS-Enumeration
// Enumerate and Pull requests to the WinRM HTTPS listener on TCP 5986.
//
// The listener's certificate is never verified against a CA and never trusted
// on first use. A target with no pinned certificate is contacted only for the
// TLS handshake, which stops with a *CertUnconfirmedError naming the
// fingerprint the host presented, so no HTTP request and no credential leaves
// Seed until an operator confirms it. A pinned target presenting another
// certificate is refused with ErrCertMismatch.
//
// Authentication is NTLMv2 inside TLS. Basic is never offered, even when the
// host asks for it, and redirects are not followed.
package wmi

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	ntlmssp "github.com/Azure/go-ntlmssp"
	"github.com/google/uuid"
)

const (
	// maxEnvelopeBytes is the MaxEnvelopeSize Seed asks for, WinRM's own
	// default. The host splits an enumeration into Pull responses that fit.
	maxEnvelopeBytes = 512000
	// maxResponseBytes caps what is read from one response: the envelope
	// limit plus room for a fault body, which the host does not size.
	maxResponseBytes = 1 << 20
	// maxInstances stops an enumeration that never ends. The largest class
	// read is installed software, a few hundred entries on a busy server.
	maxInstances = 10000
	// operationTimeout is how long the host may spend on one request.
	operationTimeout = "PT60S"
	// maxElements is how many instances Seed asks for per response; the host
	// returns fewer when they would not fit in the envelope.
	maxElements = 100

	soapContentType = "application/soap+xml;charset=UTF-8"
	actionEnumerate = "http://schemas.xmlsoap.org/ws/2004/09/enumeration/Enumerate"
	actionPull      = "http://schemas.xmlsoap.org/ws/2004/09/enumeration/Pull"
	// resourceCIMv2 with a WQL filter selects the class in the query.
	resourceCIMv2 = "http://schemas.microsoft.com/wbem/wsman/1/wmi/root/cimv2/*"
	xsiNamespace  = "http://www.w3.org/2001/XMLSchema-instance"
)

// Errors a query returns. A *CertUnconfirmedError is returned for a target
// with no pinned certificate.
var (
	ErrAuth         = errors.New("authentication failed")
	ErrCertMismatch = errors.New("certificate does not match the pinned fingerprint")
	ErrUnreachable  = errors.New("host unreachable")
	ErrFault        = errors.New("WS-Management fault")
	ErrProtocol     = errors.New("unexpected WS-Management response")
)

// CertUnconfirmedError names the certificate a target with no pin presented.
type CertUnconfirmedError struct {
	Fingerprint string
}

func (e *CertUnconfirmedError) Error() string {
	return "certificate " + e.Fingerprint + " is not confirmed"
}

// Fingerprint is the uppercase hex SHA-256 of a DER certificate: the form
// PowerShell prints for $cert.GetCertHashString('SHA256') on the host, so an
// operator can compare the two.
func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// Endpoint is one Windows host's WinRM HTTPS listener and the account Seed
// signs in with.
type Endpoint struct {
	Host string
	Port int
	// User is DOMAIN\user, user@domain, or a local account name.
	User     string
	Password string
	// PinnedCert is the confirmed Fingerprint of the listener's leaf
	// certificate. Empty means none is confirmed yet.
	PinnedCert string
}

// Instance is one WMI object: each property's values by property name. A
// scalar has one value and an array property (IPAddress) several; a property
// that is null on the host is absent.
type Instance map[string][]string

// Value is the property's first value, or "" when it is null.
func (i Instance) Value(name string) string {
	if v := i[name]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// Client queries one endpoint. It keeps one connection open between queries,
// because NTLM authenticates the connection rather than each request.
type Client struct {
	login    Endpoint
	endpoint string
	http     *http.Client
}

// NewClient prepares a client for e. Nothing is sent until Query.
func NewClient(e Endpoint) *Client {
	transport := &http.Transport{
		// No proxy from the environment: the password must reach the host
		// whose certificate is pinned, over a connection Seed opened to it.
		Proxy:           nil,
		MaxConnsPerHost: 1,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: e.Host,
			// The chain is not verified against any CA: VerifyConnection
			// accepts exactly the pinned leaf and nothing else.
			InsecureSkipVerify: true, //nolint:gosec // lgtm[go/disabled-certificate-check] -- ADR-0034
			VerifyConnection: func(cs tls.ConnectionState) error {
				return checkPin(cs, e.PinnedCert)
			},
		},
	}
	return &Client{
		login:    e,
		endpoint: "https://" + net.JoinHostPort(e.Host, strconv.Itoa(e.Port)) + "/wsman",
		http: &http.Client{
			Transport: ntlmssp.Negotiator{RoundTripper: transport},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Close drops the client's connection.
func (c *Client) Close() { c.http.CloseIdleConnections() }

// checkPin accepts the leaf only when pinned is that leaf's fingerprint.
// An empty pin, and a missing one (the zero Endpoint field), are refused:
// VerifyConnection then aborts the handshake, so InsecureSkipVerify cannot
// accept an unpinned host (ADR-0034).
func checkPin(cs tls.ConnectionState, pinned string) error {
	if len(cs.PeerCertificates) == 0 {
		return fmt.Errorf("%w: host presented no certificate", ErrProtocol)
	}
	presented := Fingerprint(cs.PeerCertificates[0].Raw)
	// Missing and empty are the same string. Both must fail closed.
	if pinned == "" {
		return &CertUnconfirmedError{Fingerprint: presented}
	}
	if presented != pinned {
		return fmt.Errorf("%w: host presented %s, pinned %s", ErrCertMismatch, presented, pinned)
	}
	return nil
}

// Query runs one WQL SELECT in root\cimv2 and returns every instance,
// pulling until the host reports the end of the enumeration.
func (c *Client) Query(ctx context.Context, wql string) ([]Instance, error) {
	resp, err := c.call(ctx, actionEnumerate, enumerateBody(wql))
	if err != nil {
		return nil, err
	}
	enum := resp.Body.Enumerate
	if enum == nil {
		return nil, fmt.Errorf("%w: no EnumerateResponse", ErrProtocol)
	}
	all := enum.Items.Instances
	for !enum.done() {
		if len(all) > maxInstances {
			return nil, fmt.Errorf("%w: more than %d instances", ErrProtocol, maxInstances)
		}
		resp, err = c.call(ctx, actionPull, pullBody(enum.Context))
		if err != nil {
			return nil, err
		}
		if enum = resp.Body.Pull; enum == nil {
			return nil, fmt.Errorf("%w: no PullResponse", ErrProtocol)
		}
		all = append(all, enum.Items.Instances...)
	}
	return all, nil
}

func (c *Client) call(ctx context.Context, action, body string) (*envelope, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint,
		strings.NewReader(c.envelopeFor(action, body)))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", soapContentType)
	// The negotiator turns these into NTLM; it never sends them as Basic.
	req.SetBasicAuth(c.login.User, c.login.Password)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, classifyTransport(ctx, err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusInternalServerError:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("%w: host refused the credential", ErrAuth)
	default:
		return nil, fmt.Errorf("%w: HTTP %d", ErrProtocol, resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, classifyTransport(ctx, err)
	}
	if len(raw) > maxResponseBytes {
		return nil, fmt.Errorf("%w: response exceeds %d bytes", ErrProtocol, maxResponseBytes)
	}
	var env envelope
	if err = xml.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProtocol, err)
	}
	if f := env.Body.Fault; f != nil {
		return nil, f.err()
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d without a fault", ErrProtocol, resp.StatusCode)
	}
	return &env, nil
}

// classifyTransport names a failed exchange. The pin errors come back from
// the TLS handshake unchanged inside the transport's error.
func classifyTransport(ctx context.Context, err error) error {
	var unconfirmed *CertUnconfirmedError
	switch {
	case errors.As(err, &unconfirmed):
		return unconfirmed
	case errors.Is(err, ErrCertMismatch), errors.Is(err, ErrProtocol):
		return err
	case ctx.Err() != nil:
		return fmt.Errorf("%w: %w", ErrUnreachable, ctx.Err())
	}
	return fmt.Errorf("%w: %w", ErrUnreachable, err)
}

// envelopeFor is the WS-Management envelope (MS-WSMV 3.1.4) every request
// shares around body, which is escaped where it is built.
func (c *Client) envelopeFor(action, body string) string {
	var b strings.Builder
	b.WriteString(`<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"` +
		` xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing"` +
		` xmlns:n="http://schemas.xmlsoap.org/ws/2004/09/enumeration"` +
		` xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd"><s:Header>`)
	b.WriteString(`<a:To>` + escape(c.endpoint) + `</a:To>`)
	b.WriteString(`<w:ResourceURI s:mustUnderstand="true">` + resourceCIMv2 + `</w:ResourceURI>`)
	b.WriteString(`<a:ReplyTo><a:Address s:mustUnderstand="true">` +
		`http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address></a:ReplyTo>`)
	b.WriteString(`<a:Action s:mustUnderstand="true">` + action + `</a:Action>`)
	b.WriteString(`<w:MaxEnvelopeSize s:mustUnderstand="true">` +
		strconv.Itoa(maxEnvelopeBytes) + `</w:MaxEnvelopeSize>`)
	b.WriteString(`<a:MessageID>uuid:` + uuid.NewString() + `</a:MessageID>`)
	b.WriteString(`<w:Locale xml:lang="en-US" s:mustUnderstand="false"/>`)
	b.WriteString(`<w:OperationTimeout>` + operationTimeout + `</w:OperationTimeout>`)
	b.WriteString(`</s:Header><s:Body>` + body + `</s:Body></s:Envelope>`)
	return b.String()
}

// enumerateBody asks for the query's instances in the Enumerate response
// itself (OptimizeEnumeration), so a short result needs no Pull.
func enumerateBody(wql string) string {
	return `<n:Enumerate><w:OptimizeEnumeration/><w:MaxElements>` + strconv.Itoa(maxElements) + `</w:MaxElements>` +
		`<w:Filter Dialect="http://schemas.microsoft.com/wbem/wsman/1/WQL">` + escape(wql) + `</w:Filter></n:Enumerate>`
}

func pullBody(enumContext string) string {
	return `<n:Pull><n:EnumerationContext>` + escape(enumContext) + `</n:EnumerationContext>` +
		`<n:MaxElements>` + strconv.Itoa(maxElements) + `</n:MaxElements></n:Pull>`
}

func escape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

type envelope struct {
	Body struct {
		Fault     *soapFault    `xml:"Fault"`
		Enumerate *enumResponse `xml:"EnumerateResponse"`
		Pull      *enumResponse `xml:"PullResponse"`
	} `xml:"Body"`
}

// enumResponse is an EnumerateResponse or a PullResponse: the enumeration
// context for the next Pull, the instances, and whether the enumeration has
// ended. EndOfSequence is in the wsman namespace in an optimized
// EnumerateResponse and in the enumeration namespace in a PullResponse; the
// local name matches both.
type enumResponse struct {
	Context       string    `xml:"EnumerationContext"`
	Items         items     `xml:"Items"`
	EndOfSequence *struct{} `xml:"EndOfSequence"`
}

func (r *enumResponse) done() bool { return r.EndOfSequence != nil || r.Context == "" }

type soapFault struct {
	Reason string `xml:"Reason>Text"`
	Detail struct {
		WSManFault struct {
			Code string `xml:"Code,attr"`
		} `xml:"WSManFault"`
	} `xml:"Detail"`
}

func (f *soapFault) err() error {
	reason := strings.TrimSpace(f.Reason)
	if code := f.Detail.WSManFault.Code; code != "" {
		return fmt.Errorf("%w %s: %s", ErrFault, code, reason)
	}
	return fmt.Errorf("%w: %s", ErrFault, reason)
}

// items decodes the instances in an Items element. Each child is one
// instance: an element named for the class when the query selects every
// property, a wsman XmlFragment when it names some. Each child of an instance
// is a property. A datetime property wraps its value in a cim:Datetime
// element, so a property's value is all the text inside it.
type items struct {
	Instances []Instance
}

func (it *items) UnmarshalXML(d *xml.Decoder, _ xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			inst, decodeErr := decodeInstance(d)
			if decodeErr != nil {
				return decodeErr
			}
			it.Instances = append(it.Instances, inst)
		case xml.EndElement:
			return nil
		}
	}
}

func decodeInstance(d *xml.Decoder) (Instance, error) {
	inst := Instance{}
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			value, textErr := propertyText(d)
			if textErr != nil {
				return nil, textErr
			}
			if !isNil(t) {
				inst[t.Name.Local] = append(inst[t.Name.Local], value)
			}
		case xml.EndElement:
			return inst, nil
		}
	}
}

// propertyText returns the text inside the element just opened, including
// text in nested elements, and consumes its end tag.
func propertyText(d *xml.Decoder) (string, error) {
	var b strings.Builder
	for depth := 1; depth > 0; {
		tok, err := d.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			b.Write(t)
		}
	}
	return strings.TrimSpace(b.String()), nil
}

func isNil(e xml.StartElement) bool {
	for _, a := range e.Attr {
		if a.Name.Space == xsiNamespace && a.Name.Local == "nil" {
			return a.Value == "true"
		}
	}
	return false
}
