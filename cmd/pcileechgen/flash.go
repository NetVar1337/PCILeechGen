package main

import (
	"fmt"
	"path/filepath"

	"github.com/sercanarga/pcileechgen/internal/firmware/output"
	"github.com/spf13/cobra"
)

var flashManifest string
var flashOutputDir string

var flashCmd = &cobra.Command{Use: "flash", Short: "Verify flash artifacts"}
var flashVerifyCmd = &cobra.Command{Use: "verify", Short: "Verify bitstream and artifacts against a build manifest", RunE: func(cmd *cobra.Command, args []string) error {
	verification, err := output.VerifyManifest(flashManifest, flashOutputDir)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), verification.Summary())
	if !verification.OK() {
		return fmt.Errorf("flash artifact verification failed")
	}
	manifest, err := output.LoadManifest(flashManifest)
	if err != nil {
		return err
	}
	foundImage := false
	for _, entry := range manifest.Files {
		ext := filepath.Ext(entry.Name)
		if ext == ".bin" || ext == ".bit" {
			foundImage = true
			break
		}
	}
	if !foundImage {
		return fmt.Errorf("manifest contains no .bin or .bit image")
	}
	return nil
}}

func init() {
	flashVerifyCmd.Flags().StringVar(&flashManifest, "manifest", "", "build manifest path")
	flashVerifyCmd.Flags().StringVar(&flashOutputDir, "output-dir", ".", "artifact directory")
	_ = flashVerifyCmd.MarkFlagRequired("manifest")
	flashCmd.AddCommand(flashVerifyCmd)
	rootCmd.AddCommand(flashCmd)
}
