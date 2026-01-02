package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/paths"
)

var validateCmd = &cobra.Command{
	Use:   "validate-config",
	Short: "Validate configuration file",
	Long:  "Validate the configuration file against the schema without starting the server",
	Run:   runValidate,
}

func initValidateCmd() {
	validateCmd.Flags().Bool("strict", false, "Treat warnings as errors")
	validateCmd.Flags().Bool("json", false, "Output results as JSON")
	rootCmd.AddCommand(validateCmd)
}

// ValidationResult holds the validation output.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Path     string   `json:"path"`
}

func runValidate(cmd *cobra.Command, _ []string) {
	strict, err := cmd.Flags().GetBool("strict")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting strict flag: %v\n", err)
		os.Exit(1)
	}
	outputAsJSON, err := cmd.Flags().GetBool("json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting json flag: %v\n", err)
		os.Exit(1)
	}

	// Resolve config path using paths package
	configPath := paths.ResolveConfigPath(cfgFile, paths.ModeAuto)

	result := ValidationResult{
		Valid: true,
		Path:  configPath,
	}

	// Check file exists
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("config file not found: %s", configPath))
		outputResult(result, outputAsJSON)
		os.Exit(1)
	}

	// Load config
	cfg, loadErr := config.Load(configPath)
	if loadErr != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to parse config: %v", loadErr))
		outputResult(result, outputAsJSON)
		os.Exit(1)
	}

	// Run validation
	if validateErr := cfg.Validate(); validateErr != nil {
		result.Valid = false
		result.Errors = append(result.Errors, validateErr.Error())
	}

	// Add warnings for missing optional configs
	result.Warnings = checkConfigWarnings(cfg)

	outputResult(result, outputAsJSON)

	if !result.Valid || (strict && len(result.Warnings) > 0) {
		os.Exit(1)
	}
}

func checkConfigWarnings(cfg *config.Config) []string {
	var warnings []string

	if cfg.Interface.Default == "" {
		warnings = append(warnings, "no default interface configured (will use auto-detection)")
	}
	if cfg.Auth.JWTSecret == "" {
		warnings = append(warnings, "JWT secret not set (will be auto-generated)")
	}
	if len(cfg.SNMP.Communities) == 0 {
		warnings = append(warnings, "no SNMP communities configured")
	}

	return warnings
}

func outputResult(result ValidationResult, asJSON bool) {
	if asJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling result: %v\n", err)
			return
		}
		fmt.Fprintln(os.Stdout, string(data))
		return
	}

	// Human-readable output
	fmt.Fprintf(os.Stdout, "Config: %s\n", result.Path)
	if result.Valid {
		fmt.Fprintln(os.Stdout, "Status: VALID")
	} else {
		fmt.Fprintln(os.Stdout, "Status: INVALID")
	}

	for _, e := range result.Errors {
		fmt.Fprintf(os.Stdout, "  ERROR: %s\n", e)
	}
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stdout, "  WARNING: %s\n", w)
	}
}
