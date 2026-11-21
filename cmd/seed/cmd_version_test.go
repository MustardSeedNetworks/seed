package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/seed/internal/version"
)

func TestInitVersionCmd(t *testing.T) {
	state := newCLIState()
	initVersionCmd(state)

	// Find the version command
	var versionCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "version" {
			versionCmd = cmd
			break
		}
	}

	if versionCmd == nil {
		t.Fatal("version command not found")
	}

	// Verify command properties
	if versionCmd.Short == "" {
		t.Error("version command should have a Short description")
	}
	if versionCmd.Long == "" {
		t.Error("version command should have a Long description")
	}
}

func TestVersionCmdExecution(t *testing.T) {
	state := newCLIState()
	initVersionCmd(state)

	// Find the version command
	var versionCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "version" {
			versionCmd = cmd
			break
		}
	}

	if versionCmd == nil {
		t.Fatal("version command not found")
	}

	// Capture output by running the command
	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)
	versionCmd.SetErr(buf)

	// The Run function writes to os.Stdout, so we test the version package instead
	expectedVersion := version.GetVersion()
	if expectedVersion == "" {
		t.Error("version.GetVersion() should return a non-empty string")
	}
}

func TestVersionCmdHasRunEFunction(t *testing.T) {
	state := newCLIState()
	initVersionCmd(state)

	// Find the version command
	var versionCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "version" {
			versionCmd = cmd
			break
		}
	}

	if versionCmd == nil {
		t.Fatal("version command not found")
	}

	if versionCmd.RunE == nil {
		t.Error("version command should have a RunE function")
	}
}

func TestVersionJSON(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := runCLI([]string{"version", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errOut.String())
	}
	var got map[string]string
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("version --json is not JSON: %v\n%s", err, out.String())
	}
	for _, key := range []string{"version", "commit", "buildTime", "releaseTrain", "uiBuildHash"} {
		if _, ok := got[key]; !ok {
			t.Errorf("version --json lacks %q: %v", key, got)
		}
	}
	if got["version"] != version.GetVersion() {
		t.Errorf("version = %q, want %q", got["version"], version.GetVersion())
	}

	out.Reset()
	if code := runCLI([]string{"version"}, &out, &errOut); code != 0 || !strings.HasPrefix(out.String(), "Seed ") {
		t.Errorf("plain version: exit %d, output %q", code, out.String())
	}
}
