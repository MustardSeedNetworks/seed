package delivery_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/quotedprintable"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
)

// P-B7: the narrative the inbox shows travels with the alert on every channel.

type fakeNarrator struct {
	text narrative.Text
	ok   bool
	err  error
}

func (f fakeNarrator) Narrate(context.Context, *alerts.Alert) (narrative.Text, bool, error) {
	return f.text, f.ok, f.err
}

func testStory() narrative.Text {
	return narrative.Text{
		Summary: "Interface Gi1/0/1 on core-sw1 went down.",
		Evidence: []string{
			"ifOperStatus changed to down (ifIndex 3) at 2023-11-14T22:13:20Z.",
			"ifInErrors on Gi1/0/1 peaked at 12 per second in the 15 minutes before.",
		},
		NextCheck: "Check the cable, the optic and the far-end port of Gi1/0/1 on core-sw1.",
	}
}

// webhookCatcher is a receiver that keeps every body it is sent.
type webhookCatcher struct {
	mu     sync.Mutex
	bodies [][]byte
}

func (c *webhookCatcher) start(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		c.mu.Lock()
		c.bodies = append(c.bodies, body)
		c.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func (c *webhookCatcher) received() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.bodies)
}

func TestEveryChannelCarriesTheNarrative(t *testing.T) {
	hook := &webhookCatcher{}
	hookURL := hook.start(t)
	sink := &smtpSink{user: sinkUser, pass: sinkPass}
	sink.start(t)
	collector := newCollector(t, delivery.SyslogUDP)

	m := delivery.NewManager(nil, fakeNarrator{text: testStory(), ok: true}, quietLogger())
	defer m.Stop(context.Background())
	m.ApplyWebhook(delivery.WebhookConfig{URL: hookURL, Secret: signingKey})
	m.ApplyEmail(emailConfig(sink, nil))
	m.ApplySyslog(syslogConfig(collector, nil))

	m.Escalate(context.Background(), testAlert(),
		[]alerts.Channel{alerts.ChannelWebhook, alerts.ChannelEmail, alerts.ChannelSyslog})
	if !waitFor(t, func() bool {
		return len(hook.received()) == 1 && len(sink.received()) == 1 && len(collector.received()) == 1
	}) {
		t.Fatalf("received webhook %d, email %d, syslog %d; want one each",
			len(hook.received()), len(sink.received()), len(collector.received()))
	}

	t.Run("webhook", func(t *testing.T) { assertWebhookNarrative(t, hook.received()[0]) })
	t.Run("email", func(t *testing.T) { assertEmailNarrative(t, sink.received()[0].data) })
	t.Run("syslog", func(t *testing.T) { assertSyslogNarrative(t, collector.received()[0]) })
}

func assertWebhookNarrative(t *testing.T, body []byte) {
	t.Helper()
	story := testStory()
	var payload struct {
		Narrative *narrative.Text `json:"narrative"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if payload.Narrative == nil {
		t.Fatal("the envelope has no narrative")
	}
	if payload.Narrative.Summary != story.Summary || payload.Narrative.NextCheck != story.NextCheck ||
		!slices.Equal(payload.Narrative.Evidence, story.Evidence) {
		t.Errorf("narrative = %+v, want %+v", *payload.Narrative, story)
	}
}

func assertEmailNarrative(t *testing.T, data string) {
	t.Helper()
	story := testStory()
	_, encoded := readMessage(t, data)
	decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(encoded)))
	if err != nil {
		t.Fatalf("decode body: %v", err)
	}
	body := string(decoded)
	want := story.Summary + "\n\nEvidence:\n- " + story.Evidence[0] + "\n- " + story.Evidence[1] +
		"\n\nNext check: " + story.NextCheck + "\n"
	if !strings.Contains(body, want) {
		t.Errorf("body does not carry the narrative block %q:\n%s", want, body)
	}
	// The narrative is what the engineer reads first; the alert's own facts
	// follow it.
	if strings.Index(body, story.Summary) > strings.Index(body, "Severity:") {
		t.Errorf("the narrative follows the alert's facts:\n%s", body)
	}
}

func assertSyslogNarrative(t *testing.T, msg string) {
	t.Helper()
	story := testStory()
	got := parseSyslog(t, msg).pairs
	if got["summary"] != story.Summary {
		t.Errorf("summary = %q", got["summary"])
	}
	if got["next_check"] != story.NextCheck {
		t.Errorf("next_check = %q", got["next_check"])
	}
	if want := strings.Join(story.Evidence, "; "); got["evidence"] != want {
		t.Errorf("evidence = %q, want %q", got["evidence"], want)
	}
}

func TestSyslogPutsEvidenceLastSoATruncatedDatagramKeepsTheSummary(t *testing.T) {
	collector := newCollector(t, delivery.SyslogUDP)
	long := testStory()
	long.Evidence = []string{strings.Repeat("x", 4096)}
	cfg := syslogConfig(collector, nil)
	cfg.Narrator = fakeNarrator{text: long, ok: true}
	n, err := delivery.NewSyslog(cfg)
	if err != nil {
		t.Fatalf("NewSyslog: %v", err)
	}
	n.Start()
	defer n.Stop(context.Background())

	n.Deliver(context.Background(), testAlert())
	if !waitFor(t, func() bool { return len(collector.received()) == 1 }) {
		t.Fatal("no datagram arrived")
	}
	msg := collector.received()[0]
	for _, kept := range []string{"summary=", "next_check=", "title=", "message="} {
		if !strings.Contains(msg, kept) {
			t.Errorf("the truncated datagram lost %s", kept)
		}
	}
}

func TestAnAlertWithoutANarrativeIsStillDelivered(t *testing.T) {
	for name, narrator := range map[string]fakeNarrator{
		"nothing to say":    {},
		"cannot be read":    {err: errors.New("database is closed")},
		"read but rejected": {text: testStory(), err: errors.New("database is closed")},
	} {
		t.Run(name, func(t *testing.T) {
			hook := &webhookCatcher{}
			n, err := delivery.NewWebhook(delivery.WebhookConfig{
				URL: hook.start(t), Secret: signingKey, Narrator: narrator, Logger: quietLogger(),
			})
			if err != nil {
				t.Fatalf("NewWebhook: %v", err)
			}
			n.Start()
			defer n.Stop(context.Background())

			n.Deliver(context.Background(), testAlert())
			if !waitFor(t, func() bool { return len(hook.received()) == 1 }) {
				t.Fatal("the alert was not delivered")
			}
			var payload map[string]json.RawMessage
			if unmarshalErr := json.Unmarshal(hook.received()[0], &payload); unmarshalErr != nil {
				t.Fatalf("payload: %v", unmarshalErr)
			}
			if _, ok := payload["alert"]; !ok {
				t.Error("the envelope has no alert")
			}
			if raw, ok := payload["narrative"]; ok {
				t.Errorf("the envelope carries a narrative it has none of: %s", raw)
			}
		})
	}
}
