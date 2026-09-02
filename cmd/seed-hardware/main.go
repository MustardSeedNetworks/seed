// Command seed-hardware regenerates HARDWARE.md's Platform Support Matrix from
// internal/capabilities.
//
// The matrix used to be hand-written and was wrong in both directions: macOS
// ARP reading listed Full while it returned nothing, macOS Wi-Fi scanning
// listed Full while it shelled a binary Apple had removed. Neither could fail a
// build. Now the document is a rendering of the code, and
// scripts/check-hardware-matrix.sh fails CI when the two disagree.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/capabilities"
)

const (
	beginMarker = "<!-- BEGIN GENERATED MATRIX -->"
	endMarker   = "<!-- END GENERATED MATRIX -->"
)

func main() {
	path := flag.String("file", "HARDWARE.md", "document to update")
	check := flag.Bool("check", false, "exit non-zero if the document is out of date, writing nothing")
	flag.Parse()

	if err := run(*path, *check); err != nil {
		fmt.Fprintln(os.Stderr, "seed-hardware:", err)
		os.Exit(1)
	}
}

func run(path string, check bool) error {
	current, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	updated, err := replaceSection(string(current), capabilities.RenderMatrix())
	if err != nil {
		return err
	}

	if updated == string(current) {
		return nil
	}
	if check {
		return fmt.Errorf(
			"%s is out of date with internal/capabilities; run `make hardware-matrix`", path)
	}

	return os.WriteFile(filepath.Clean(path), []byte(updated), 0o644) //nolint:gosec // a published document
}

// replaceSection swaps the content between the generated-matrix markers.
func replaceSection(document, generated string) (string, error) {
	begin := strings.Index(document, beginMarker)
	end := strings.Index(document, endMarker)
	if begin < 0 || end < 0 || end < begin {
		return "", fmt.Errorf("markers %q and %q not found in order", beginMarker, endMarker)
	}

	return document[:begin+len(beginMarker)] + "\n\n" + generated + "\n" + document[end:], nil
}
