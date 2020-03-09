package backend

import (
	"context"
	"testing"
	"time"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/task/backend/scheduler"
)

var (
	mockTaskID     = influxdb.ID(1)
	mockTimeNow    = time.Now()
	mockTimeNowStr = time.Now().Format(time.RFC3339Nano)
)

func (m MockTaskService) UpdateTask(_ context.Context, id influxdb.ID, _ influxdb.TaskUpdate) (*influxdb.Task, error) {
	return &influxdb.Task{ID: id, UpdatedAt: mockTimeNowStr}, nil
}

type MockTaskService struct{}

func Test_Schedulable_Task_Service(t *testing.T) {

	for _, test := range []struct {
		name string
		task *influxdb.Task
	}{
		{
			name: "Create New Schedulable Task Service",
			task: taskOne,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ts := MockTaskService{}

			schedulableService := NewSchedulableTaskService(ts)

			err := schedulableService.UpdateLastScheduled(context.Background(), scheduler.ID(mockTaskID), mockTimeNow)
			if err != nil {
				t.Fatalf("expected nil error, got: %v", err)
			}
		})
	}
}
