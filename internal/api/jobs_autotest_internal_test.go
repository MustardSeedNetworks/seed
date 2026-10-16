package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

func submitAutotest(t *testing.T, stages []autotest.Stage, params string) (*jobs.Runner, string) {
	t.Helper()
	return submitAutotestKeeping(t, stages, params, func(context.Context, *autotest.Report) error { return nil })
}

func submitAutotestKeeping(
	t *testing.T,
	stages []autotest.Stage,
	params string,
	keep func(context.Context, *autotest.Report) error,
) (*jobs.Runner, string) {
	t.Helper()
	_, runner := newJobsTestServer(t, jobs.Config{})
	handler := newAutotestHandler(func() string { return "eth1" }, stages, keep)
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

// P-E2: a run carries its labels and is kept before the job reports it, under
// the id the job hands back.
func TestAutotestKindKeepsTheLabelledRun(t *testing.T) {
	t.Parallel()

	var kept *autotest.Report
	stages := []autotest.Stage{fixedStage(autotest.StageLink, autotest.OutcomePass)}
	runner, id := submitAutotestKeeping(t, stages, `{"jackLabel":"3F-R301-P07","assetTag":"PP-0042"}`,
		func(_ context.Context, r *autotest.Report) error {
			kept = r
			return nil
		})
	j := waitForState(t, runner, id, jobs.StateSucceeded)

	res, ok := j.Result.(*autotest.Report)
	if !ok || kept != res {
		t.Fatalf("result = %#v, kept = %#v, want the kept record", j.Result, kept)
	}
	if res.ID == "" || res.JackLabel != "3F-R301-P07" || res.AssetTag != "PP-0042" {
		t.Fatalf("record = %+v, want an id and both labels", res)
	}
}

func TestAutotestKindFailsWhenTheRunCannotBeKept(t *testing.T) {
	t.Parallel()

	stages := []autotest.Stage{fixedStage(autotest.StageLink, autotest.OutcomePass)}
	runner, id := submitAutotestKeeping(t, stages, `{}`,
		func(context.Context, *autotest.Report) error { return autotest.ErrUnavailable })
	j := waitForState(t, runner, id, jobs.StateFailed)
	if !strings.Contains(j.Err, "keep autotest result") {
		t.Fatalf("error = %q, want the keep failure", j.Err)
	}
}

func TestAutotestKindRejectsABadLabel(t *testing.T) {
	t.Parallel()

	ran := false
	stages := []autotest.Stage{
		{Name: autotest.StageLink, Run: func(context.Context, string, autotest.Spec) autotest.StageReport {
			ran = true
			return autotest.StageReport{Outcome: autotest.OutcomePass}
		}},
	}
	runner, id := submitAutotestKeeping(t, stages, `{"jackLabel":"a\u0007b"}`,
		func(context.Context, *autotest.Report) error { return errors.New("must not be kept") })
	j := waitForState(t, runner, id, jobs.StateFailed)
	if ran || !strings.Contains(j.Err, "invalid autotest params") {
		t.Fatalf("ran = %v, error = %q, want the label refused before any stage", ran, j.Err)
	}
}

func fixedStage(name autotest.StageName, o autotest.Outcome) autotest.Stage {
	return autotest.Stage{Name: name, Run: func(context.Context, string, autotest.Spec) autotest.StageReport {
		return autotest.StageReport{Outcome: o}
	}}
}
