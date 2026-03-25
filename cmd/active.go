package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/anandyadav3559/devflow/internal/storage"
	"github.com/spf13/cobra"
)

var activeCmd = &cobra.Command{
	Use:     "active",
	Short:   "List all currently running services",
	Example: `  devflow active`,
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := storage.GetAllActive()
		if err != nil {
			return fmt.Errorf("failed to read active state: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No services are currently running.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "WORKFLOW\tSERVICE\tPID\tTYPE\tSTARTED")
		fmt.Fprintln(w, "────────\t───────\t───\t────\t───────")
		for _, e := range entries {
			kind := "terminal"
			if e.Detached {
				kind = "detached"
			}
			started := e.StartedAt
			if len(started) > 16 {
				started = started[:16] // trim to minute precision
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				e.WorkflowName, e.ServiceName, e.PID, kind, started)
		}
		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(activeCmd)
}
