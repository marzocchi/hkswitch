package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"mrz.io/hkswitch/app"
	"os"
	"os/signal"
	"syscall"
)

var version = "v0.0.1"

var printConfCmd = &cobra.Command{
	Use:   "print-conf",
	Short: "Print configuration files for different init systems",
}

var printSystemdConfCmd = &cobra.Command{
	Use:   "systemd CONFIG_FILE",
	Short: "Print a systemd unit file for hkswitch and a config file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		env, err := cmd.Flags().GetStringArray("env")
		if err != nil {
			return err
		}
		return app.PrintConf("systemd", args[0], env)
	},
}

var printLaunchdConfCmd = &cobra.Command{
	Use:   "launchd CONFIG_FILE",
	Short: "Print a launchd plist file for hkswitch and a config file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		env, err := cmd.Flags().GetStringArray("env")
		if err != nil {
			return err
		}
		return app.PrintConf("launchd", args[0], env)
	},
}

var rootCmd = &cobra.Command{
	Version: version,
	Use:     fmt.Sprintf("%s CONFIG_FILE", os.Args[0]),
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.Serve(cmd.Context(), os.Stdout, os.Stderr, args[0])
	},
}

func init() {
	app.Version = version

	printLaunchdConfCmd.Flags().StringArrayP("env", "e", []string{}, "copy the value of an environment variable into the"+
		" generated config file")

	printSystemdConfCmd.Flags().StringArrayP("env", "e", []string{}, "copy the value of an environment variable into the"+
		" generated config file")

	printConfCmd.AddCommand(printSystemdConfCmd, printLaunchdConfCmd)
	rootCmd.AddCommand(printConfCmd)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
