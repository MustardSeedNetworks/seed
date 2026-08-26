package vuln

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// corpusUsableFloor is the number of the 125 real sysDescr strings in
// testdata/sysdescr-corpus.txt that must yield both a product and a version.
//
// A floor rather than an exact figure: adding a vendor should be allowed to
// raise it, and this asserting equality would make every improvement a test
// failure. But it must not fall. An empty version makes ScanDevice give up with
// "Unable to extract product/version information" before it queries any CVE
// feed, so a drop here means devices silently stop being scanned — which is how
// this sat at 78 without anyone noticing.
const corpusUsableFloor = 107

// TestCorpusExtractionRate measures product/version extraction against real
// device strings rather than invented ones. The corpus is sysDescr values
// harvested from the SNMP walk corpus — Cisco, Juniper, HP, Aruba, Brocade,
// Extreme, Fortinet, Palo Alto, 3Com, H3C, Force10, VMware and Linux hosts,
// including the malformed and versionless ones.
func TestCorpusExtractionRate(t *testing.T) {
	lines := readCorpus(t)

	var usable, noProduct, noVersion int
	for _, line := range lines {
		product, version := parseSNMPSysDescr(line)
		switch {
		case product == "" && version == "":
			noProduct++
			noVersion++
		case product == "":
			noProduct++
		case version == "":
			noVersion++
		default:
			usable++
		}
	}

	t.Logf("corpus=%d usable=%d no-product=%d no-version=%d",
		len(lines), usable, noProduct, noVersion)

	if usable < corpusUsableFloor {
		t.Errorf("only %d of %d corpus strings yield a product and a version, "+
			"want at least %d — devices below this line are never scanned for CVEs",
			usable, len(lines), corpusUsableFloor)
	}
}

// TestCorpusVersionsAreWellFormed pins that whatever the extractor returns is
// shaped like a version. A malformed one is worse than none: it is fed straight
// into a CPE lookup, which then matches nothing while the scan reports success.
func TestCorpusVersionsAreWellFormed(t *testing.T) {
	for _, line := range readCorpus(t) {
		_, version := parseSNMPSysDescr(line)
		if version == "" {
			continue
		}
		switch {
		case strings.HasSuffix(version, "."):
			t.Errorf("version %q ends in a dot\n  from: %.80s", version, line)
		case !strings.ContainsAny(version, "0123456789"):
			t.Errorf("version %q contains no digit\n  from: %.80s", version, line)
		case !strings.Contains(version, "."):
			t.Errorf("version %q has no dot\n  from: %.80s", version, line)
		case len(version) > maxVersionTokenLen:
			t.Errorf("version %q is %d chars, over the %d cap\n  from: %.80s",
				version, len(version), maxVersionTokenLen, line)
		case strings.ContainsAny(version, " \t,;\""):
			t.Errorf("version %q carries punctuation or whitespace\n  from: %.80s",
				version, line)
		}
	}
}

// TestCorpusNeverPanics is the cheapest guarantee worth having about a parser
// fed unauthenticated network data: every entry point survives every string.
func TestCorpusNeverPanics(t *testing.T) {
	for _, line := range readCorpus(t) {
		parseSNMPSysDescr(line)
		parseLLDPDescription(line)
		parseServiceBanner(line)
		parseHTTPServer(line)
		parseOSGuess(line)
		parseEDPVersion(line)
		parseCDPVersion(line, line)
		extractVersion(line)
	}
}

func readCorpus(t *testing.T) []string {
	t.Helper()

	file, err := os.Open("testdata/sysdescr-corpus.txt")
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	t.Cleanup(func() { _ = file.Close() })

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			lines = append(lines, line)
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		t.Fatalf("read corpus: %v", scanErr)
	}
	if len(lines) == 0 {
		t.Fatal("corpus is empty; these tests would pass without asserting anything")
	}
	return lines
}
