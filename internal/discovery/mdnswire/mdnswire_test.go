package mdnswire_test

import (
	"bytes"
	"net/netip"
	"slices"
	"testing"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/MustardSeedNetworks/seed/internal/discovery/mdnswire"
)

// response is a header for a response with the given section counts.
func response(answers, additionals byte) []byte {
	return []byte{0, 0, 0x84, 0x00, 0, 0, 0, answers, 0, 0, 0, additionals}
}

func labels(raw []byte, ls ...string) []byte {
	for _, l := range ls {
		raw = append(raw, byte(len(l)))
		raw = append(raw, l...)
	}
	return append(raw, 0)
}

func TestParseReadsEveryRecordTypeAndFollowsPointers(t *testing.T) {
	// Built by dnsmessage with compression on, so the reader is checked
	// against an independent encoder and against pointers it did not write.
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true})
	b.EnableCompression()
	must(t, b.StartQuestions())
	must(t, b.Question(dnsmessage.Question{
		Name: dnsmessage.MustNewName("_ipp._tcp.local."), Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET,
	}))
	must(t, b.StartAnswers())
	hdr := func(name string, typ dnsmessage.Type) dnsmessage.ResourceHeader {
		return dnsmessage.ResourceHeader{Name: dnsmessage.MustNewName(name), Type: typ, Class: dnsmessage.ClassINET}
	}
	must(t, b.PTRResource(hdr("_ipp._tcp.local.", dnsmessage.TypePTR),
		dnsmessage.PTRResource{PTR: dnsmessage.MustNewName("Front Desk._ipp._tcp.local.")}))
	must(t, b.StartAuthorities())
	must(t, b.StartAdditionals())
	must(t, b.SRVResource(hdr("Front Desk._ipp._tcp.local.", dnsmessage.TypeSRV),
		dnsmessage.SRVResource{Priority: 1, Weight: 2, Port: 631, Target: dnsmessage.MustNewName("printer.local.")}))
	must(t, b.TXTResource(hdr("Front Desk._ipp._tcp.local.", dnsmessage.TypeTXT),
		dnsmessage.TXTResource{TXT: []string{"rp=ipp/print", ""}}))
	must(t, b.AResource(hdr("printer.local.", dnsmessage.TypeA), dnsmessage.AResource{A: [4]byte{10, 0, 0, 5}}))
	must(t, b.AAAAResource(hdr("printer.local.", dnsmessage.TypeAAAA),
		dnsmessage.AAAAResource{AAAA: netip.MustParseAddr("2001:db8::5").As16()}))
	must(t, b.MXResource(hdr("printer.local.", dnsmessage.TypeMX),
		dnsmessage.MXResource{MX: dnsmessage.MustNewName("mail.local.")}))
	data, err := b.Finish()
	must(t, err)

	msg, err := mdnswire.Parse(data)
	must(t, err)
	if !msg.Response {
		t.Error("Response = false, want true")
	}
	want := []mdnswire.Record{
		{
			Section: mdnswire.Answer, Name: mdnswire.Name{"_ipp", "_tcp", "local"}, Type: mdnswire.TypePTR,
			Target: mdnswire.Name{"Front Desk", "_ipp", "_tcp", "local"},
		},
		{
			Section: mdnswire.Additional, Name: mdnswire.Name{"Front Desk", "_ipp", "_tcp", "local"},
			Type: mdnswire.TypeSRV, Target: mdnswire.Name{"printer", "local"}, Port: 631,
		},
		{
			Section: mdnswire.Additional, Name: mdnswire.Name{"Front Desk", "_ipp", "_tcp", "local"},
			Type: mdnswire.TypeTXT, TXT: []string{"rp=ipp/print", ""},
		},
		{
			Section: mdnswire.Additional, Name: mdnswire.Name{"printer", "local"}, Type: mdnswire.TypeA,
			Addr: netip.MustParseAddr("10.0.0.5"),
		},
		{
			Section: mdnswire.Additional, Name: mdnswire.Name{"printer", "local"}, Type: mdnswire.TypeAAAA,
			Addr: netip.MustParseAddr("2001:db8::5"),
		},
	}
	if len(msg.Records) != len(want) {
		t.Fatalf("records = %+v, want %d (the MX skipped)", msg.Records, len(want))
	}
	for i, w := range want {
		g := msg.Records[i]
		if g.Section != w.Section || g.Type != w.Type || !slices.Equal(g.Name, w.Name) ||
			!slices.Equal(g.Target, w.Target) || g.Port != w.Port || !slices.Equal(g.TXT, w.TXT) || g.Addr != w.Addr {
			t.Errorf("record %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestParseKeepsADotInsideALabel(t *testing.T) {
	raw := labels(response(1, 0), "Office No. 5", "_ipp", "_tcp", "local")
	// A, IN, TTL 120, 10.0.0.5.
	raw = append(raw, 0, 1, 0, 1, 0, 0, 0, 120, 0, 4, 10, 0, 0, 5)

	msg, err := mdnswire.Parse(raw)
	must(t, err)
	if len(msg.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(msg.Records))
	}
	name := msg.Records[0].Name
	if want := (mdnswire.Name{"Office No. 5", "_ipp", "_tcp", "local"}); !slices.Equal(name, want) {
		t.Errorf("labels = %q, want %q", name, want)
	}
}

func TestParseRefusesMalformedMessages(t *testing.T) {
	aRecord := func(rdata ...byte) []byte {
		raw := labels(response(1, 0), "h", "local")
		raw = append(raw, 0, 1, 0, 1, 0, 0, 0, 120, 0, byte(len(rdata)))
		return append(raw, rdata...)
	}
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"shorter than a header", []byte{0, 0, 0x84}},
		{"a count with no record behind it", response(1, 0)},
		{"a label running past the end", append(response(1, 0), 5, 'a', 'b')},
		{"a reserved label type", append(response(1, 0), 0x40, 0)},
		{"a pointer to itself", append(response(1, 0), 0xC0, 12)},
		{"a pointer past the end", append(response(1, 0), 0xC0, 0xFF)},
		{"a pointer cut in half", append(response(1, 0), 0xC0)},
		{"a name over 255 bytes", labels(response(1, 0), slices.Repeat([]string{"abcdefghij"}, 24)...)},
		{"an A record of three bytes", aRecord(10, 0, 0)},
		{"record data running past the end", aRecord(10, 0, 0, 5)[:len(aRecord(10, 0, 0, 5))-1]},
		{"a TXT string running past its record", txtRecord(3, 'a')},
		{"an SRV with no target", srvRecord(0, 0, 0, 0, 2, 0x77)},
		{"a PTR whose name overruns its record", ptrRecord(1, 2, 'a', 'b', 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if msg, err := mdnswire.Parse(tc.data); err == nil {
				t.Errorf("Parse succeeded with %+v, want an error", msg)
			}
		})
	}
}

func txtRecord(rdata ...byte) []byte { return record(16, rdata) }
func srvRecord(rdata ...byte) []byte { return record(33, rdata) }

// ptrRecord declares rdLen bytes of record data but carries rdata, which may
// be longer, so a name inside it can overrun the declared length.
func ptrRecord(rdLen byte, rdata ...byte) []byte {
	raw := labels(response(1, 0), "h", "local")
	raw = append(raw, 0, 12, 0, 1, 0, 0, 0, 120, 0, rdLen)
	return append(raw, rdata...)
}

func record(rtype byte, rdata []byte) []byte {
	raw := labels(response(1, 0), "h", "local")
	raw = append(raw, 0, rtype, 0, 1, 0, 0, 0, 120, 0, byte(len(rdata)))
	return append(raw, rdata...)
}

func TestNameRoundTripsThroughPresentationForm(t *testing.T) {
	for _, tc := range []struct {
		name mdnswire.Name
		text string
	}{
		{mdnswire.Name{"printer", "local"}, "printer.local"},
		{mdnswire.Name{"Office No. 5", "_ipp", "_tcp", "local"}, `Office No\. 5._ipp._tcp.local`},
		{mdnswire.Name{`has\backslash`, "local"}, `has\\backslash.local`},
		{mdnswire.Name{"Café", "local"}, "Café.local"},
	} {
		if got := tc.name.String(); got != tc.text {
			t.Errorf("String(%q) = %q, want %q", tc.name, got, tc.text)
		}
		back, err := mdnswire.ParseName(tc.text + ".")
		if err != nil || !slices.Equal(back, tc.name) {
			t.Errorf("ParseName(%q) = %q, %v; want %q", tc.text+".", back, err, tc.name)
		}
	}
	for _, bad := range []string{"", ".", "a..b", `trailing\`} {
		if got, err := mdnswire.ParseName(bad); err == nil {
			t.Errorf("ParseName(%q) = %q, want an error", bad, got)
		}
	}
}

func TestQueryEncodesOneQuestion(t *testing.T) {
	data, err := mdnswire.Query(mdnswire.Name{"Office No. 5", "_ipp", "_tcp", "local"}, mdnswire.TypeSRV)
	must(t, err)
	want := labels([]byte{0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0}, "Office No. 5", "_ipp", "_tcp", "local")
	want = append(want, 0, 33, 0, 1)
	if !bytes.Equal(data, want) {
		t.Errorf("query = % x\nwant    % x", data, want)
	}

	for _, bad := range []mdnswire.Name{
		nil,
		{"a", "", "local"},
		{string(make([]byte, 64)), "local"},
		slices.Repeat(mdnswire.Name{"abcdefghij"}, 24),
	} {
		if _, badErr := mdnswire.Query(bad, mdnswire.TypeA); badErr == nil {
			t.Errorf("Query(%q) succeeded, want an error", bad)
		}
	}
}

// FuzzParse feeds arbitrary bytes from the untrusted segment. Parse must
// never panic or loop, and every name it returns must survive the
// presentation round trip the browse keys records by.
func FuzzParse(f *testing.F) {
	f.Add(labels(response(1, 0), "Office No. 5", "local"))
	f.Add(append(response(1, 0), 0xC0, 12))
	f.Add(ptrRecord(4, 0xC0, 12))
	f.Add(txtRecord(1, 'a', 0))
	f.Fuzz(func(t *testing.T, data []byte) {
		msg, err := mdnswire.Parse(data)
		if err != nil {
			return
		}
		for _, r := range msg.Records {
			for _, n := range []mdnswire.Name{r.Name, r.Target} {
				if len(n) == 0 || slices.Contains(n, "") {
					continue
				}
				back, nameErr := mdnswire.ParseName(n.String())
				if nameErr != nil || !slices.Equal(back, n) {
					t.Fatalf("name %q round-tripped to %q, %v", n, back, nameErr)
				}
			}
		}
	})
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
