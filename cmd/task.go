package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage pull request tasks",
}

var taskListPRID int

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks on a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		tasks, err := client.Tasks(ws, r, taskListPRID).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(tasks, func() { render.TaskList(tasks) })
	},
}

var (
	taskCompletePRID int
	taskCompleteIDs  []int
)

var taskCompleteCmd = &cobra.Command{
	Use:   "complete",
	Short: "Mark one or more tasks as complete",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		res := client.Tasks(ws, r, taskCompletePRID)
		for _, id := range taskCompleteIDs {
			if err := res.SetState(context.Background(), id, true); err != nil {
				return fmt.Errorf("complete task %d: %w", id, err)
			}
		}
		return printOutput(map[string]any{"completed": taskCompleteIDs}, func() {
			fmt.Printf("Completed %d task(s).\n", len(taskCompleteIDs))
		})
	},
}

var (
	taskReopenPRID int
	taskReopenIDs  []int
)

var taskReopenCmd = &cobra.Command{
	Use:   "reopen",
	Short: "Reopen one or more tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		res := client.Tasks(ws, r, taskReopenPRID)
		for _, id := range taskReopenIDs {
			if err := res.SetState(context.Background(), id, false); err != nil {
				return fmt.Errorf("reopen task %d: %w", id, err)
			}
		}
		return printOutput(map[string]any{"reopened": taskReopenIDs}, func() {
			fmt.Printf("Reopened %d task(s).\n", len(taskReopenIDs))
		})
	},
}

func init() {
	prCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskListCmd, taskCompleteCmd, taskReopenCmd)

	taskListCmd.Flags().IntVarP(&taskListPRID, "pr-id", "p", 0, "pull request ID")
	taskListCmd.MarkFlagRequired("pr-id")

	taskCompleteCmd.Flags().IntVarP(&taskCompletePRID, "pr-id", "p", 0, "pull request ID")
	taskCompleteCmd.Flags().IntSliceVar(&taskCompleteIDs, "task-id", nil,
		"task ID to complete (repeatable: --task-id 1 --task-id 2)")
	taskCompleteCmd.MarkFlagRequired("pr-id")
	taskCompleteCmd.MarkFlagRequired("task-id")

	taskReopenCmd.Flags().IntVarP(&taskReopenPRID, "pr-id", "p", 0, "pull request ID")
	taskReopenCmd.Flags().IntSliceVar(&taskReopenIDs, "task-id", nil,
		"task ID to reopen (repeatable: --task-id 1 --task-id 2)")
	taskReopenCmd.MarkFlagRequired("pr-id")
	taskReopenCmd.MarkFlagRequired("task-id")
}
