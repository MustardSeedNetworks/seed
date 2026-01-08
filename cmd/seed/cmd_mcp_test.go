package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestInitMCPCmd(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initMCPCmd(state)

	// Find the mcp command
	var mcpCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "mcp" {
			mcpCmd = cmd
			break
		}
	}

	if mcpCmd == nil {
		t.Fatal("mcp command not found")
	}

	// Verify command properties
	if mcpCmd.Short == "" {
		t.Error("mcp command should have a Short description")
	}
	if mcpCmd.Long == "" {
		t.Error("mcp command should have a Long description")
	}
}

func TestMCPCmdHasRunFunction(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initMCPCmd(state)

	// Find the mcp command
	var mcpCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "mcp" {
			mcpCmd = cmd
			break
		}
	}

	if mcpCmd == nil {
		t.Fatal("mcp command not found")
	}

	if mcpCmd.Run == nil {
		t.Error("mcp command should have a Run function")
	}
}

func TestMCPCmdLongDescriptionContent(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initMCPCmd(state)

	// Find the mcp command
	var mcpCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "mcp" {
			mcpCmd = cmd
			break
		}
	}

	if mcpCmd == nil {
		t.Fatal("mcp command not found")
	}

	// Check that long description mentions key features
	expectedContent := []string{
		"MCP",
		"Model Context Protocol",
		"stdio",
		"AI",
	}

	for _, content := range expectedContent {
		if !containsSubstring(mcpCmd.Long, content) {
			t.Errorf("mcp Long description should mention %q", content)
		}
	}
}

func TestMCPCmdNoSubcommands(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initMCPCmd(state)

	// Find the mcp command
	var mcpCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "mcp" {
			mcpCmd = cmd
			break
		}
	}

	if mcpCmd == nil {
		t.Fatal("mcp command not found")
	}

	// MCP should have no subcommands
	if len(mcpCmd.Commands()) != 0 {
		t.Errorf("mcp command should have no subcommands, got %d", len(mcpCmd.Commands()))
	}
}

func TestMCPCmdExampleInLongDescription(t *testing.T) {
	t.Parallel()

	state := newCLIState()
	initMCPCmd(state)

	// Find the mcp command
	var mcpCmd *cobra.Command
	for _, cmd := range state.rootCmd.Commands() {
		if cmd.Use == "mcp" {
			mcpCmd = cmd
			break
		}
	}

	if mcpCmd == nil {
		t.Fatal("mcp command not found")
	}

	// Check that the long description includes an example configuration
	if !containsSubstring(mcpCmd.Long, "mcpServers") {
		t.Error("mcp Long description should include mcpServers example")
	}
	if !containsSubstring(mcpCmd.Long, "mcp.json") {
		t.Error("mcp Long description should reference mcp.json")
	}
}
