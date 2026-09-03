package capabilities

import (
	"strings"
	"testing"
)

// The rendered matrix is what HARDWARE.md publishes, so its shape is part of
// the contract with the drift gate.
func TestRenderMatrixCoversEveryRowAndPlatform(t *testing.T) {
	t.Parallel()

	rendered := RenderMatrix()

	for _, platform := range Platforms() {
		if !strings.Contains(rendered, displayPlatform(platform)) {
			t.Errorf("rendered matrix has no %s column", displayPlatform(platform))
		}
	}
	for _, capability := range order() {
		if !strings.Contains(rendered, Title(capability)) {
			t.Errorf("rendered matrix has no row for %q", Title(capability))
		}
	}
	for _, legend := range []string{"Full", "Partial", "Limited", "None"} {
		if !strings.Contains(rendered, "**"+legend+"**") {
			t.Errorf("rendered matrix has no legend entry for %q", legend)
		}
	}
}

// Rendering twice must produce the same bytes, or the drift gate reports a
// change on every run and stops meaning anything. Map iteration order is the
// usual way this breaks.
func TestRenderMatrixIsDeterministic(t *testing.T) {
	t.Parallel()

	first := RenderMatrix()
	for range 20 {
		if RenderMatrix() != first {
			t.Fatal("RenderMatrix is not deterministic; the drift gate would flap")
		}
	}
}

// Every caveat has to reach the document — a note nobody renders is a note
// nobody reads.
func TestRenderMatrixIncludesEveryNote(t *testing.T) {
	t.Parallel()

	// Compared with whitespace collapsed: a long note is wrapped onto
	// continuation lines to stay inside MD013, so it is not present verbatim.
	// What matters is that the words reached the document.
	rendered := strings.Join(strings.Fields(RenderMatrix()), " ")
	for _, platform := range Platforms() {
		for _, note := range notesByPlatform()[platform] {
			if !strings.Contains(rendered, strings.Join(strings.Fields(note), " ")) {
				t.Errorf("%s note %q is not in the rendered document", platform, note)
			}
		}
	}
}

// A note long enough to trip MD013 must be wrapped by the generator, not left
// for a human to notice. Otherwise the linter and the drift gate fight: one
// wants the line shorter, the other wants it exactly as generated.
func TestRenderMatrixWrapsLongNotes(t *testing.T) {
	t.Parallel()

	for i, line := range strings.Split(RenderMatrix(), "\n") {
		if len(line) > maxLineLength {
			t.Errorf("line %d is %d characters, over the %d limit:\n%s",
				i+1, len(line), maxLineLength, line)
		}
	}
}
