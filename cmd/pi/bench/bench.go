package bench

import (
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
	benchCmd.AddCommand(RunCmd())
}
