package backend

import (
	"context"
	"time"

	"github.com/influxdata/influxdb"
	"go.uber.org/zap"
)

var now = func() time.Time {
	return time.Now().UTC()
}

// TaskService is a type on which tasks can be listed
type TaskService interface {
	FindTasks(context.Context, influxdb.TaskFilter) ([]*influxdb.Task, int, error)
	UpdateTask(context.Context, influxdb.ID, influxdb.TaskUpdate) (*influxdb.Task, error)
}

// Coordinator is a type with a single method which
// is called when a task has been created
type Coordinator interface {
	TaskCreated(context.Context, *influxdb.Task) error
}

// NotifyCoordinatorOfExisting lists all tasks by the provided task service and for
// each task it calls the provided coordinators task created method
func NotifyCoordinatorOfExisting(ctx context.Context, ts TaskService, coord Coordinator, logger *zap.Logger) error {
	// If we missed a Create Action
	tasks, _, err := ts.FindTasks(ctx, influxdb.TaskFilter{})
	if err != nil {
		return err
	}

	latestCompleted := now().Format(time.RFC3339)
	for len(tasks) > 0 {
		for _, task := range tasks {
			if task.Status != string(TaskActive) {
				continue
			}

			task, err := ts.UpdateTask(context.Background(), task.ID, influxdb.TaskUpdate{
				LatestCompleted: &latestCompleted,
			})
			if err != nil {
				logger.Error("failed to set latestCompleted", zap.Error(err))
				continue
			}

			coord.TaskCreated(ctx, task)
		}

		tasks, _, err = ts.FindTasks(ctx, influxdb.TaskFilter{
			After: &tasks[len(tasks)-1].ID,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
