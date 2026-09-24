package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/multicast"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// fakeMulticastListen records the request it was handed and, when release is
// set, holds the listen open until the job is cancelled — returning what it
// "heard", as the real listen does.
type fakeMulticastListen struct {
	got     multicast.ListenRequest
	release chan struct{}
	err     error
}

func (f *fakeMulticastListen) listen(
	ctx context.Context,
	req multicast.ListenRequest,
) (*multicast.ListenResult, error) {
	f.got = req
	if f.err != nil {
		return nil, f.err
	}
	if f.release != nil {
		<-ctx.Done()
	}
	return &multicast.ListenResult{Group: req.Group, Port: req.Port, Interface: req.Interface, Packets: 7}, nil
}

func submitMulticastListen(t *testing.T, fake *fakeMulticastListen, params string) (*jobs.Runner, string) {
	t.Helper()
	srv, runner := newJobsTestServer(t, jobs.Config{})
	srv.registerMulticastListenKind(fake.listen)
	id, err := runner.Submit(multicastListenJobKind, json.RawMessage(params))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return runner, id
}

func waitForState(t *testing.T, runner *jobs.Runner, id string, want jobs.State) jobs.Job {
	t.Helper()
	waitFor(t, "job "+string(want), func() bool {
		j, ok := runner.Get(id)
		return ok && j.State == want
	})
	j, _ := runner.Get(id)
	return j
}

func TestMulticastListenKindPassesTheRequestThrough(t *testing.T) {
	t.Parallel()

	fake := &fakeMulticastListen{}
	runner, id := submitMulticastListen(t, fake,
		`{"group":"239.1.1.1","port":5004,"interface":"en0","durationSeconds":5}`)
	j := waitForState(t, runner, id, jobs.StateSucceeded)

	want := multicast.ListenRequest{Group: "239.1.1.1", Port: 5004, Interface: "en0", DurationSeconds: 5}
	if fake.got != want {
		t.Fatalf("listen got %+v, want %+v", fake.got, want)
	}
	res, ok := j.Result.(*multicast.ListenResult)
	if !ok || res.Packets != 7 {
		t.Fatalf("result = %#v, want the listen's result", j.Result)
	}
}

// Stopping a listen is how an operator says "I have my answer", so a
// cancelled listen succeeds with what it heard rather than discarding it.
func TestCancelledMulticastListenKeepsItsResult(t *testing.T) {
	t.Parallel()

	fake := &fakeMulticastListen{release: make(chan struct{})}
	runner, id := submitMulticastListen(t, fake, `{"group":"239.1.1.1","port":5004,"interface":"en0"}`)
	waitForState(t, runner, id, jobs.StateRunning)
	if err := runner.Cancel(id); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	j := waitForState(t, runner, id, jobs.StateSucceeded)
	if res, ok := j.Result.(*multicast.ListenResult); !ok || res.Packets != 7 {
		t.Fatalf("result = %#v, want the partial listen", j.Result)
	}
}

func TestMulticastListenKindRejectsBadParams(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"missing":       ``,
		"not json":      `{`,
		"unknown field": `{"group":"239.1.1.1","port":5004,"interface":"en0","ttl":4}`,
	}
	for name, params := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeMulticastListen{}
			runner, id := submitMulticastListen(t, fake, params)
			if j := waitForState(t, runner, id, jobs.StateFailed); j.Err == "" {
				t.Fatal("failed job has an empty error")
			}
		})
	}
}

func TestMulticastListenKindReportsTheListenError(t *testing.T) {
	t.Parallel()

	fake := &fakeMulticastListen{err: multicast.ErrNotMulticast}
	runner, id := submitMulticastListen(t, fake, `{"group":"10.0.0.1","port":5004,"interface":"en0"}`)
	j := waitForState(t, runner, id, jobs.StateFailed)
	if j.Err != multicast.ErrNotMulticast.Error() {
		t.Fatalf("Err = %q, want %q", j.Err, multicast.ErrNotMulticast)
	}
}

// The production registration, not a test double: a unicast group is refused
// by the real listen before any socket opens, which proves the kind is wired
// to multicast.Listen rather than being an unknown kind.
func TestServerRegistersTheMulticastListenKind(t *testing.T) {
	t.Parallel()

	srv, runner := newJobsTestServer(t, jobs.Config{})
	srv.registerJobKinds()
	id, err := runner.Submit(multicastListenJobKind,
		json.RawMessage(`{"group":"10.0.0.1","port":5004,"interface":"lo0"}`))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	j := waitForState(t, runner, id, jobs.StateFailed)
	if !strings.Contains(j.Err, multicast.ErrNotMulticast.Error()) {
		t.Fatalf("Err = %q, want the listen's refusal", j.Err)
	}
}
