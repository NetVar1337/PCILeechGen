package main

import (
	"fmt"

	"github.com/sercanarga/pcileechgen/internal/safety"
	"github.com/sercanarga/pcileechgen/internal/vfiomgr"
	"github.com/spf13/cobra"
)

var vfioStatePath string
var vfioBDF string

var vfioCmd = &cobra.Command{Use: "vfio", Short: "Prepare or restore complete VFIO groups"}

var vfioPrepareCmd = &cobra.Command{
	Use: "prepare", Short: "Bind a complete IOMMU group to vfio-pci transactionally",
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := vfiomgr.New("").Prepare(vfioBDF, func(member string) error {
			return safety.Check("", "", member)
		})
		if err != nil {
			return err
		}
		if err := vfiomgr.Save(vfioStatePath, state); err != nil {
			if restoreErr := vfiomgr.New("").Restore(state); restoreErr != nil {
				return fmt.Errorf("save VFIO restoration state: %w; rollback failed: %w", err, restoreErr)
			}
			return fmt.Errorf("save VFIO restoration state: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "prepared IOMMU group %s; restoration state: %s\n", state.Group, vfioStatePath)
		return nil
	},
}

var vfioRestoreCmd = &cobra.Command{
	Use: "restore", Short: "Restore drivers from a VFIO state file",
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := vfiomgr.Load(vfioStatePath)
		if err != nil {
			return err
		}
		if err := vfiomgr.New("").Restore(state); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "restored IOMMU group drivers")
		return nil
	},
}

func init() {
	vfioPrepareCmd.Flags().StringVar(&vfioBDF, "bdf", "", "donor PCI BDF")
	vfioPrepareCmd.Flags().StringVar(&vfioStatePath, "state", "pcileech_vfio_state.json", "restoration state path")
	_ = vfioPrepareCmd.MarkFlagRequired("bdf")
	vfioRestoreCmd.Flags().StringVar(&vfioStatePath, "state", "pcileech_vfio_state.json", "restoration state path")
	vfioCmd.AddCommand(vfioPrepareCmd, vfioRestoreCmd)
	rootCmd.AddCommand(vfioCmd)
}
