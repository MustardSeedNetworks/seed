package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/license"
)

func initLicenseCmd(state *cliState) {
	licenseCmd := &cobra.Command{
		Use:   "license",
		Short: "Manage license activation",
		Long: `The license command handles offline license activation and status
for Seed. Without a license, Seed runs in the Free tier (basic
diagnostics only). Paid tiers:

  • Starter ($299/yr) — multi-interface, scheduled monitoring,
    basic Wi-Fi visibility, basic compliance, CSV/JSON export
  • Pro ($999/yr)    — everything in Starter plus Wi-Fi roam
    analysis, association forensics, AirMapper baseline diff,
    anomaly detection, path analysis, live telemetry, advanced
    compliance, scheduled PDF reports, multi-site, white-label,
    and REST API access

A 14-day trial of the full Pro tier is available without a key.`,
	}

	licenseCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show current license status",
		Run:   func(_ *cobra.Command, _ []string) { runLicenseStatus(state) },
	})

	activateCmd := &cobra.Command{
		Use:   "activate",
		Short: "Activate a license key",
		Run:   func(cmd *cobra.Command, _ []string) { runLicenseActivate(cmd) },
	}
	activateCmd.Flags().StringP("key", "k", "", "License key to activate (XXXX-XXXX-XXXX-XXXX)")
	_ = activateCmd.MarkFlagRequired("key")
	licenseCmd.AddCommand(activateCmd)

	licenseCmd.AddCommand(&cobra.Command{
		Use:   "deactivate",
		Short: "Remove the current license from this device",
		Run:   func(_ *cobra.Command, _ []string) { runLicenseDeactivate(state) },
	})

	licenseCmd.AddCommand(&cobra.Command{
		Use:   "trial",
		Short: "Start the 14-day Pro trial",
		Run:   func(_ *cobra.Command, _ []string) { runLicenseTrial(state) },
	})

	state.rootCmd.AddCommand(licenseCmd)
}

func runLicenseStatus(_ *cliState) {
	mgr, err := license.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	state := mgr.GetState()
	if state == nil {
		_, _ = fmt.Fprintln(os.Stdout, "Tier:        Free (no license activated)")
		_, _ = fmt.Fprintln(os.Stdout, "Run `seed license trial` to start a 14-day Pro trial,")
		_, _ = fmt.Fprintln(os.Stdout, "or `seed license activate -k <KEY>` to enter a key.")
		return
	}

	if state.IsTrialMode {
		remaining := mgr.TrialDaysRemaining()
		_, _ = fmt.Fprintln(os.Stdout, "Tier:        Trial (Pro features)")
		_, _ = fmt.Fprintf(os.Stdout, "Days left:   %d of %d\n", remaining, license.TrialDays)
		if remaining <= 0 {
			_, _ = fmt.Fprintln(os.Stdout, "Trial expired. Run `seed license activate -k <KEY>` to continue.")
		}
		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "Tier:        %s\n", state.Tier)
	_, _ = fmt.Fprintf(os.Stdout, "Key:         %s\n", license.FormatKey(state.LicenseKey))
	_, _ = fmt.Fprintf(os.Stdout, "Activated:   %s\n", state.ActivatedAt.Format("2006-01-02"))
	_, _ = fmt.Fprintf(os.Stdout, "Expires:     %s\n", state.ExpiresAt.Format("2006-01-02"))
	_, _ = fmt.Fprintf(os.Stdout, "Device:      %s\n", state.DeviceHash)
	if len(state.Features) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Features:    %d unlocked\n", len(state.Features))
	}
}

func runLicenseActivate(cmd *cobra.Command) {
	key, err := cmd.Flags().GetString("key")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading --key flag: %v\n", err)
		os.Exit(1)
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "Error: --key is required")
		os.Exit(1)
	}

	mgr, mgrErr := license.NewManager()
	if mgrErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", mgrErr)
		os.Exit(1)
	}

	res := mgr.Activate(key)
	if !res.Success {
		fmt.Fprintf(os.Stderr, "Activation failed: %s\n", res.Message)
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout, res.Message)
}

func runLicenseDeactivate(_ *cliState) {
	mgr, err := license.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if deactErr := mgr.Deactivate(); deactErr != nil {
		fmt.Fprintf(os.Stderr, "Deactivation failed: %v\n", deactErr)
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout, "License removed. Seed will run in the Free tier.")
}

func runLicenseTrial(_ *cliState) {
	mgr, err := license.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	res := mgr.StartTrial()
	if !res.Success {
		fmt.Fprintf(os.Stderr, "Trial failed: %s\n", res.Message)
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout, res.Message)
}
