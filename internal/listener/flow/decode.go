package flow

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"time"
)

// Decode errors. A datagram that fails part-way still returns the
// records decoded before the failure.
var (
	ErrTruncated          = errors.New("flow: truncated datagram")
	ErrUnsupportedVersion = errors.New("flow: unsupported export version")
	ErrMalformed          = errors.New("flow: malformed datagram")
)

const (
	v5HeaderLen    = 24
	v5RecordLen    = 48
	v5MaxRecords   = 30
	v9HeaderLen    = 20
	ipfixHeaderLen = 16
	setHeaderLen   = 4
	versionLen     = 2
	bitsPerByte    = 8

	// A field specifier without an enterprise number, the enterprise
	// number that may follow it, an IPFIX withdrawal record (ID and a
	// zero count) and an options template record header.
	specifierLen     = 4
	enterpriseNumLen = 4
	withdrawalLen    = 4
	optionsHeaderLen = 6

	// v9 set IDs (RFC 3954 §5.2) and IPFIX set IDs (RFC 7011 §3.3.2).
	v9TemplateSet        = 0
	v9OptionsTemplateSet = 1
	ipfixTemplateSet     = 2
	ipfixOptionsTemplate = 3
	minDataSetID         = 256

	// variableLength marks an IPFIX field whose length is carried in
	// each record (RFC 7011 §7).
	variableLength = 0xFFFF
	// longVariableLength is the one-byte length that announces a
	// two-byte length follows.
	longVariableLength = 255
	// enterpriseBit marks an IPFIX field specifier followed by a
	// four-byte enterprise number (RFC 7011 §3.2).
	enterpriseBit = 0x8000

	// v5SamplingMask keeps the 14-bit interval of the v5 sampling field;
	// the top two bits are the sampling mode.
	v5SamplingMask = 0x3FFF

	// maxTemplateFields bounds one template so the cache's worst case
	// stays small: a v9 set could otherwise declare 16k fields.
	maxTemplateFields = 256
	// maxTemplates bounds the cache across every exporter. The exporter
	// boot times and sampling rates learned from options records share it.
	maxTemplates = 4096
	// templateLifetime drops a template its exporter has stopped
	// refreshing. Exporters resend templates every few minutes to every
	// 30 minutes (RFC 7011 §8.4 leaves the interval to them); twice the
	// slowest common interval keeps a live exporter's templates.
	templateLifetime = time.Hour

	// uptimeWrap is the period of a 32-bit millisecond uptime.
	uptimeWrap = (math.MaxUint32 + 1) * time.Millisecond
)

// Information element IDs (RFC 7012; v9 field types share the numbers).
const (
	ieOctetDeltaCount       = 1
	iePacketDeltaCount      = 2
	ieProtocolIdentifier    = 4
	ieTCPControlBits        = 6
	ieSourceTransportPort   = 7
	ieSourceIPv4Address     = 8
	ieIngressInterface      = 10
	ieDestTransportPort     = 11
	ieDestIPv4Address       = 12
	ieEgressInterface       = 14
	ieFlowEndSysUpTime      = 21
	ieFlowStartSysUpTime    = 22
	ieSourceIPv6Address     = 27
	ieDestIPv6Address       = 28
	ieOctetTotalCount       = 85
	iePacketTotalCount      = 86
	ieFlowStartSeconds      = 150
	ieFlowEndSeconds        = 151
	ieFlowStartMilliseconds = 152
	ieFlowEndMilliseconds   = 153
	// ieSystemInitTimeMilliseconds is the exporter's boot time, sent in
	// an options record; IPFIX uptime-relative times count from it.
	ieSystemInitTimeMilliseconds = 160
)

// Result is what one datagram decoded to.
type Result struct {
	Records []Record
	// MissingTemplate counts data sets skipped because their template
	// has not been learned yet or has expired.
	MissingTemplate int
}

// Decoder turns export datagrams into records. It holds the template
// cache, so one Decoder serves one read loop and is not safe for
// concurrent use.
type Decoder struct {
	templates map[templateKey]*template
	// initTimes holds each IPFIX exporter's systemInitTimeMilliseconds,
	// sent in an options record, against which uptime-relative flow
	// times resolve.
	initTimes map[domainKey]initTime
	// rates holds the sampling rates exporters announced in options
	// records, by the scope each covers.
	rates map[rateKey]learnedRate
}

// domainKey identifies one exporter's observation domain.
type domainKey struct {
	exporter netip.Addr
	domain   uint32
}

type initTime struct {
	at, learned time.Time
}

// NewDecoder returns a Decoder with an empty template cache.
func NewDecoder() *Decoder {
	return &Decoder{
		templates: make(map[templateKey]*template),
		initTimes: make(map[domainKey]initTime),
		rates:     make(map[rateKey]learnedRate),
	}
}

// templateKey scopes a template ID to the exporter and observation
// domain that defined it (RFC 7011 §8).
type templateKey struct {
	exporter netip.Addr
	version  uint16
	domain   uint32
	id       uint16
}

type field struct {
	id         uint16
	length     uint16
	enterprise bool
	// scope marks an options template's scope field. In v9 its id is a
	// scope type, not a field type.
	scope bool
}

type template struct {
	fields []field
	// options templates describe exporter metadata, not flows; their
	// data sets are recognised and skipped.
	options bool
	// minLen is the smallest record the template can describe: the fixed
	// fields plus one length byte per variable-length field.
	minLen  int
	learned time.Time
}

// exportHeader carries the per-datagram values records are resolved
// against.
type exportHeader struct {
	exporter   netip.Addr
	version    uint16
	domain     uint32
	exportTime time.Time
	// sysUptime is the v9 header's uptime in milliseconds; IPFIX has none.
	sysUptime uint32
	hasUptime bool
	// initTime is the IPFIX exporter's boot time, when it has sent one.
	initTime time.Time
	hasInit  bool
}

func (h *exportHeader) format() Format {
	if h.version == VersionIPFIX {
		return FormatIPFIX
	}
	return FormatNetFlow9
}

// Decode decodes one datagram received from exporter at now.
func (d *Decoder) Decode(exporter netip.Addr, pkt []byte, now time.Time) (Result, error) {
	if len(pkt) < versionLen {
		return Result{}, ErrTruncated
	}
	switch v := binary.BigEndian.Uint16(pkt); v {
	case 0:
		return decodeSFlow(exporter, pkt, now)
	case VersionNetFlow5:
		return decodeV5(exporter, pkt)
	case VersionNetFlow9:
		return d.decodeV9(exporter, pkt, now)
	case VersionIPFIX:
		return d.decodeIPFIX(exporter, pkt, now)
	default:
		return Result{}, fmt.Errorf("%w: %d", ErrUnsupportedVersion, v)
	}
}

func decodeV5(exporter netip.Addr, pkt []byte) (Result, error) {
	if len(pkt) < v5HeaderLen {
		return Result{}, ErrTruncated
	}
	be := binary.BigEndian
	count := int(be.Uint16(pkt[2:]))
	if count == 0 || count > v5MaxRecords {
		return Result{}, fmt.Errorf("%w: v5 record count %d", ErrMalformed, count)
	}
	if len(pkt) < v5HeaderLen+count*v5RecordLen {
		return Result{}, ErrTruncated
	}
	uptime := be.Uint32(pkt[4:])
	exportTime := time.Unix(int64(be.Uint32(pkt[8:])), int64(be.Uint32(pkt[12:]))).UTC()
	domain := uint32(be.Uint16(pkt[20:]))
	sampling := uint64(be.Uint16(pkt[22:]) & v5SamplingMask)
	if sampling == 0 {
		sampling = 1
	}

	records := make([]Record, 0, count)
	for i := range count {
		r := pkt[v5HeaderLen+i*v5RecordLen:]
		records = append(records, Record{
			Exporter:          exporter,
			Format:            FormatNetFlow5,
			ObservationDomain: domain,
			Start:             uptimeTime(exportTime, uptime, be.Uint32(r[24:])),
			End:               uptimeTime(exportTime, uptime, be.Uint32(r[28:])),
			SrcAddr:           netip.AddrFrom4([4]byte(r[0:4])),
			DstAddr:           netip.AddrFrom4([4]byte(r[4:8])),
			SrcPort:           be.Uint16(r[32:]),
			DstPort:           be.Uint16(r[34:]),
			Protocol:          r[38],
			TCPFlags:          r[37],
			Bytes:             uint64(be.Uint32(r[20:])) * sampling,
			Packets:           uint64(be.Uint32(r[16:])) * sampling,
			InputIf:           uint32(be.Uint16(r[12:])),
			OutputIf:          uint32(be.Uint16(r[14:])),
		})
	}
	return Result{Records: records}, nil
}

func (d *Decoder) decodeV9(exporter netip.Addr, pkt []byte, now time.Time) (Result, error) {
	if len(pkt) < v9HeaderLen {
		return Result{}, ErrTruncated
	}
	be := binary.BigEndian
	hdr := exportHeader{
		exporter:   exporter,
		version:    VersionNetFlow9,
		domain:     be.Uint32(pkt[16:]),
		exportTime: time.Unix(int64(be.Uint32(pkt[8:])), 0).UTC(),
		sysUptime:  be.Uint32(pkt[4:]),
		hasUptime:  true,
	}
	return d.decodeSets(&hdr, pkt[v9HeaderLen:], now)
}

func (d *Decoder) decodeIPFIX(exporter netip.Addr, pkt []byte, now time.Time) (Result, error) {
	if len(pkt) < ipfixHeaderLen {
		return Result{}, ErrTruncated
	}
	be := binary.BigEndian
	length := int(be.Uint16(pkt[2:]))
	if length < ipfixHeaderLen || length > len(pkt) {
		return Result{}, ErrTruncated
	}
	hdr := exportHeader{
		exporter:   exporter,
		version:    VersionIPFIX,
		domain:     be.Uint32(pkt[12:]),
		exportTime: time.Unix(int64(be.Uint32(pkt[4:])), 0).UTC(),
	}
	key := domainKey{exporter: exporter, domain: hdr.domain}
	if it, ok := d.initTimes[key]; ok && now.Sub(it.learned) <= templateLifetime {
		hdr.initTime, hdr.hasInit = it.at, true
	}
	return d.decodeSets(&hdr, pkt[ipfixHeaderLen:length], now)
}

// decodeSets walks the sets (v9 "FlowSets") that follow a v9 or IPFIX
// header. Both formats frame sets identically and differ only in the
// template set IDs and the field-specifier encoding.
func (d *Decoder) decodeSets(hdr *exportHeader, body []byte, now time.Time) (Result, error) {
	var res Result
	for len(body) >= setHeaderLen {
		id := binary.BigEndian.Uint16(body)
		length := int(binary.BigEndian.Uint16(body[2:]))
		if length < setHeaderLen || length > len(body) {
			return res, fmt.Errorf("%w: set %d length %d", ErrMalformed, id, length)
		}
		set := body[setHeaderLen:length]
		body = body[length:]

		var err error
		switch {
		case id >= minDataSetID:
			err = d.decodeDataSet(hdr, id, set, now, &res)
		case hdr.version == VersionNetFlow9 && id == v9TemplateSet:
			err = d.learnTemplates(hdr, set, false, now)
		case hdr.version == VersionNetFlow9 && id == v9OptionsTemplateSet:
			err = d.learnV9OptionsTemplates(hdr, set, now)
		case hdr.version == VersionIPFIX && id == ipfixTemplateSet:
			err = d.learnTemplates(hdr, set, false, now)
		case hdr.version == VersionIPFIX && id == ipfixOptionsTemplate:
			err = d.learnTemplates(hdr, set, true, now)
		}
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// learnTemplates reads the template records of a v9 template set, or an
// IPFIX template or options template set.
func (d *Decoder) learnTemplates(hdr *exportHeader, set []byte, options bool, now time.Time) error {
	for len(set) >= specifierLen {
		// Zero is never a template ID; the rest is padding.
		if binary.BigEndian.Uint16(set) == 0 {
			return nil
		}
		rest, err := d.learnTemplate(hdr, set, options, now)
		if err != nil {
			return err
		}
		set = rest
	}
	return nil
}

// learnTemplate reads one template record, or an IPFIX withdrawal, and
// returns the bytes after it. An IPFIX options template record carries a
// scope field count before its specifiers; the scope fields are ordinary
// specifiers for decoding purposes.
func (d *Decoder) learnTemplate(hdr *exportHeader, set []byte, options bool, now time.Time) ([]byte, error) {
	id := binary.BigEndian.Uint16(set)
	count := int(binary.BigEndian.Uint16(set[2:]))
	if count == 0 && hdr.version == VersionIPFIX {
		d.withdraw(hdr, id, options)
		return set[withdrawalLen:], nil
	}
	headerLen := withdrawalLen
	if options {
		headerLen = optionsHeaderLen
	}
	if len(set) < headerLen {
		return nil, ErrTruncated
	}
	if id < minDataSetID || count == 0 || count > maxTemplateFields {
		return nil, fmt.Errorf("%w: template %d with %d fields", ErrMalformed, id, count)
	}
	scope := 0
	if options {
		if scope = int(binary.BigEndian.Uint16(set[4:])); scope == 0 || scope > count {
			return nil, fmt.Errorf("%w: options template %d scope count %d", ErrMalformed, id, scope)
		}
	}
	fields, rest, err := readFieldSpecifiers(set[headerLen:], count, hdr.version == VersionIPFIX)
	if err != nil {
		return nil, err
	}
	for i := range scope {
		fields[i].scope = true
	}
	return rest, d.learn(hdr, id, fields, options, now)
}

// learnV9OptionsTemplates reads a v9 options template set (RFC 3954
// §6.1), whose records give scope and option lengths in bytes.
func (d *Decoder) learnV9OptionsTemplates(hdr *exportHeader, set []byte, now time.Time) error {
	for len(set) >= optionsHeaderLen {
		id := binary.BigEndian.Uint16(set)
		if id == 0 {
			return nil
		}
		scopeLen := int(binary.BigEndian.Uint16(set[2:]))
		optionLen := int(binary.BigEndian.Uint16(set[4:]))
		count := (scopeLen + optionLen) / specifierLen
		if id < minDataSetID || scopeLen%specifierLen != 0 || optionLen%specifierLen != 0 ||
			count == 0 || count > maxTemplateFields {
			return fmt.Errorf("%w: options template %d", ErrMalformed, id)
		}
		fields, rest, err := readFieldSpecifiers(set[optionsHeaderLen:], count, false)
		if err != nil {
			return err
		}
		for i := range scopeLen / specifierLen {
			fields[i].scope = true
		}
		set = rest
		if err = d.learn(hdr, id, fields, true, now); err != nil {
			return err
		}
	}
	return nil
}

func readFieldSpecifiers(b []byte, count int, ipfix bool) ([]field, []byte, error) {
	fields := make([]field, 0, count)
	for range count {
		if len(b) < specifierLen {
			return nil, nil, ErrTruncated
		}
		f := field{id: binary.BigEndian.Uint16(b), length: binary.BigEndian.Uint16(b[2:])}
		b = b[specifierLen:]
		if ipfix && f.id&enterpriseBit != 0 {
			if len(b) < enterpriseNumLen {
				return nil, nil, ErrTruncated
			}
			f.id &^= enterpriseBit
			f.enterprise = true
			b = b[enterpriseNumLen:]
		}
		if f.length == variableLength && !ipfix {
			return nil, nil, fmt.Errorf("%w: variable-length field in v9", ErrMalformed)
		}
		fields = append(fields, f)
	}
	return fields, b, nil
}

func (d *Decoder) learn(hdr *exportHeader, id uint16, fields []field, options bool, now time.Time) error {
	minLen := 0
	for _, f := range fields {
		if f.length == variableLength {
			minLen++
		} else {
			minLen += int(f.length)
		}
	}
	if minLen == 0 {
		return fmt.Errorf("%w: template %d describes empty records", ErrMalformed, id)
	}
	key := templateKey{exporter: hdr.exporter, version: hdr.version, domain: hdr.domain, id: id}
	if _, ok := d.templates[key]; !ok && len(d.templates) >= maxTemplates {
		evictOldest(d.templates, func(t *template) time.Time { return t.learned })
	}
	d.templates[key] = &template{fields: fields, options: options, minLen: minLen, learned: now}
	return nil
}

// withdraw handles an IPFIX template withdrawal (RFC 7011 §8.1). ID 2
// (or 3 in an options template set) withdraws every template of that
// kind in the domain.
func (d *Decoder) withdraw(hdr *exportHeader, id uint16, options bool) {
	if id != ipfixTemplateSet && id != ipfixOptionsTemplate {
		delete(d.templates, templateKey{exporter: hdr.exporter, version: hdr.version, domain: hdr.domain, id: id})
		return
	}
	for k, t := range d.templates {
		if k.exporter == hdr.exporter && k.version == hdr.version && k.domain == hdr.domain && t.options == options {
			delete(d.templates, k)
		}
	}
}

// evictOldest deletes the entry of m learned longest ago.
func evictOldest[K comparable, V any](m map[K]V, learned func(V) time.Time) {
	var (
		oldest    K
		oldestAt  time.Time
		haveFirst bool
	)
	for k, v := range m {
		if at := learned(v); !haveFirst || at.Before(oldestAt) {
			oldest, oldestAt, haveFirst = k, at, true
		}
	}
	delete(m, oldest)
}

func (d *Decoder) lookup(hdr *exportHeader, id uint16, now time.Time) *template {
	key := templateKey{exporter: hdr.exporter, version: hdr.version, domain: hdr.domain, id: id}
	t := d.templates[key]
	if t == nil {
		return nil
	}
	if now.Sub(t.learned) > templateLifetime {
		delete(d.templates, key)
		return nil
	}
	return t
}

func (d *Decoder) decodeDataSet(hdr *exportHeader, id uint16, set []byte, now time.Time, res *Result) error {
	t := d.lookup(hdr, id, now)
	if t == nil {
		res.MissingTemplate++
		return nil
	}
	if t.options {
		return d.readExporterOptions(hdr, id, t, set, now)
	}
	// Anything shorter than one record is padding.
	for len(set) >= t.minLen {
		rec, n, ok := d.decodeRecord(hdr, t, set, now)
		if !ok {
			return fmt.Errorf("%w: data set %d overruns its record", ErrMalformed, id)
		}
		res.Records = append(res.Records, rec)
		set = set[n:]
	}
	return nil
}

// flowTimes collects whichever timestamp elements a template carries.
type flowTimes struct {
	start, end         time.Time
	startUp, endUp     uint32
	hasStartUp, hasEnd bool
	hasEndUp, hasStart bool
}

// counters keeps delta and total counts apart: a template may carry
// either, and the delta is the one that sums correctly across records.
type counters struct {
	deltaBytes, deltaPackets, totalBytes, totalPackets uint64
	hasDeltaBytes, hasDeltaPackets                     bool
}

func (d *Decoder) decodeRecord(hdr *exportHeader, t *template, b []byte, now time.Time) (Record, int, bool) {
	rec := Record{Exporter: hdr.exporter, Format: hdr.format(), ObservationDomain: hdr.domain}
	var (
		times flowTimes
		cnt   counters
		smp   sampling
	)
	n, ok := walkRecord(t, b, func(f field, v []byte) {
		if !smp.apply(f.id, v) {
			applyField(&rec, &times, &cnt, f.id, v)
		}
	})
	if !ok {
		return Record{}, 0, false
	}
	rec.Bytes, rec.Packets = cnt.deltaBytes, cnt.deltaPackets
	if !cnt.hasDeltaBytes {
		rec.Bytes = cnt.totalBytes
	}
	if !cnt.hasDeltaPackets {
		rec.Packets = cnt.totalPackets
	}
	r := d.recordRate(hdr, &rec, &smp, now)
	rec.Bytes, rec.Packets = r.scale(rec.Bytes), r.scale(rec.Packets)
	rec.Start, rec.End = resolveTimes(hdr, &times)
	return rec, n, true
}

// readExporterOptions reads an options data set for the values flows
// depend on: the exporter's systemInitTimeMilliseconds and its sampling
// rates. Every other option is skipped.
func (d *Decoder) readExporterOptions(hdr *exportHeader, id uint16, t *template, set []byte, now time.Time) error {
	for len(set) >= t.minLen {
		n, ok := d.readOptionsRecord(hdr, t, set, now)
		if !ok {
			return fmt.Errorf("%w: options data set %d overruns its record", ErrMalformed, id)
		}
		set = set[n:]
	}
	return nil
}

// readOptionsRecord reads the options record at the start of b and returns
// its length, or false when it overruns b.
func (d *Decoder) readOptionsRecord(hdr *exportHeader, t *template, b []byte, now time.Time) (int, bool) {
	var (
		smp     sampling
		ifIndex uint32
		hasIf   bool
	)
	n, ok := walkRecord(t, b, func(f field, v []byte) {
		switch {
		case f.scope && hdr.version == VersionNetFlow9:
			if f.id == v9ScopeInterface {
				ifIndex, hasIf = readUint[uint32](v)
			}
		case f.scope && f.id == ieIngressInterface:
			ifIndex, hasIf = readUint[uint32](v)
		case f.id == ieSystemInitTimeMilliseconds && hdr.version == VersionIPFIX:
			if at, valid := readMilliseconds(v); valid {
				hdr.initTime, hdr.hasInit = at, true
				d.learnInitTime(hdr, at, now)
			}
		default:
			smp.apply(f.id, v)
		}
	})
	if !ok {
		return 0, false
	}
	if r, valid := smp.rate(); valid {
		key := rateKey{exporter: hdr.exporter, domain: hdr.domain, scope: scopeDomain}
		switch {
		case smp.hasSampler:
			key.scope, key.id = scopeSampler, smp.sampler
		case hasIf:
			key.scope, key.id = scopeInterface, uint64(ifIndex)
		}
		d.learnRate(key, r, now)
	}
	return n, true
}

func (d *Decoder) learnInitTime(hdr *exportHeader, at, now time.Time) {
	key := domainKey{exporter: hdr.exporter, domain: hdr.domain}
	if _, ok := d.initTimes[key]; !ok && len(d.initTimes) >= maxTemplates {
		evictOldest(d.initTimes, func(it initTime) time.Time { return it.learned })
	}
	d.initTimes[key] = initTime{at: at, learned: now}
}

// walkRecord calls fn with each IANA field of the record at the start of b
// and returns the record's length, or false when the record overruns b.
// Enterprise-specific fields are stepped over.
func walkRecord(t *template, b []byte, fn func(f field, v []byte)) (int, bool) {
	off := 0
	for _, f := range t.fields {
		n := int(f.length)
		if f.length == variableLength {
			if off >= len(b) {
				return 0, false
			}
			n = int(b[off])
			off++
			if n == longVariableLength {
				if off+2 > len(b) {
					return 0, false
				}
				n = int(binary.BigEndian.Uint16(b[off:]))
				off += 2
			}
		}
		if off+n > len(b) {
			return 0, false
		}
		if !f.enterprise {
			fn(f, b[off:off+n])
		}
		off += n
	}
	return off, true
}

func applyField(rec *Record, times *flowTimes, cnt *counters, id uint16, v []byte) {
	switch id {
	case ieSourceIPv4Address, ieSourceIPv6Address:
		if a, ok := netip.AddrFromSlice(v); ok {
			rec.SrcAddr = a
		}
	case ieDestIPv4Address, ieDestIPv6Address:
		if a, ok := netip.AddrFromSlice(v); ok {
			rec.DstAddr = a
		}
	case ieSourceTransportPort:
		rec.SrcPort, _ = readUint[uint16](v)
	case ieDestTransportPort:
		rec.DstPort, _ = readUint[uint16](v)
	case ieProtocolIdentifier:
		if len(v) == 1 {
			rec.Protocol = v[0]
		}
	case ieTCPControlBits:
		// IPFIX widened the field to 16 bits; the classic flags are the
		// low byte either way.
		if len(v) == 1 || len(v) == 2 {
			rec.TCPFlags = v[len(v)-1]
		}
	case ieIngressInterface:
		rec.InputIf, _ = readUint[uint32](v)
	case ieEgressInterface:
		rec.OutputIf, _ = readUint[uint32](v)
	case ieOctetDeltaCount:
		cnt.deltaBytes, cnt.hasDeltaBytes = readUint[uint64](v)
	case iePacketDeltaCount:
		cnt.deltaPackets, cnt.hasDeltaPackets = readUint[uint64](v)
	case ieOctetTotalCount:
		cnt.totalBytes, _ = readUint[uint64](v)
	case iePacketTotalCount:
		cnt.totalPackets, _ = readUint[uint64](v)
	default:
		applyTimeField(times, id, v)
	}
}

func applyTimeField(times *flowTimes, id uint16, v []byte) {
	switch id {
	case ieFlowStartSysUpTime:
		times.startUp, times.hasStartUp = readUint[uint32](v)
	case ieFlowEndSysUpTime:
		times.endUp, times.hasEndUp = readUint[uint32](v)
	case ieFlowStartSeconds:
		times.start, times.hasStart = readSeconds(v)
	case ieFlowEndSeconds:
		times.end, times.hasEnd = readSeconds(v)
	case ieFlowStartMilliseconds:
		times.start, times.hasStart = readMilliseconds(v)
	case ieFlowEndMilliseconds:
		times.end, times.hasEnd = readMilliseconds(v)
	}
}

// resolveTimes picks absolute timestamps over uptime-relative ones and
// falls back to the export time, so every record has a start and end.
func resolveTimes(hdr *exportHeader, times *flowTimes) (time.Time, time.Time) {
	start, end := hdr.exportTime, hdr.exportTime
	switch {
	case times.hasStart:
		start = times.start
	case times.hasStartUp && hdr.hasUptime:
		start = uptimeTime(hdr.exportTime, hdr.sysUptime, times.startUp)
	case times.hasStartUp && hdr.hasInit:
		start = sinceInit(hdr, times.startUp)
	}
	switch {
	case times.hasEnd:
		end = times.end
	case times.hasEndUp && hdr.hasUptime:
		end = uptimeTime(hdr.exportTime, hdr.sysUptime, times.endUp)
	case times.hasEndUp && hdr.hasInit:
		end = sinceInit(hdr, times.endUp)
	}
	return start, end
}

// uptimeTime converts an exporter uptime in milliseconds to wall time,
// given the export time and the uptime at export. The difference is taken
// modulo 2^32 so it survives the 49.7-day uptime wrap; a timestamp a
// little after the export uptime reads as a small negative age.
func uptimeTime(exportTime time.Time, uptime, at uint32) time.Time {
	if age := uptime - at; age <= math.MaxInt32 {
		return exportTime.Add(-time.Duration(age) * time.Millisecond)
	}
	return exportTime.Add(time.Duration(at-uptime) * time.Millisecond)
}

// sinceInit resolves an IPFIX uptime-relative time against the exporter's
// boot time. The 32-bit uptime wraps every 49.7 days, so of the instants
// it can denote, the one nearest the export time is the flow's.
func sinceInit(hdr *exportHeader, uptime uint32) time.Time {
	t := hdr.initTime.Add(time.Duration(uptime) * time.Millisecond)
	wraps := math.Round(float64(hdr.exportTime.Sub(t)) / float64(uptimeWrap))
	return t.Add(time.Duration(wraps) * uptimeWrap)
}

// readUint reads a big-endian unsigned integer into T. IPFIX allows
// reduced-size encoding of any integer (RFC 7011 §6.2), so any length up
// to T's size is accepted; a longer or empty field is not T and is
// ignored.
func readUint[T uint16 | uint32 | uint64](v []byte) (T, bool) {
	var n T
	if len(v) == 0 || len(v) > binary.Size(n) {
		return 0, false
	}
	for _, b := range v {
		n = n<<bitsPerByte | T(b)
	}
	return n, true
}

func readSeconds(v []byte) (time.Time, bool) {
	s, ok := readUint[uint32](v)
	return time.Unix(int64(s), 0).UTC(), ok
}

func readMilliseconds(v []byte) (time.Time, bool) {
	ms, ok := readUint[uint64](v)
	if !ok || ms > math.MaxInt64 {
		return time.Time{}, false
	}
	return time.UnixMilli(int64(ms)).UTC(), true
}
