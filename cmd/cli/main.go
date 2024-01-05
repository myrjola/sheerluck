package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/myrjola/sheerluck/cmd/cli/bundle"
	"github.com/myrjola/sheerluck/cmd/cli/img"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	if err := godotenv.Load(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rootCmd.AddGroup(img.Group)
	rootCmd.AddCommand(img.Generate)
	rootCmd.AddGroup(bundle.Group)
	rootCmd.AddCommand(bundle.CustomElements)
}

var rootCmd = &cobra.Command{
	Use:  "sheerluck-cli",
	Long: `Command line utilities for Sheerluck https://github.com/myrjola/sheerluck`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
