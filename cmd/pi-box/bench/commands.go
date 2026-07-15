package bench

import (
	"fmt"

	"github.com/pi-sandbox/pi/pkg/bench"
	"github.com/spf13/cobra"
)

var mode string
var jsonFlag bool

// RunCmd returns the bench run command.
func RunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags]",
		Short: "Run benchmarks",
		Run: func(cmd *cobra.Command, _ []string) {
			if mode == "" {
				mode = "fast"
			}
			if !bench.IsSupportedMode(mode) {
				fmt.Printf("error: unsupported mode %q. supported: %v\n", mode, bench.SupportedModes())
				return
			}

			suite := bench.NewSuite(mode, "base")
			for _, b := range bench.All() {
				suite.AddBenchmark(b)
			}

			stats := suite.ComputeStats()

			if jsonFlag {
				suite.PrintJSON(stats)
				return
			}

			suite.PrintResults(stats)
			fmt.Println()
			fmt.Println(bench.FormatResults(stats))
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "fast", "Benchmark mode: fast, compat, secure, microvm")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}
