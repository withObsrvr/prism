package prism

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd is the base command. All subcommands attach here.
var rootCmd = &cobra.Command{
	Use:   "prism",
	Short: "Prism — Soroban-first block explorer for Stellar",
	Long: `Prism is a block explorer for the Stellar network with
full Soroban smart contract support, human-readable
transactions via SEP-41/SEP-50, and CAP-67 event streams.

Built by Obsrvr. Powered by Obsrvr Lake.`,
}

// Execute runs the root command. Called from main.go.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./prism.yaml)")

	// Global flags available to all subcommands.
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("prism")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.prism")
		viper.AddConfigPath("/etc/prism")
	}

	// Environment variables override config file.
	// PRISM_PORT=4000 overrides port, etc.
	viper.SetEnvPrefix("PRISM")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config: %s\n", err)
		}
	}
}
