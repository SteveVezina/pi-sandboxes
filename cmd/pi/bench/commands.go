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
		Run: func(*cobra.Command, []string) {
			if mode == "" {
				mode = "fast"
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
	cmd.Flags().StringVar(&mode, "mode", "fast", "Benchmark mode: fast, compat")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}
