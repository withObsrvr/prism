package prism

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/withObsrvr/prism/internal/jev"
)

var jevCmd = &cobra.Command{
	Use:   "jev",
	Short: "Exercise Prism's TypeSafe Jev integration",
}

var jevAnalyzeCmd = &cobra.Command{
	Use:   "analyze <query>",
	Short: "Analyze an Ask Prism query with the versioned Jev registry",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.TrimSpace(strings.Join(args, " "))
		cfg, err := jevConfig()
		if err != nil {
			return err
		}
		client, err := jev.New(cfg)
		if err != nil {
			return err
		}
		analysis, err := client.AnalyzeQuery(cmd.Context(), query)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(analysis); err != nil {
			return fmt.Errorf("encode Jev analysis: %w", err)
		}
		return nil
	},
}

var jevEvaluateLedgersCmd = &cobra.Command{
	Use:   "evaluate-ledgers <shadow.jsonl>",
	Short: "Summarize recorded Ledger Jev shadow decisions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		file, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("open Jev ledger shadow records: %w", err)
		}
		defer file.Close()
		report, err := jev.EvaluateLedgerShadow(file)
		if err != nil {
			return err
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return fmt.Errorf("encode Jev ledger evaluation: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(jevCmd)
	jevCmd.AddCommand(jevAnalyzeCmd)
	jevCmd.AddCommand(jevEvaluateLedgersCmd)
}

func jevConfig() (jev.Config, error) {
	timeout, err := time.ParseDuration(viper.GetString("jev.timeout"))
	if err != nil {
		return jev.Config{}, fmt.Errorf("invalid jev.timeout: %w", err)
	}
	return jev.Config{
		Enabled:       viper.GetBool("jev.enabled"),
		BaseURL:       viper.GetString("jev.base_url"),
		APIKey:        viper.GetString("jev.api_key"),
		Model:         viper.GetString("jev.model"),
		Timeout:       timeout,
		ShadowLogPath: viper.GetString("jev.shadow_log_path"),
	}, nil
}
