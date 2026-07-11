package main

import (
	"encoding/json"
	"fmt"

	"github.com/sercanarga/pcileechgen/internal/donor/session"
	"github.com/sercanarga/pcileechgen/internal/replay"
	"github.com/spf13/cobra"
)

var validateInitDonor string
var validateInitEmulator string
var validateInitJSON bool

var validateInitCmd = &cobra.Command{Use: "validate-init", Short: "Compare donor and emulator initialization sessions", RunE: func(cmd *cobra.Command, args []string) error {
	donorCapture, err := session.LoadCapture(validateInitDonor)
	if err != nil {
		return fmt.Errorf("load donor session: %w", err)
	}
	emulatorCapture, err := session.LoadCapture(validateInitEmulator)
	if err != nil {
		return fmt.Errorf("load emulator session: %w", err)
	}
	result := replay.Compare(donorCapture, emulatorCapture)
	if validateInitJSON {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(result); err != nil {
			return err
		}
	} else if result.Divergence != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "matched %d accesses; first divergence: %s\n", result.Matched, result.Divergence.Reason)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "all %d initialization accesses match\n", result.Matched)
	}
	if !result.OK() {
		return fmt.Errorf("initialization replay diverged")
	}
	return nil
}}

func init() {
	validateInitCmd.Flags().StringVar(&validateInitDonor, "donor", "", "donor session directory")
	validateInitCmd.Flags().StringVar(&validateInitEmulator, "emulator", "", "emulator session directory")
	validateInitCmd.Flags().BoolVar(&validateInitJSON, "json", false, "emit JSON")
	_ = validateInitCmd.MarkFlagRequired("donor")
	_ = validateInitCmd.MarkFlagRequired("emulator")
	rootCmd.AddCommand(validateInitCmd)
}
