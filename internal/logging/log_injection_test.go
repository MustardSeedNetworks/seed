package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// forgedEntry is what an attacker would inject: a newline, then a line that
// looks like a legitimate log record. If it ever appears at the start of an
// output line, the attacker has written a log entry.
const forgedEntry = "harmless\nlevel=ERROR msg=\"admin password changed\" user=root"

// TestLogValuesCannotForgeAnEntry pins the property that makes seed's standing
// CodeQL go/log-injection alerts false positives: every log line leaves the
// process through an encoder that escapes control characters, so a value
// carrying a newline cannot become a line of its own.
//
// Handler-agnostic on purpose. The claim is about the pipeline rather than one
// handler, so this drives both formats InitLogger can build (logger.go) under
// the redacting wrapper it always applies, and asserts on the bytes written.
func TestLogValuesCannotForgeAnEntry(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func(buf *bytes.Buffer) slog.Handler
	}{
		{"text", func(buf *bytes.Buffer) slog.Handler {
			return logging.NewRedactingHandler(slog.NewTextHandler(buf, nil))
		}},
		{"json", func(buf *bytes.Buffer) slog.Handler {
			return logging.NewRedactingHandler(slog.NewJSONHandler(buf, nil))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			slog.New(tc.make(&buf)).Info("http request", "path", forgedEntry)

			if got := strings.Count(strings.TrimRight(buf.String(), "\n"), "\n"); got != 0 {
				t.Fatalf("one call produced %d extra lines; the value broke out:\n%s",
					got, buf.String())
			}
			if !strings.Contains(buf.String(), `\n`) {
				t.Errorf("output carries no escaped newline, so the attacker value "+
					"was dropped rather than encoded — this test proves nothing:\n%s",
					buf.String())
			}
		})
	}
}

// TestStreamedLogValuesCannotForgeAnEntry covers the second exit: records also
// reach the UI log viewer through the broadcaster, a separate encoding path the
// test above does not touch.
func TestStreamedLogValuesCannotForgeAnEntry(t *testing.T) {
	b := logging.NewLogBroadcaster(8)
	defer b.Stop()

	h := logging.NewStreamingHandler(slog.NewTextHandler(&bytes.Buffer{}, nil), b)
	slog.New(h).Info("http request", "path", forgedEntry)

	recent := b.GetRecentLogs(1)
	if len(recent) != 1 {
		t.Fatalf("broadcaster holds %d entries, want 1", len(recent))
	}
	raw, err := json.Marshal(recent[0])
	if err != nil {
		t.Fatalf("marshal streamed entry: %v", err)
	}
	if bytes.ContainsAny(raw, "\n\r") {
		t.Errorf("streamed entry carries a raw control character:\n%s", raw)
	}
	if !bytes.Contains(raw, []byte(`\n`)) {
		t.Errorf("streamed entry carries no escaped newline, so the attacker "+
			"value was dropped rather than encoded:\n%s", raw)
	}
}
