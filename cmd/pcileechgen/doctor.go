package main

import (
	"encoding/json"
	"fmt"

	"github.com/sercanarga/pcileechgen/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorBDF string
var doctorVivado string
var doctorJSON bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose host, VFIO, donor, BAR, and Vivado readiness",
	RunE: func(cmd *cobra.Command, args []string) error {
		report := doctor.Run(doctor.Options{BDF: doctorBDF, VivadoPath: doctorVivado})
		if doctorJSON {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(report); err != nil {
				return fmt.Errorf("encode doctor report: %w", err)
			}
		} else {
			fmt.Fprint(cmd.OutOrStdout(), doctor.Format(report))
		}
		if report.HasFailures() {
			return fmt.Errorf("doctor found blocking conditions")
		}
		return nil
	},
}

func init() {
	doctorCmd.Flags().StringVar(&doctorBDF, "bdf", "", "donor PCI BDF to diagnose")
	doctorCmd.Flags().StringVar(&doctorVivado, "vivado", "", "Vivado executable to preflight")
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "emit machine-readable diagnostics")
	rootCmd.AddCommand(doctorCmd)
}
