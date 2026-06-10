package lifecycle_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/update"
	"github.com/MustardSeedNetworks/seed/internal/update/lifecycle"
)

// --- fake ----------------------------------------------------------------------

type fakeUpdater struct {
	available  bool
	info       *update.UpdateInfo
	status     update.UpdateStatus
	lastCheck  time.Time
	downloaded bool
	config     update.UpdateConfig
	err        error

	downloadCalled bool
	applyCalled    bool
	rollbackCalled bool
	setConfig      *update.UpdateConfig
}

func (f *fakeUpdater) Available() bool { return f.available }
func (f *fakeUpdater) Check(_ context.Context) (*update.UpdateInfo, error) {
	return f.info, f.err
}
func (f *fakeUpdater) Info() *update.UpdateInfo    { return f.info }
func (f *fakeUpdater) Status() update.UpdateStatus { return f.status }
func (f *fakeUpdater) LastCheck() time.Time        { return f.lastCheck }
func (f *fakeUpdater) Downloaded() bool            { return f.downloaded }
func (f *fakeUpdater) Download(_ context.Context) error {
	f.downloadCalled = true
	return f.err
}

func (f *fakeUpdater) Apply(_ context.Context) error {
	f.applyCalled = true
	return f.err
}

func (f *fakeUpdater) Rollback() error {
	f.rollbackCalled = true
	return f.err
}
func (f *fakeUpdater) Config() update.UpdateConfig { return f.config }
func (f *fakeUpdater) SetConfig(cfg update.UpdateConfig) {
	f.setConfig = &cfg
}

// newService wires one fake as both the Updater and ConfigStore ports,
// mirroring the production adapter.
func newService(f *fakeUpdater) *lifecycle.Service {
	return lifecycle.NewService(f, f)
}

// --- tests ---------------------------------------------------------------------

func TestUnavailable(t *testing.T) {
	t.Parallel()
	svc := newService(&fakeUpdater{})

	tests := []struct {
		name string
		call func() error
	}{
		{"check", func() error { _, err := svc.Check(context.Background()); return err }},
		{"status", func() error { _, err := svc.Status(); return err }},
		{"info", func() error { _, err := svc.Info(); return err }},
		{"download", func() error { return svc.Download(context.Background()) }},
		{"apply", func() error { return svc.Apply(context.Background()) }},
		{"rollback", svc.Rollback},
		{"config", func() error { _, err := svc.Config(); return err }},
		{"configure", func() error { _, err := svc.Configure(lifecycle.ConfigPatch{}); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.call(); !errors.Is(err, lifecycle.ErrUnavailable) {
				t.Fatalf("want ErrUnavailable, got %v", err)
			}
		})
	}
}

func TestStatus(t *testing.T) {
	t.Parallel()
	checked := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		info           *update.UpdateInfo
		downloaded     bool
		wantReady      bool
		wantRequiresAc bool
	}{
		{"no check yet", nil, false, false, false},
		{"update available, not downloaded", &update.UpdateInfo{Available: true}, false, false, true},
		{"update available, downloaded", &update.UpdateInfo{Available: true}, true, true, false},
		{"no update available", &update.UpdateInfo{Available: false}, false, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := newService(&fakeUpdater{
				available:  true,
				info:       tc.info,
				downloaded: tc.downloaded,
				lastCheck:  checked,
				status:     update.UpdateStatus{State: update.StateIdle},
			})
			got, err := svc.Status()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Ready != tc.wantReady {
				t.Errorf("Ready = %v, want %v", got.Ready, tc.wantReady)
			}
			if got.RequiresAction != tc.wantRequiresAc {
				t.Errorf("RequiresAction = %v, want %v", got.RequiresAction, tc.wantRequiresAc)
			}
			if !got.LastCheck.Equal(checked) {
				t.Errorf("LastCheck = %v, want %v", got.LastCheck, checked)
			}
			if got.Status.State != update.StateIdle {
				t.Errorf("Status.State = %v, want idle", got.Status.State)
			}
		})
	}
}

func TestInfoDefaultsBeforeFirstCheck(t *testing.T) {
	t.Parallel()
	svc := newService(&fakeUpdater{available: true})

	info, err := svc.Info()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil || info.Available {
		t.Fatalf("want zero-value info with Available=false, got %+v", info)
	}
}

func TestInfoPassesThrough(t *testing.T) {
	t.Parallel()
	want := &update.UpdateInfo{Available: true, LatestVersion: "v9.9.9"}
	svc := newService(&fakeUpdater{available: true, info: want})

	info, err := svc.Info()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != want {
		t.Fatalf("want pass-through info, got %+v", info)
	}
}

func TestDownload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		info       *update.UpdateInfo
		wantErr    error
		wantCalled bool
	}{
		{"no check yet", nil, lifecycle.ErrNoUpdate, false},
		{"no update available", &update.UpdateInfo{Available: false}, lifecycle.ErrNoUpdate, false},
		{"update available", &update.UpdateInfo{Available: true}, nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeUpdater{available: true, info: tc.info}
			err := newService(f).Download(context.Background())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if f.downloadCalled != tc.wantCalled {
				t.Errorf("downloadCalled = %v, want %v", f.downloadCalled, tc.wantCalled)
			}
		})
	}
}

func TestApply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		downloaded bool
		wantErr    error
		wantCalled bool
	}{
		{"not downloaded", false, lifecycle.ErrNotDownloaded, false},
		{"downloaded", true, nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeUpdater{available: true, downloaded: tc.downloaded}
			err := newService(f).Apply(context.Background())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if f.applyCalled != tc.wantCalled {
				t.Errorf("applyCalled = %v, want %v", f.applyCalled, tc.wantCalled)
			}
		})
	}
}

func TestRollback(t *testing.T) {
	t.Parallel()
	f := &fakeUpdater{available: true}
	if err := newService(f).Rollback(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.rollbackCalled {
		t.Error("rollback not invoked on the port")
	}
}

func TestConfigure(t *testing.T) {
	t.Parallel()
	base := update.UpdateConfig{
		Enabled:       true,
		CheckInterval: 24 * time.Hour,
		GitHubOwner:   "MustardSeedNetworks",
		GitHubRepo:    "seed",
	}

	tests := []struct {
		name     string
		patch    lifecycle.ConfigPatch
		wantErr  error
		wantCfg  func(update.UpdateConfig) bool
		wantSets bool
	}{
		{
			name:     "empty patch keeps config",
			patch:    lifecycle.ConfigPatch{},
			wantCfg:  func(c update.UpdateConfig) bool { return c == base },
			wantSets: true,
		},
		{
			name: "boolean fields applied",
			patch: lifecycle.ConfigPatch{
				Enabled:           new(false),
				AutoDownload:      new(true),
				AutoApply:         new(true),
				IncludePrerelease: new(true),
			},
			wantCfg: func(c update.UpdateConfig) bool {
				return !c.Enabled && c.AutoDownload && c.AutoApply && c.IncludePrerelease
			},
			wantSets: true,
		},
		{
			name:     "check interval applied",
			patch:    lifecycle.ConfigPatch{CheckInterval: new("1h")},
			wantCfg:  func(c update.UpdateConfig) bool { return c.CheckInterval == time.Hour },
			wantSets: true,
		},
		{
			name:    "unparsable interval rejected",
			patch:   lifecycle.ConfigPatch{CheckInterval: new("daily")},
			wantErr: lifecycle.ErrInvalidInterval,
		},
		{
			name:    "sub-minute interval rejected",
			patch:   lifecycle.ConfigPatch{CheckInterval: new("5s")},
			wantErr: lifecycle.ErrInvalidInterval,
		},
		{
			name:    "negative interval rejected",
			patch:   lifecycle.ConfigPatch{CheckInterval: new("-1h")},
			wantErr: lifecycle.ErrInvalidInterval,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeUpdater{available: true, config: base}
			got, err := newService(f).Configure(tc.patch)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
			if tc.wantErr != nil {
				if f.setConfig != nil {
					t.Fatal("rejected patch must not reach SetConfig")
				}
				return
			}
			if !tc.wantCfg(got) {
				t.Errorf("unexpected resulting config: %+v", got)
			}
			if tc.wantSets && (f.setConfig == nil || *f.setConfig != got) {
				t.Errorf("SetConfig got %+v, want %+v", f.setConfig, got)
			}
		})
	}
}
