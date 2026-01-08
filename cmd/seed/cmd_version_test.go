package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/version"
)

func TestInitVersionCmd(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

func TestVersionCmdHasRunFunction(t *testing.T) {
	t.Parallel()

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

	if versionCmd.Run == nil {
		t.Error("version command should have a Run function")
	}
}
