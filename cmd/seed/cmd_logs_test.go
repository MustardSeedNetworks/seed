package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLogsCommand(t *testing.T) {
	tests := []struct {
		name       string
		log        *string
		args       []string
		wantCode   int
		wantLines  []string
		wantStderr string
	}{
		{
			name:      "last lines by default",
			log:       new("one\ntwo\nthree\n"),
			wantLines: []string{"one", "two", "three"},
		},
		{
			name:      "lines limits the tail",
			log:       new("one\ntwo\nthree\n"),
			args:      []string{"--lines", "2"},
			wantLines: []string{"two", "three"},
		},
		{
			name:      "lines zero prints nothing",
			log:       new("one\n"),
			args:      []string{"--lines", "0"},
			wantLines: nil,
		},
		{
			name:       "negative lines is an error",
			log:        new("one\n"),
			args:       []string{"--lines", "-3"},
			wantCode:   1,
			wantStderr: "must not be negative",
		},
		{
			name:       "missing log names the workdir flag",
			wantCode:   1,
			wantStderr: "--workdir",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := newOpsInstall(t, nil)
			if tc.log != nil {
				in.writeFile(in.logFile, *tc.log)
			}
			code, stdout, stderr := in.run(append([]string{"logs"}, tc.args...)...)
			if code != tc.wantCode {
				t.Fatalf("exit = %d, want %d (stderr %q)", code, tc.wantCode, stderr)
			}
			if tc.wantStderr != "" && !strings.Contains(stderr, tc.wantStderr) {
				t.Errorf("stderr lacks %q: %s", tc.wantStderr, stderr)
			}
			if tc.wantCode == 0 {
				got := strings.Fields(stdout)
				if !slices.Equal(got, tc.wantLines) {
					t.Errorf("lines = %q, want %q", got, tc.wantLines)
				}
			}
		})
	}
}

// TestLogsResolveRelativePathsAgainstTheWorkDir covers the default the daemon
// really writes: logs/seed.log under its working directory, not anywhere under
// /var/log.
func TestLogsResolveRelativePathsAgainstTheWorkDir(t *testing.T) {
	in := newOpsInstall(t, func(doc map[string]any) { setPath(doc, "logging.file", "") })
	in.writeFile(filepath.Join(in.workDir, "logs", "seed.log"), "daemon default log\n")
	code, stdout, stderr := in.run("logs")
	if code != 0 || !strings.Contains(stdout, "daemon default log") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *lockedBuffer) waitFor(t *testing.T, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(b.String(), want) {
		if time.Now().After(deadline) {
			t.Fatalf("followed output never contained %q; got %q", want, b.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func appendTo(t *testing.T, path, content string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestFollowLogSurvivesRotation proves -f keeps following when the file grows,
// is truncated in place, and is rotated away and recreated.
func TestFollowLogSurvivesRotation(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "seed.log")
	appendTo(t, logPath, "before follow\n")

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open("seed.log")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if _, err = file.Seek(0, io.SeekEnd); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	out := new(lockedBuffer)
	done := make(chan error, 1)
	go func() { done <- followLog(ctx, root, "seed.log", file, out, 2*time.Millisecond) }()

	steps := []struct {
		name  string
		act   func()
		wants string
	}{
		{"append", func() { appendTo(t, logPath, "grown line\n") }, "grown line"},
		{"truncate in place", func() {
			if err = os.WriteFile(logPath, []byte("after truncate\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "after truncate"},
		{"rotate and recreate", func() {
			if err = os.Rename(logPath, logPath+".1"); err != nil {
				t.Fatal(err)
			}
			appendTo(t, logPath, "after rotation\n")
		}, "after rotation"},
		{"append to the new file", func() { appendTo(t, logPath, "still following\n") }, "still following"},
	}
	for _, step := range steps {
		step.act()
		out.waitFor(t, step.wants)
	}
	if strings.Contains(out.String(), "before follow") {
		t.Error("follow replayed what was already shown")
	}
	cancel()
	if err = <-done; err != nil {
		t.Errorf("followLog = %v, want a clean stop on cancel", err)
	}
}
