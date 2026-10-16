package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

func submitAutotest(t *testing.T, stages []autotest.Stage, params string) (*jobs.Runner, string) {
	t.Helper()
	_, runner := newJobsTestServer(t, jobs.Config{})
	handler := newAutotestHandler(func() string { return "eth1" }, stages)
	if err := runner.Register(autotestJobKind, handler); err != nil {
		t.Fatalf("Register: %v", err)
	}
	id, err := runner.Submit(autotestJobKind, json.RawMessage(params))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	return runner, id
}

func TestAutotestKindRunsEveryStageOnTheSelectedInterface(t *testing.T) {
	t.Parallel()

	var gotIface string
	var gotReq autotest.Spec
	stages := []autotest.Stage{
		{Name: autotest.StageLink, Run: func(context.Context, string, autotest.Spec) autotest.StageReport {
			return autotest.StageReport{Outcome: autotest.OutcomeFail, Reason: autotest.ReasonNoCarrier}
		}},
		{Name: autotest.StageVLAN, Run: func(_ context.Context, iface string, req autotest.Spec) autotest.StageReport {
			gotIface, gotReq = iface, req
			return autotest.StageReport{Outcome: autotest.OutcomePass}
		}},
	}
	runner, id := submitAutotest(t, stages, `{"expectedVlan":10}`)
	j := waitForState(t, runner, id, jobs.StateSucceeded)

	res, ok := j.Result.(*autotest.Report)
	if !ok {
		t.Fatalf("result = %#v, want an AutoTest result", j.Result)
	}
	if res.Verdict != autotest.OutcomeFail || len(res.Stages) != 2 || res.Stages[0].Stage != autotest.StageLink {
		t.Fatalf("result = %+v, want a failed run naming the link stage, with both stages", res)
	}
	if gotIface != "eth1" || gotReq.ExpectedVLAN != 10 {
		t.Fatalf("stage got %q %+v, want eth1 and expectedVlan 10", gotIface, gotReq)
	}
}

func TestAutotestKindRejectsUnknownParams(t *testing.T) {
	t.Parallel()

	runner, id := submitAutotest(t, nil, `{"bogus":true}`)
	j := waitForState(t, runner, id, jobs.StateFailed)
	if !strings.Contains(j.Err, "invalid autotest params") {
		t.Fatalf("error = %q, want an invalid-params error", j.Err)
	}
}
