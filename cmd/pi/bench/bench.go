package bench

import (
	"fmt"
	"os"

	"github.com/pi-sandbox/pi/cmd/pi/cli"
	"github.com/spf13/cobra"
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Benchmark suite",
	Long:  `Run benchmarks to measure sandbox performance.`,
}

// Command is exported for initialization.
var Command = benchCmd

func init() {
	cli.AddCommand(benchCmd)
	benchCmd.AddCommand(&cobra.Command{
		Use:   "run [flags]",
		Short: "Run benchmarks",
		Run: func(*cobra.Command, []string) {
			fmt.Fprintln(os.Stderr, "stub: bench run")
			os.Exit(1)
		},
	})
}
