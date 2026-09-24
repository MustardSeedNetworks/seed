package api

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/qos"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// fakeQoS records what each half was handed and, when hold is set, keeps the
// listen open until the job is cancelled, returning what it "read" as the
// real listen does.
type fakeQoS struct {
	sent  qos.SendRequest
	heard qos.ListenRequest
	hold  bool
	err   error
}

func (f *fakeQoS) send(_ context.Context, req qos.SendRequest) (*qos.SendResult, error) {
	f.sent = req
	if f.err != nil {
		return nil, f.err
	}
	return &qos.SendResult{Target: req.Target, Port: req.Port, RunID: "0000000000000001", Marked: true}, nil
}

func (f *fakeQoS) listen(ctx context.Context, req qos.ListenRequest) (*qos.ListenResult, error) {
	f.heard = req
	if f.err != nil {
		return nil, f.err
	}
	if f.hold {
		<-ctx.Done()
	}
	return &qos.ListenResult{Port: req.Port, Probes: 30, DSCPObserved: true}, nil
}

func submitQoS(t *testing.T, fake *fakeQoS, kind, params string) (*jobs.Runner, string) {
	t.Helper()
	srv, runner := newJobsTestServer(t, jobs.Config{})
	srv.registerQoSKinds(fake.send, fake.listen)
	id, err := runner.Submit(kind, json.RawMessage(params))
	if err != nil {
		t.Fatalf("Submit %s: %v", kind, err)
	}
	return runner, id
}

func TestQoSKindsPassTheRequestThrough(t *testing.T) {
	t.Parallel()

	fake := &fakeQoS{}
	runner, id := submitQoS(t, fake, qosSendJobKind,
		`{"target":"192.0.2.7","port":5004,"dscp":[46,0],"count":3}`)
	j := waitForState(t, runner, id, jobs.StateSucceeded)
	wantSend := qos.SendRequest{Target: "192.0.2.7", Port: 5004, DSCP: []int{46, 0}, Count: 3}
	if !reflect.DeepEqual(fake.sent, wantSend) {
		t.Fatalf("send got %+v, want %+v", fake.sent, wantSend)
	}
	if res, ok := j.Result.(*qos.SendResult); !ok || res.RunID != "0000000000000001" {
		t.Fatalf("send result = %#v", j.Result)
	}

	runner, id = submitQoS(t, fake, qosListenJobKind, `{"port":5004,"family":"ipv6","durationSeconds":20}`)
	j = waitForState(t, runner, id, jobs.StateSucceeded)
	if want := (qos.ListenRequest{Port: 5004, Family: "ipv6", DurationSeconds: 20}); fake.heard != want {
		t.Fatalf("listen got %+v, want %+v", fake.heard, want)
	}
	if res, ok := j.Result.(*qos.ListenResult); !ok || res.Probes != 30 {
		t.Fatalf("listen result = %#v", j.Result)
	}
}

// Stopping a listen is how the operator says the far side has finished, so a
// cancelled listen succeeds with what it read rather than discarding it.
func TestCancelledQoSListenKeepsItsResult(t *testing.T) {
	t.Parallel()

	runner, id := submitQoS(t, &fakeQoS{hold: true}, qosListenJobKind, `{"port":5004}`)
	waitForState(t, runner, id, jobs.StateRunning)
	if err := runner.Cancel(id); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	j := waitForState(t, runner, id, jobs.StateSucceeded)
	if res, ok := j.Result.(*qos.ListenResult); !ok || res.Probes != 30 {
		t.Fatalf("result = %#v, want the partial listen", j.Result)
	}
}

func TestQoSKindsRejectBadParams(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ kind, params string }{
		"send missing":         {qosSendJobKind, ``},
		"send not json":        {qosSendJobKind, `{`},
		"send unknown field":   {qosSendJobKind, `{"target":"192.0.2.7","port":5004,"ttl":4}`},
		"listen missing":       {qosListenJobKind, ``},
		"listen unknown field": {qosListenJobKind, `{"port":5004,"interface":"en0"}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runner, id := submitQoS(t, &fakeQoS{}, tc.kind, tc.params)
			if j := waitForState(t, runner, id, jobs.StateFailed); !strings.Contains(j.Err, "qos") {
				t.Fatalf("Err = %q, want it to name the kind", j.Err)
			}
		})
	}
}

func TestQoSKindsReportTheirError(t *testing.T) {
	t.Parallel()

	runner, id := submitQoS(t, &fakeQoS{err: qos.ErrTarget}, qosSendJobKind, `{"target":"239.1.1.1","port":5004}`)
	if j := waitForState(t, runner, id, jobs.StateFailed); j.Err != qos.ErrTarget.Error() {
		t.Fatalf("Err = %q, want %q", j.Err, qos.ErrTarget)
	}
}

// The production registration, not a test double: each half refuses a bad
// request before any socket opens, which proves the kinds are wired to
// qos.Send and qos.Listen rather than being unknown kinds.
func TestServerRegistersTheQoSKinds(t *testing.T) {
	t.Parallel()

	for kind, tc := range map[string]struct {
		params string
		want   error
	}{
		qosSendJobKind:   {`{"target":"239.1.1.1","port":5004}`, qos.ErrTarget},
		qosListenJobKind: {`{"port":0}`, qos.ErrPort},
	} {
		srv, runner := newJobsTestServer(t, jobs.Config{})
		srv.registerJobKinds()
		id, err := runner.Submit(kind, json.RawMessage(tc.params))
		if err != nil {
			t.Fatalf("Submit %s: %v", kind, err)
		}
		if j := waitForState(t, runner, id, jobs.StateFailed); !strings.Contains(j.Err, tc.want.Error()) {
			t.Fatalf("%s Err = %q, want %q", kind, j.Err, tc.want)
		}
	}
}
