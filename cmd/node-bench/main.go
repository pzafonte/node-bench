// Command node-bench benchmarks Bitcoin node implementations by building from
// source, running timed IBD trials, and storing structured JSON results.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := buildRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "node-bench",
		Short: "Bitcoin node IBD benchmarking harness",
	}
	root.AddCommand(cmdRun(), cmdBench(), cmdAnalyze(), cmdCompare(), cmdHistory(), cmdProfile())
	return root
}
