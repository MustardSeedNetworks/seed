package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/krisarmstrong/seed/internal/config"
)

func TestFullCLIInitialization(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Verify all expected commands are registered
	expectedCommands := map[string]bool{
		"completion":      false,
		"credentials":     false,
		"export-config":   false,
		"install":         false,
		"mcp":             false,
		"reset-config":    false,
		"serve":           false,
		"setup-wizard":    false,
		"uninstall":       false,
		"validate-config": false,
		"version":         false,
	}

	for _, cmd := range state.rootCmd.Commands() {
		// Extract base command name (before any args like "[bash|zsh|...]")
		cmdName := cmd.Use
		for i, c := range cmdName {
			if c == ' ' || c == '[' {
				cmdName = cmdName[:i]
				break
			}
		}

		if _, ok := expectedCommands[cmdName]; ok {
			expectedCommands[cmdName] = true
		}
	}

	for cmd, found := range expectedCommands {
		if !found {
			t.Errorf("Command %q was not registered", cmd)
		}
	}
}

func TestCLIStateWithCustomConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "seed.yaml")

	// Create config file
	cfg := config.DefaultConfig()
	cfg.Server.Port = 9999
	cfg.Auth.DefaultUsername = "customuser"

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if writeErr := os.WriteFile(configPath, data, 0o600); writeErr != nil {
		t.Fatalf("Failed to write config: %v", writeErr)
	}

	// Create state with custom config
	state := &cliState{
		cfgFile: configPath,
		devMode: true,
	}

	if state.cfgFile != configPath {
		t.Errorf("cfgFile should be %q, got %q", configPath, state.cfgFile)
	}
	if !state.devMode {
		t.Error("devMode should be true")
	}
}

func TestAllCommandsHaveDescription(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Short == "" {
			t.Errorf("Command %q should have a Short description", cmd.Use)
		}
		if cmd.Long == "" {
			t.Errorf("Command %q should have a Long description", cmd.Use)
		}
	}
}

func TestAllCommandsHaveRunFunction(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Run == nil && cmd.RunE == nil {
			t.Errorf("Command %q should have a Run or RunE function", cmd.Use)
		}
	}
}

func TestPersistentFlagsAvailableToSubcommands(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Check that persistent flags are defined on root
	configFlag := state.rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Fatal("--config persistent flag not found on root")
	}

	devFlag := state.rootCmd.PersistentFlags().Lookup("dev")
	if devFlag == nil {
		t.Fatal("--dev persistent flag not found on root")
	}

	proxyFlag := state.rootCmd.PersistentFlags().Lookup("trusted-proxies")
	if proxyFlag == nil {
		t.Fatal("--trusted-proxies persistent flag not found on root")
	}

	// Verify flags are inherited by subcommands
	for _, cmd := range state.rootCmd.Commands() {
		inheritedConfig := cmd.InheritedFlags().Lookup("config")
		if inheritedConfig == nil {
			t.Errorf("Command %q should inherit --config flag", cmd.Use)
		}

		inheritedDev := cmd.InheritedFlags().Lookup("dev")
		if inheritedDev == nil {
			t.Errorf("Command %q should inherit --dev flag", cmd.Use)
		}
	}
}

func TestRootCommandVersion(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	if state.rootCmd.Version == "" {
		t.Error("Root command should have version set")
	}
}

func TestRootCommandUse(t *testing.T) {
	t.Parallel()

	state := newCLIState()

	if state.rootCmd.Use != "seed" {
		t.Errorf("Root command Use should be 'seed', got %q", state.rootCmd.Use)
	}
}

func TestCommandHelpOutput(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Capture help output
	buf := new(bytes.Buffer)
	state.rootCmd.SetOut(buf)
	state.rootCmd.SetErr(buf)

	// Generate help (don't execute)
	state.rootCmd.SetArgs([]string{"--help"})

	// Just verify help generation doesn't panic
	// Actually running Execute would try to run the serve command
	helpFunc := state.rootCmd.HelpFunc()
	helpFunc(state.rootCmd, []string{})

	output := buf.String()
	if output == "" {
		t.Error("Help output should not be empty")
	}

	// Verify help mentions key commands
	if !containsSubstring(output, "serve") {
		t.Error("Help should mention 'serve' command")
	}
	if !containsSubstring(output, "version") {
		t.Error("Help should mention 'version' command")
	}
}

func TestSubcommandHelpOutput(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Test help for each subcommand
	for _, cmd := range state.rootCmd.Commands() {
		t.Run(cmd.Use, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			helpFunc := cmd.HelpFunc()
			helpFunc(cmd, []string{})

			output := buf.String()
			if output == "" {
				t.Errorf("Help output for %q should not be empty", cmd.Use)
			}
		})
	}
}

func TestCompletionCommandIsUsable(t *testing.T) {
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

	// Verify it can generate completions for all shells
	shells := []string{"bash", "zsh", "fish", "powershell"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			buf := new(bytes.Buffer)

			var err error
			switch shell {
			case "bash":
				err = state.rootCmd.GenBashCompletion(buf)
			case "zsh":
				err = state.rootCmd.GenZshCompletion(buf)
			case "fish":
				err = state.rootCmd.GenFishCompletion(buf, true)
			case "powershell":
				err = state.rootCmd.GenPowerShellCompletionWithDesc(buf)
			}

			if err != nil {
				t.Errorf("Failed to generate %s completion: %v", shell, err)
			}
			if buf.Len() == 0 {
				t.Errorf("%s completion output should not be empty", shell)
			}
		})
	}
}

func TestNoCommandConflicts(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Check for duplicate command names
	cmdNames := make(map[string]bool)
	for _, cmd := range state.rootCmd.Commands() {
		name := cmd.Use
		// Extract base name
		for i, c := range name {
			if c == ' ' || c == '[' {
				name = name[:i]
				break
			}
		}

		if cmdNames[name] {
			t.Errorf("Duplicate command name: %q", name)
		}
		cmdNames[name] = true
	}
}

func TestFlagsDoNotConflict(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initCommands(state)

	// Check each command's flags don't conflict with persistent flags
	persistentShorthands := make(map[string]bool)
	state.rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Shorthand != "" {
			persistentShorthands[f.Shorthand] = true
		}
	})

	for _, cmd := range state.rootCmd.Commands() {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Shorthand != "" && persistentShorthands[f.Shorthand] {
				t.Errorf("Command %q flag %q conflicts with persistent flag shorthand -%s",
					cmd.Use, f.Name, f.Shorthand)
			}
		})
	}
}
