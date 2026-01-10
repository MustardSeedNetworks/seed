package main

import (
	"bytes"
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionCmdExists(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Find the completion command
	var completionCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if hasPrefix(cmd.Use, "completion") {
			completionCmd = cmd
			break
		}
	}

	if completionCmd == nil {
		t.Fatal("completion command not found")
	}
}

func TestCompletionCmdValidArgs(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	expectedShells := []string{"bash", "zsh", "fish", "powershell"}

	for i, shell := range expectedShells {
		if i >= len(state.completionCmd.ValidArgs) {
			t.Errorf("Missing valid arg for shell: %s", shell)
			continue
		}

		if !slices.Contains(state.completionCmd.ValidArgs, shell) {
			t.Errorf("completionCmd.ValidArgs should contain %q", shell)
		}
	}
}

func TestCompletionCmdDisablesFlagsInUseLine(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	if !state.completionCmd.DisableFlagsInUseLine {
		t.Error("completionCmd should have DisableFlagsInUseLine set to true")
	}
}

func TestCompletionCmdHasRun(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	if state.completionCmd.Run == nil {
		t.Error("completionCmd should have a Run function")
	}
}

func TestCompletionCmdArgs(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Find completion command
	var completionCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if hasPrefix(cmd.Use, "completion") {
			completionCmd = cmd
			break
		}
	}

	if completionCmd == nil {
		t.Fatal("completion command not found")
	}

	// Test that Args validator is set
	if completionCmd.Args == nil {
		t.Error("completionCmd should have Args validator")
	}
}

func TestCompletionCmdShortDescription(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	if state.completionCmd.Short == "" {
		t.Error("completionCmd should have a Short description")
	}

	if !containsSubstring(state.completionCmd.Short, "completion") {
		t.Error("completionCmd Short description should mention 'completion'")
	}
}

func TestCompletionCmdLongDescriptionContainsInstructions(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	// Long description should contain instructions for all shells
	expectedContent := []string{
		"Bash",
		"Zsh",
		"Fish",
		"PowerShell",
	}

	for _, content := range expectedContent {
		if !containsSubstring(state.completionCmd.Long, content) {
			t.Errorf("completionCmd Long description should mention %q", content)
		}
	}
}

func TestCompletionBashGeneration(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	buf := new(bytes.Buffer)
	err := state.rootCmd.GenBashCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate bash completion: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Bash completion output should not be empty")
	}

	// Check for bash-specific content
	if !containsSubstring(output, "bash") || !containsSubstring(output, "complete") {
		t.Error("Bash completion should contain bash-specific content")
	}
}

func TestCompletionZshGeneration(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	buf := new(bytes.Buffer)
	err := state.rootCmd.GenZshCompletion(buf)
	if err != nil {
		t.Fatalf("Failed to generate zsh completion: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Zsh completion output should not be empty")
	}
}

func TestCompletionFishGeneration(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	buf := new(bytes.Buffer)
	err := state.rootCmd.GenFishCompletion(buf, true)
	if err != nil {
		t.Fatalf("Failed to generate fish completion: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Fish completion output should not be empty")
	}
}

func TestCompletionPowerShellGeneration(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	buf := new(bytes.Buffer)
	err := state.rootCmd.GenPowerShellCompletionWithDesc(buf)
	if err != nil {
		t.Fatalf("Failed to generate PowerShell completion: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("PowerShell completion output should not be empty")
	}
}

func TestCompletionCmdArgsValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "bash is valid",
			args:    []string{"bash"},
			wantErr: false,
		},
		{
			name:    "zsh is valid",
			args:    []string{"zsh"},
			wantErr: false,
		},
		{
			name:    "fish is valid",
			args:    []string{"fish"},
			wantErr: false,
		},
		{
			name:    "powershell is valid",
			args:    []string{"powershell"},
			wantErr: false,
		},
		{
			name:    "invalid shell",
			args:    []string{"invalid"},
			wantErr: true,
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "too many args",
			args:    []string{"bash", "zsh"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := newCLIState()
			initCommands(state)

			// Find completion command
			var completionCmd *cobra.Command
			for _, cmd := range state.rootCmd.Commands() {
				if hasPrefix(cmd.Use, "completion") {
					completionCmd = cmd
					break
				}
			}

			if completionCmd == nil {
				t.Fatal("completion command not found")
			}

			// Validate args
			err := completionCmd.Args(completionCmd, tc.args)
			if (err != nil) != tc.wantErr {
				t.Errorf("Args validation: got error %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
