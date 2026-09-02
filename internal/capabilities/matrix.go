package capabilities

// matrix.go renders the Platform Support Matrix that HARDWARE.md publishes.
//
// The point of generating it: the hand-written table said macOS ARP reading was
// Full while it returned nothing, and macOS Wi-Fi scanning was Full while it
// shelled a binary Apple had deleted. Neither could fail a build.
// scripts/check-hardware-matrix.sh regenerates this and diffs, so now they can.

import (
	"fmt"
	"strings"
)

// displayLevel is the word the document uses for a level.
func displayLevel() map[Level]string {
	return map[Level]string{
		LevelFull:    "Full",
		LevelPartial: "Partial",
		LevelLimited: "Limited",
		LevelNone:    "None",
	}
}

// displayPlatform is the column heading for a GOOS value.
func displayPlatform(goos string) string {
	switch goos {
	case "linux":
		return "Linux"
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	default:
		return goos
	}
}

// RenderMatrix returns the Markdown table plus the per-platform caveats, for
// every platform in the matrix.
func RenderMatrix() string {
	platforms := Platforms()
	words := displayLevel()
	rows := order()

	var b strings.Builder

	b.WriteString("| Feature |")
	for _, p := range platforms {
		fmt.Fprintf(&b, " %s |", displayPlatform(p))
	}
	b.WriteString("\n|---------|")
	for range platforms {
		b.WriteString("-------|")
	}
	b.WriteString("\n")

	for _, capability := range rows {
		fmt.Fprintf(&b, "| %s |", Title(capability))
		for _, p := range platforms {
			level, ok := LevelsFor(p)[capability]
			if !ok {
				level = LevelNone
			}
			fmt.Fprintf(&b, " %s |", words[level])
		}
		b.WriteString("\n")
	}

	b.WriteString("\n**Legend:**\n")
	b.WriteString("- **Full**: Complete feature support through standard OS APIs\n")
	b.WriteString("- **Partial**: Limited functionality through available APIs\n")
	b.WriteString("- **Limited**: Requires vendor-specific tools or drivers\n")
	b.WriteString("- **None**: Not available through standard APIs\n")

	renderNotes(&b, platforms, rows)

	return b.String()
}

// renderNotes writes the per-platform caveats — where a level that is not Full
// says why. Driven by the row order rather than map iteration, so the output is
// byte-identical every run and the drift gate cannot flap.
func renderNotes(b *strings.Builder, platforms []string, rows []Capability) {
	for _, p := range platforms {
		notes := NotesFor(p)
		if len(notes) == 0 {
			continue
		}
		fmt.Fprintf(b, "\n### %s caveats\n\n", displayPlatform(p))
		for _, capability := range rows {
			note, ok := notes[capability]
			if !ok {
				continue
			}
			fmt.Fprintf(b, "- **%s**: %s\n", Title(capability), note)
		}
	}
}
