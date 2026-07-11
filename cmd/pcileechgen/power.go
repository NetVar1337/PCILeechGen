package main

import (
	"encoding/json"
	"fmt"
	"time"

	powerctl "github.com/sercanarga/pcileechgen/internal/power"
	"github.com/sercanarga/pcileechgen/internal/safety"
	"github.com/spf13/cobra"
)

var powerBDF string
var powerJSON bool

var powerCmd = &cobra.Command{Use: "power", Short: "Inspect or recover donor power state"}

func renderPower(cmd *cobra.Command, status powerctl.Status) error {
	if powerJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(status)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "PCI=%s runtime=%s control=%s\n", status.PCIState, status.RuntimeStatus, status.Control)
	return nil
}

var powerStatusCmd = &cobra.Command{Use: "status", RunE: func(cmd *cobra.Command, args []string) error {
	status, err := powerctl.Read("", powerBDF)
	if err != nil {
		return err
	}
	return renderPower(cmd, status)
}}

var powerWakeCmd = &cobra.Command{Use: "wake", RunE: func(cmd *cobra.Command, args []string) error {
	if err := safety.Check("", "", powerBDF); err != nil {
		return err
	}
	status, err := powerctl.Wake("", powerBDF, 2*time.Second)
	if err != nil {
		return err
	}
	return renderPower(cmd, status)
}}

func init() {
	for _, command := range []*cobra.Command{powerStatusCmd, powerWakeCmd} {
		command.Flags().StringVar(&powerBDF, "bdf", "", "donor PCI BDF")
		command.Flags().BoolVar(&powerJSON, "json", false, "emit JSON")
		_ = command.MarkFlagRequired("bdf")
	}
	powerCmd.AddCommand(powerStatusCmd, powerWakeCmd)
	rootCmd.AddCommand(powerCmd)
}
