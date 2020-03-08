package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	platform "github.com/influxdata/influxdb"
	pcontext "github.com/influxdata/influxdb/context"
	"github.com/influxdata/influxdb/inmem"
	"github.com/influxdata/influxdb/mock"
	_ "github.com/influxdata/influxdb/query/builtin"
	"github.com/influxdata/influxdb/task/backend"
	platformtesting "github.com/influxdata/influxdb/testing"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// NewMockTaskBackend returns a TaskBackend with mock services.
func NewMockTaskBackend(t *testing.T) *TaskBackend {
	return &TaskBackend{
		Logger: zaptest.NewLogger(t).With(zap.String("handler", "task")),

		AuthorizationService: mock.NewAuthorizationService(),
		TaskService:          &mock.TaskService{},
		OrganizationService: &mock.OrganizationService{
			FindOrganizationByIDF: func(ctx context.Context, id platform.ID) (*platform.Organization, error) {
				return &platform.Organization{ID: id, Name: "test"}, nil
			},
			FindOrganizationF: func(ctx context.Context, filter platform.OrganizationFilter) (*platform.Organization, error) {
				org := &platform.Organization{}
				if filter.Name != nil {
					if *filter.Name == "non-existent-org" {
						return nil, &platform.Error{
							Err:  errors.New("org not found or unauthorized"),
							Msg:  "org " + *filter.Name + " not found or unauthorized",
							Code: platform.ENotFound,
						}
					}
					org.Name = *filter.Name
				}
				if filter.ID != nil {
					org.ID = *filter.ID
				}

				return org, nil
			},
		},
		UserResourceMappingService: inmem.NewService(),
		LabelService:               mock.NewLabelService(),
		UserService:                mock.NewUserService(),
	}
}

func TestTaskHandler_handleGetTasks(t *testing.T) {
	type fields struct {
		taskService  platform.TaskService
		labelService platform.LabelService
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name      string
		getParams string
		fields    fields
		wants     wants
	}{
		{
			name: "get tasks",
			fields: fields{
				taskService: &mock.TaskService{
					FindTasksFn: func(ctx context.Context, f platform.TaskFilter) ([]*platform.Task, int, error) {
						tasks := []*platform.Task{
							{
								ID:              1,
								Name:            "task1",
								Description:     "A little Task",
								OrganizationID:  1,
								Organization:    "test",
								AuthorizationID: 0x100,
							},
							{
								ID:              2,
								Name:            "task2",
								OrganizationID:  2,
								Organization:    "test",
								AuthorizationID: 0x200,
							},
						}
						return tasks, len(tasks), nil
					},
				},
				labelService: &mock.LabelService{
					FindResourceLabelsFn: func(ctx context.Context, f platform.LabelMappingFilter) ([]*platform.Label, error) {
						labels := []*platform.Label{
							{
								ID:   platformtesting.MustIDBase16("fc3dc670a4be9b9a"),
								Name: "label",
								Properties: map[string]string{
									"color": "fff000",
								},
							},
						}
						return labels, nil
					},
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/tasks?limit=100"
  },
  "tasks": [
    {
      "links": {
        "self": "/api/v2/tasks/0000000000000001",
        "owners": "/api/v2/tasks/0000000000000001/owners",
        "members": "/api/v2/tasks/0000000000000001/members",
        "labels": "/api/v2/tasks/0000000000000001/labels",
        "runs": "/api/v2/tasks/0000000000000001/runs",
        "logs": "/api/v2/tasks/0000000000000001/logs"
      },
      "id": "0000000000000001",
      "name": "task1",
	  "description": "A little Task",
	  "labels": [
        {
          "id": "fc3dc670a4be9b9a",
          "name": "label",
          "properties": {
            "color": "fff000"
          }
        }
      ],
      "orgID": "0000000000000001",
      "org": "test",
      "status": "",
			"authorizationID": "0000000000000100",
      "flux": ""
    },
    {
      "links": {
        "self": "/api/v2/tasks/0000000000000002",
        "owners": "/api/v2/tasks/0000000000000002/owners",
        "members": "/api/v2/tasks/0000000000000002/members",
        "labels": "/api/v2/tasks/0000000000000002/labels",
        "runs": "/api/v2/tasks/0000000000000002/runs",
        "logs": "/api/v2/tasks/0000000000000002/logs"
      },
      "id": "0000000000000002",
      "name": "task2",
			"labels": [
        {
          "id": "fc3dc670a4be9b9a",
          "name": "label",
          "properties": {
            "color": "fff000"
          }
        }
      ],
	  "orgID": "0000000000000002",
	  "org": "test",
      "status": "",
			"authorizationID": "0000000000000200",
      "flux": ""
    }
  ]
}`,
			},
		},
		{
			name:      "get tasks by after and limit",
			getParams: "after=0000000000000001&limit=1",
			fields: fields{
				taskService: &mock.TaskService{
					FindTasksFn: func(ctx context.Context, f platform.TaskFilter) ([]*platform.Task, int, error) {
						tasks := []*platform.Task{
							{
								ID:              2,
								Name:            "task2",
								OrganizationID:  2,
								Organization:    "test",
								AuthorizationID: 0x200,
							},
						}
						return tasks, len(tasks), nil
					},
				},
				labelService: &mock.LabelService{
					FindResourceLabelsFn: func(ctx context.Context, f platform.LabelMappingFilter) ([]*platform.Label, error) {
						labels := []*platform.Label{
							{
								ID:   platformtesting.MustIDBase16("fc3dc670a4be9b9a"),
								Name: "label",
								Properties: map[string]string{
									"color": "fff000",
								},
							},
						}
						return labels, nil
					},
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/tasks?after=0000000000000001&limit=1",
    "next": "/api/v2/tasks?after=0000000000000002&limit=1"
  },
  "tasks": [
    {
      "links": {
        "self": "/api/v2/tasks/0000000000000002",
        "owners": "/api/v2/tasks/0000000000000002/owners",
        "members": "/api/v2/tasks/0000000000000002/members",
        "labels": "/api/v2/tasks/0000000000000002/labels",
        "runs": "/api/v2/tasks/0000000000000002/runs",
        "logs": "/api/v2/tasks/0000000000000002/logs"
      },
      "id": "0000000000000002",
      "name": "task2",
			"labels": [
        {
          "id": "fc3dc670a4be9b9a",
          "name": "label",
          "properties": {
            "color": "fff000"
          }
        }
      ],
      "orgID": "0000000000000002",
      "org": "test",
      "status": "",
			"authorizationID": "0000000000000200",
      "flux": ""
    }
  ]
}`,
			},
		},
		{
			name:      "get tasks by org name",
			getParams: "org=test2",
			fields: fields{
				taskService: &mock.TaskService{
					FindTasksFn: func(ctx context.Context, f platform.TaskFilter) ([]*platform.Task, int, error) {
						tasks := []*platform.Task{
							{
								ID:              2,
								Name:            "task2",
								OrganizationID:  2,
								Organization:    "test2",
								AuthorizationID: 0x200,
							},
						}
						return tasks, len(tasks), nil
					},
				},
				labelService: &mock.LabelService{
					FindResourceLabelsFn: func(ctx context.Context, f platform.LabelMappingFilter) ([]*platform.Label, error) {
						labels := []*platform.Label{
							{
								ID:   platformtesting.MustIDBase16("fc3dc670a4be9b9a"),
								Name: "label",
								Properties: map[string]string{
									"color": "fff000",
								},
							},
						}
						return labels, nil
					},
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/tasks?limit=100&org=test2"
  },
  "tasks": [
    {
      "links": {
        "self": "/api/v2/tasks/0000000000000002",
        "owners": "/api/v2/tasks/0000000000000002/owners",
        "members": "/api/v2/tasks/0000000000000002/members",
        "labels": "/api/v2/tasks/0000000000000002/labels",
        "runs": "/api/v2/tasks/0000000000000002/runs",
        "logs": "/api/v2/tasks/0000000000000002/logs"
      },
      "id": "0000000000000002",
      "name": "task2",
			"labels": [
        {
          "id": "fc3dc670a4be9b9a",
          "name": "label",
          "properties": {
            "color": "fff000"
          }
        }
      ],
	  "orgID": "0000000000000002",
	  "org": "test2",
      "status": "",
			"authorizationID": "0000000000000200",
      "flux": ""
    }
  ]
}`,
			},
		},
		{
			name:      "get tasks by org name bad",
			getParams: "org=non-existent-org",
			fields: fields{
				taskService: &mock.TaskService{
					FindTasksFn: func(ctx context.Context, f platform.TaskFilter) ([]*platform.Task, int, error) {
						tasks := []*platform.Task{
							{
								ID:              1,
								Name:            "task1",
								OrganizationID:  1,
								Organization:    "test2",
								AuthorizationID: 0x100,
							},
							{
								ID:              2,
								Name:            "task2",
								OrganizationID:  2,
								Organization:    "test2",
								AuthorizationID: 0x200,
							},
						}
						return tasks, len(tasks), nil
					},
				},
				labelService: &mock.LabelService{
					FindResourceLabelsFn: func(ctx context.Context, f platform.LabelMappingFilter) ([]*platform.Label, error) {
						labels := []*platform.Label{
							{
								ID:   platformtesting.MustIDBase16("fc3dc670a4be9b9a"),
								Name: "label",
								Properties: map[string]string{
									"color": "fff000",
								},
							},
						}
						return labels, nil
					},
				},
			},
			wants: wants{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json; charset=utf-8",
				body: `{
"code": "invalid",
"error": {
"code": "not found",
"error": "org not found or unauthorized",
"message": "org non-existent-org not found or unauthorized"
},
"message": "failed to decode request"
}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://any.url?"+tt.getParams, nil)
			w := httptest.NewRecorder()

			taskBackend := NewMockTaskBackend(t)
			taskBackend.HTTPErrorHandler = ErrorHandler(0)
			taskBackend.TaskService = tt.fields.taskService
			taskBackend.LabelService = tt.fields.labelService
			h := NewTaskHandler(taskBackend)
			h.handleGetTasks(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetTasks() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetTasks() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handleGetTasks(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handleGetTasks() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestTaskHandler_handlePostTasks(t *testing.T) {
	type args struct {
		taskCreate platform.TaskCreate
	}
	type fields struct {
		taskService platform.TaskService
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		args   args
		fields fields
		wants  wants
	}{
		{
			name: "create task",
			args: args{
				taskCreate: platform.TaskCreate{
					OrganizationID: 1,
					Token:          "mytoken",
					Flux:           "abc",
				},
			},
			fields: fields{
				taskService: &mock.TaskService{
					CreateTaskFn: func(ctx context.Context, tc platform.TaskCreate) (*platform.Task, error) {
						return &platform.Task{
							ID:              1,
							Name:            "task1",
							Description:     "Brand New Task",
							OrganizationID:  1,
							Organization:    "test",
							AuthorizationID: 0x100,
							Flux:            "abc",
						}, nil
					},
				},
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/tasks/0000000000000001",
    "owners": "/api/v2/tasks/0000000000000001/owners",
    "members": "/api/v2/tasks/0000000000000001/members",
    "labels": "/api/v2/tasks/0000000000000001/labels",
    "runs": "/api/v2/tasks/0000000000000001/runs",
    "logs": "/api/v2/tasks/0000000000000001/logs"
  },
  "id": "0000000000000001",
  "name": "task1",
  "description": "Brand New Task",
  "labels": [],
  "orgID": "0000000000000001",
  "org": "test",
  "status": "",
	"authorizationID": "0000000000000100",
  "flux": "abc"
}
`,
			},
		},
		{
			name: "create task - platform error creating task",
			args: args{
				taskCreate: platform.TaskCreate{
					OrganizationID: 1,
					Token:          "mytoken",
					Flux:           "abc",
				},
			},
			fields: fields{
				taskService: &mock.TaskService{
					CreateTaskFn: func(ctx context.Context, tc platform.TaskCreate) (*platform.Task, error) {
						return nil, platform.NewError(
							platform.WithErrorErr(errors.New("something went wrong")),
							platform.WithErrorMsg("something really went wrong"),
							platform.WithErrorCode(platform.EInvalid),
						)
					},
				},
			},
			wants: wants{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json; charset=utf-8",
				body: `
{
    "code": "invalid",
    "message": "something really went wrong",
    "error": "something went wrong"
}
`,
			},
		},
		{
			name: "create task - error creating task",
			args: args{
				taskCreate: platform.TaskCreate{
					OrganizationID: 1,
					Token:          "mytoken",
					Flux:           "abc",
				},
			},
			fields: fields{
				taskService: &mock.TaskService{
					CreateTaskFn: func(ctx context.Context, tc platform.TaskCreate) (*platform.Task, error) {
						return nil, errors.New("something bad happened")
					},
				},
			},
			wants: wants{
				statusCode:  http.StatusInternalServerError,
				contentType: "application/json; charset=utf-8",
				body: `
{
    "code": "internal error",
    "message": "failed to create task",
    "error": "something bad happened"
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.args.taskCreate)
			if err != nil {
				t.Fatalf("failed to unmarshal task: %v", err)
			}

			r := httptest.NewRequest("POST", "http://any.url", bytes.NewReader(b))
			ctx := pcontext.SetAuthorizer(context.TODO(), new(platform.Authorization))
			r = r.WithContext(ctx)

			w := httptest.NewRecorder()

			taskBackend := NewMockTaskBackend(t)
			taskBackend.HTTPErrorHandler = ErrorHandler(0)
			taskBackend.TaskService = tt.fields.taskService
			h := NewTaskHandler(taskBackend)
			h.handlePostTask(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostTask() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostTask() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handlePostTask(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handlePostTask() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestTaskHandler_handleGetRun(t *testing.T) {
	type fields struct {
		taskService platform.TaskService
	}
	type args struct {
		taskID platform.ID
		runID  platform.ID
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "get a run by id",
			fields: fields{
				taskService: &mock.TaskService{
					FindRunByIDFn: func(ctx context.Context, taskID platform.ID, runID platform.ID) (*platform.Run, error) {
						run := platform.Run{
							ID:           runID,
							TaskID:       taskID,
							Status:       "success",
							ScheduledFor: "2018-12-01T17:00:13Z",
							StartedAt:    "2018-12-01T17:00:03.155645Z",
							FinishedAt:   "2018-12-01T17:00:13.155645Z",
							RequestedAt:  "2018-12-01T17:00:13Z",
						}
						return &run, nil
					},
				},
			},
			args: args{
				taskID: 1,
				runID:  2,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/tasks/0000000000000001/runs/0000000000000002",
    "task": "/api/v2/tasks/0000000000000001",
    "retry": "/api/v2/tasks/0000000000000001/runs/0000000000000002/retry",
    "logs": "/api/v2/tasks/0000000000000001/runs/0000000000000002/logs"
  },
  "id": "0000000000000002",
  "taskID": "0000000000000001",
  "status": "success",
  "scheduledFor": "2018-12-01T17:00:13Z",
  "startedAt": "2018-12-01T17:00:03.155645Z",
  "finishedAt": "2018-12-01T17:00:13.155645Z",
  "requestedAt": "2018-12-01T17:00:13Z"
}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://any.url", nil)
			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.taskID.String(),
					},
					{
						Key:   "rid",
						Value: tt.args.runID.String(),
					},
				}))
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), &platform.Authorization{Permissions: platform.OperPermissions()}))
			w := httptest.NewRecorder()
			taskBackend := NewMockTaskBackend(t)
			taskBackend.HTTPErrorHandler = ErrorHandler(0)
			taskBackend.TaskService = tt.fields.taskService
			h := NewTaskHandler(taskBackend)
			h.handleGetRun(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetRun() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetRun() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handleGetRun(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handleGetRun() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestTaskHandler_handleGetRuns(t *testing.T) {
	type fields struct {
		taskService platform.TaskService
	}
	type args struct {
		taskID platform.ID
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "get runs by task id",
			fields: fields{
				taskService: &mock.TaskService{
					FindRunsFn: func(ctx context.Context, f platform.RunFilter) ([]*platform.Run, int, error) {
						runs := []*platform.Run{
							{
								ID:           platform.ID(2),
								TaskID:       f.Task,
								Status:       "success",
								ScheduledFor: "2018-12-01T17:00:13Z",
								StartedAt:    "2018-12-01T17:00:03.155645Z",
								FinishedAt:   "2018-12-01T17:00:13.155645Z",
								RequestedAt:  "2018-12-01T17:00:13Z",
							},
						}
						return runs, len(runs), nil
					},
				},
			},
			args: args{
				taskID: 1,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/tasks/0000000000000001/runs",
    "task": "/api/v2/tasks/0000000000000001"
  },
  "runs": [
    {
      "links": {
        "self": "/api/v2/tasks/0000000000000001/runs/0000000000000002",
        "task": "/api/v2/tasks/0000000000000001",
        "retry": "/api/v2/tasks/0000000000000001/runs/0000000000000002/retry",
        "logs": "/api/v2/tasks/0000000000000001/runs/0000000000000002/logs"
      },
      "id": "0000000000000002",
      "taskID": "0000000000000001",
      "status": "success",
      "scheduledFor": "2018-12-01T17:00:13Z",
      "startedAt": "2018-12-01T17:00:03.155645Z",
      "finishedAt": "2018-12-01T17:00:13.155645Z",
      "requestedAt": "2018-12-01T17:00:13Z"
    }
  ]
}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://any.url", nil)
			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.taskID.String(),
					},
				}))
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), &platform.Authorization{Permissions: platform.OperPermissions()}))
			w := httptest.NewRecorder()
			taskBackend := NewMockTaskBackend(t)
			taskBackend.HTTPErrorHandler = ErrorHandler(0)
			taskBackend.TaskService = tt.fields.taskService
			h := NewTaskHandler(taskBackend)
			h.handleGetRuns(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetRuns() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetRuns() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handleGetRuns(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handleGetRuns() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestTaskHandler_NotFoundStatus(t *testing.T) {
	// Ensure that the HTTP handlers return 404s for missing resources, and OKs for matching.

	im := inmem.NewService()
	taskBackend := NewMockTaskBackend(t)
	taskBackend.HTTPErrorHandler = ErrorHandler(0)
	h := NewTaskHandler(taskBackend)
	h.UserResourceMappingService = im
	h.LabelService = im
	h.UserService = im
	h.OrganizationService = im

	o := platform.Organization{Name: "o"}
	ctx := context.Background()
	if err := h.OrganizationService.CreateOrganization(ctx, &o); err != nil {
		t.Fatal(err)
	}

	// Create a session to associate with the contexts, so authorization checks pass.
	authz := &platform.Authorization{Permissions: platform.OperPermissions()}

	const taskID, runID = platform.ID(0xCCCCCC), platform.ID(0xAAAAAA)

	var (
		okTask    = []interface{}{taskID}
		okTaskRun = []interface{}{taskID, runID}

		notFoundTask = [][]interface{}{
			{taskID + 1},
		}
		notFoundTaskRun = [][]interface{}{
			{taskID, runID + 1},
			{taskID + 1, runID},
			{taskID + 1, runID + 1},
		}
	)

	tcs := []struct {
		name             string
		svc              *mock.TaskService
		method           string
		body             string
		pathFmt          string
		okPathArgs       []interface{}
		notFoundPathArgs [][]interface{}
	}{
		{
			name: "get task",
			svc: &mock.TaskService{
				FindTaskByIDFn: func(_ context.Context, id platform.ID) (*platform.Task, error) {
					if id == taskID {
						return &platform.Task{ID: taskID, Organization: "o"}, nil
					}

					return nil, platform.ErrTaskNotFound
				},
			},
			method:           http.MethodGet,
			pathFmt:          "/tasks/%s",
			okPathArgs:       okTask,
			notFoundPathArgs: notFoundTask,
		},
		{
			name: "update task",
			svc: &mock.TaskService{
				UpdateTaskFn: func(_ context.Context, id platform.ID, _ platform.TaskUpdate) (*platform.Task, error) {
					if id == taskID {
						return &platform.Task{ID: taskID, Organization: "o"}, nil
					}

					return nil, platform.ErrTaskNotFound
				},
			},
			method:           http.MethodPatch,
			body:             `{"status": "active"}`,
			pathFmt:          "/tasks/%s",
			okPathArgs:       okTask,
			notFoundPathArgs: notFoundTask,
		},
		{
			name: "delete task",
			svc: &mock.TaskService{
				DeleteTaskFn: func(_ context.Context, id platform.ID) error {
					if id == taskID {
						return nil
					}

					return platform.ErrTaskNotFound
				},
			},
			method:           http.MethodDelete,
			pathFmt:          "/tasks/%s",
			okPathArgs:       okTask,
			notFoundPathArgs: notFoundTask,
		},
		{
			name: "get task logs",
			svc: &mock.TaskService{
				FindLogsFn: func(_ context.Context, f platform.LogFilter) ([]*platform.Log, int, error) {
					if f.Task == taskID {
						return nil, 0, nil
					}

					return nil, 0, platform.ErrTaskNotFound
				},
			},
			method:           http.MethodGet,
			pathFmt:          "/tasks/%s/logs",
			okPathArgs:       okTask,
			notFoundPathArgs: notFoundTask,
		},
		{
			name: "get run logs",
			svc: &mock.TaskService{
				FindLogsFn: func(_ context.Context, f platform.LogFilter) ([]*platform.Log, int, error) {
					if f.Task != taskID {
						return nil, 0, platform.ErrTaskNotFound
					}
					if *f.Run != runID {
						return nil, 0, platform.ErrNoRunsFound
					}

					return nil, 0, nil
				},
			},
			method:           http.MethodGet,
			pathFmt:          "/tasks/%s/runs/%s/logs",
			okPathArgs:       okTaskRun,
			notFoundPathArgs: notFoundTaskRun,
		},
		{
			name: "get runs: task not found",
			svc: &mock.TaskService{
				FindRunsFn: func(_ context.Context, f platform.RunFilter) ([]*platform.Run, int, error) {
					if f.Task != taskID {
						return nil, 0, platform.ErrTaskNotFound
					}

					return nil, 0, nil
				},
			},
			method:           http.MethodGet,
			pathFmt:          "/tasks/%s/runs",
			okPathArgs:       okTask,
			notFoundPathArgs: notFoundTask,
		},
		{
			name: "get runs: task found but no runs found",
			svc: &mock.TaskService{
				FindRunsFn: func(_ context.Context, f platform.RunFilter) ([]*platform.Run, int, error) {
					if f.Task != taskID {
						return nil, 0, platform.ErrNoRunsFound
					}

					return nil, 0, nil
				},
			},
			method:           http.MethodGet,
			pathFmt:          "/tasks/%s/runs",
			okPathArgs:       okTask,
			notFoundPathArgs: notFoundTask,
		},
		{
			name: "force run",
			svc: &mock.TaskService{
				ForceRunFn: func(_ context.Context, tid platform.ID, _ int64) (*platform.Run, error) {
					if tid != taskID {
						return nil, platform.ErrTaskNotFound
					}

					return &platform.Run{ID: runID, TaskID: taskID, Status: backend.RunScheduled.String()}, nil
				},
			},
			method:           http.MethodPost,
			body:             "{}",
			pathFmt:          "/tasks/%s/runs",
			okPathArgs:       okTask,
			notFoundPathArgs: notFoundTask,
		},
		{
			name: "get run",
			svc: &mock.TaskService{
				FindRunByIDFn: func(_ context.Context, tid, rid platform.ID) (*platform.Run, error) {
					if tid != taskID {
						return nil, platform.ErrTaskNotFound
					}
					if rid != runID {
						return nil, platform.ErrRunNotFound
					}

					return &platform.Run{ID: runID, TaskID: taskID, Status: backend.RunScheduled.String()}, nil
				},
			},
			method:           http.MethodGet,
			pathFmt:          "/tasks/%s/runs/%s",
			okPathArgs:       okTaskRun,
			notFoundPathArgs: notFoundTaskRun,
		},
		{
			name: "retry run",
			svc: &mock.TaskService{
				RetryRunFn: func(_ context.Context, tid, rid platform.ID) (*platform.Run, error) {
					if tid != taskID {
						return nil, platform.ErrTaskNotFound
					}
					if rid != runID {
						return nil, platform.ErrRunNotFound
					}

					return &platform.Run{ID: runID, TaskID: taskID, Status: backend.RunScheduled.String()}, nil
				},
			},
			method:           http.MethodPost,
			pathFmt:          "/tasks/%s/runs/%s/retry",
			okPathArgs:       okTaskRun,
			notFoundPathArgs: notFoundTaskRun,
		},
		{
			name: "cancel run",
			svc: &mock.TaskService{
				CancelRunFn: func(_ context.Context, tid, rid platform.ID) error {
					if tid != taskID {
						return platform.ErrTaskNotFound
					}
					if rid != runID {
						return platform.ErrRunNotFound
					}

					return nil
				},
			},
			method:           http.MethodDelete,
			pathFmt:          "/tasks/%s/runs/%s",
			okPathArgs:       okTaskRun,
			notFoundPathArgs: notFoundTaskRun,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			h.TaskService = tc.svc

			okPath := fmt.Sprintf(tc.pathFmt, tc.okPathArgs...)
			t.Run("matching ID: "+tc.method+" "+okPath, func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(tc.method, "http://task.example/api/v2"+okPath, strings.NewReader(tc.body)).WithContext(
					pcontext.SetAuthorizer(context.Background(), authz),
				)

				h.ServeHTTP(w, r)

				res := w.Result()
				defer res.Body.Close()

				if res.StatusCode < 200 || res.StatusCode > 299 {
					t.Errorf("expected OK, got %d", res.StatusCode)
					b, _ := ioutil.ReadAll(res.Body)
					t.Fatalf("body: %s", string(b))
				}
			})

			t.Run("mismatched ID", func(t *testing.T) {
				for _, nfa := range tc.notFoundPathArgs {
					path := fmt.Sprintf(tc.pathFmt, nfa...)
					t.Run(tc.method+" "+path, func(t *testing.T) {
						w := httptest.NewRecorder()
						r := httptest.NewRequest(tc.method, "http://task.example/api/v2"+path, strings.NewReader(tc.body)).WithContext(
							pcontext.SetAuthorizer(context.Background(), authz),
						)

						h.ServeHTTP(w, r)

						res := w.Result()
						defer res.Body.Close()

						if res.StatusCode != http.StatusNotFound {
							t.Errorf("expected Not Found, got %d", res.StatusCode)
							b, _ := ioutil.ReadAll(res.Body)
							t.Fatalf("body: %s", string(b))
						}
					})
				}
			})
		})
	}
}

func TestService_handlePostTaskLabel(t *testing.T) {
	type fields struct {
		LabelService platform.LabelService
	}
	type args struct {
		labelMapping *platform.LabelMapping
		taskID       platform.ID
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "add label to task",
			fields: fields{
				LabelService: &mock.LabelService{
					FindLabelByIDFn: func(ctx context.Context, id platform.ID) (*platform.Label, error) {
						return &platform.Label{
							ID:   1,
							Name: "label",
							Properties: map[string]string{
								"color": "fff000",
							},
						}, nil
					},
					CreateLabelMappingFn: func(ctx context.Context, m *platform.LabelMapping) error { return nil },
				},
			},
			args: args{
				labelMapping: &platform.LabelMapping{
					ResourceID: 100,
					LabelID:    1,
				},
				taskID: 100,
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "label": {
    "id": "0000000000000001",
    "name": "label",
    "properties": {
      "color": "fff000"
    }
  },
  "links": {
    "self": "/api/v2/labels/0000000000000001"
  }
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskBE := NewMockTaskBackend(t)
			taskBE.LabelService = tt.fields.LabelService
			h := NewTaskHandler(taskBE)

			b, err := json.Marshal(tt.args.labelMapping)
			if err != nil {
				t.Fatalf("failed to unmarshal label mapping: %v", err)
			}

			url := fmt.Sprintf("http://localhost:9999/api/v2/tasks/%s/labels", tt.args.taskID)
			r := httptest.NewRequest("POST", url, bytes.NewReader(b))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("got %v, want %v", res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("got %v, want %v", content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handlePostTaskLabel(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handlePostTaskLabel() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

// Test that org name to org ID translation happens properly in the HTTP layer.
// Regression test for https://github.com/influxdata/influxdb/issues/12089.
func TestTaskHandler_CreateTaskWithOrgName(t *testing.T) {
	i := inmem.NewService()
	ctx := context.Background()

	// Set up user and org.
	u := &platform.User{Name: "u"}
	if err := i.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	o := &platform.Organization{Name: "o"}
	if err := i.CreateOrganization(ctx, o); err != nil {
		t.Fatal(err)
	}

	// Source and destination buckets for use in task.
	bSrc := platform.Bucket{OrgID: o.ID, Name: "b-src"}
	if err := i.CreateBucket(ctx, &bSrc); err != nil {
		t.Fatal(err)
	}
	bDst := platform.Bucket{OrgID: o.ID, Name: "b-dst"}
	if err := i.CreateBucket(ctx, &bDst); err != nil {
		t.Fatal(err)
	}

	authz := platform.Authorization{OrgID: o.ID, UserID: u.ID, Permissions: platform.OperPermissions()}
	if err := i.CreateAuthorization(ctx, &authz); err != nil {
		t.Fatal(err)
	}

	ts := &mock.TaskService{
		CreateTaskFn: func(_ context.Context, tc platform.TaskCreate) (*platform.Task, error) {
			if tc.OrganizationID != o.ID {
				t.Fatalf("expected task to be created with org ID %s, got %s", o.ID, tc.OrganizationID)
			}
			if tc.Token != authz.Token {
				t.Fatalf("expected task to be created with previous token %s, got %s", authz.Token, tc.Token)
			}

			return &platform.Task{ID: 9, OrganizationID: o.ID, AuthorizationID: authz.ID, Name: "x", Flux: tc.Flux}, nil
		},
	}

	h := NewTaskHandler(&TaskBackend{
		Logger: zaptest.NewLogger(t),

		TaskService:                ts,
		AuthorizationService:       i,
		OrganizationService:        i,
		UserResourceMappingService: i,
		LabelService:               i,
		UserService:                i,
		BucketService:              i,
	})

	const script = `option task = {name:"x", every:1m} from(bucket:"b-src") |> range(start:-1m) |> to(bucket:"b-dst", org:"o")`

	url := "http://localhost:9999/api/v2/tasks"

	b, err := json.Marshal(platform.TaskCreate{
		Flux:         script,
		Organization: o.Name,
		Token:        authz.Token,
	})
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", url, bytes.NewReader(b)).WithContext(
		pcontext.SetAuthorizer(ctx, &authz),
	)
	w := httptest.NewRecorder()

	h.handlePostTask(w, r)

	res := w.Result()
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Logf("response body: %s", body)
		t.Fatalf("expected status created, got %v", res.StatusCode)
	}

	// The task should have been created with a valid token.
	var createdTask platform.Task
	if err := json.Unmarshal([]byte(body), &createdTask); err != nil {
		t.Fatal(err)
	}
	if createdTask.Flux != script {
		t.Fatalf("Unexpected script returned:\n got: %s\nwant: %s", createdTask.Flux, script)
	}
}

func TestTaskHandler_Sessions(t *testing.T) {
	t.Skip("rework these")
	// Common setup to get a working base for using tasks.
	i := inmem.NewService()

	ctx := context.Background()

	// Set up user and org.
	u := &platform.User{Name: "u"}
	if err := i.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	o := &platform.Organization{Name: "o"}
	if err := i.CreateOrganization(ctx, o); err != nil {
		t.Fatal(err)
	}

	// Map user to org.
	if err := i.CreateUserResourceMapping(ctx, &platform.UserResourceMapping{
		ResourceType: platform.OrgsResourceType,
		ResourceID:   o.ID,
		UserID:       u.ID,
		UserType:     platform.Owner,
	}); err != nil {
		t.Fatal(err)
	}

	// Source and destination buckets for use in task.
	bSrc := platform.Bucket{OrgID: o.ID, Name: "b-src"}
	if err := i.CreateBucket(ctx, &bSrc); err != nil {
		t.Fatal(err)
	}
	bDst := platform.Bucket{OrgID: o.ID, Name: "b-dst"}
	if err := i.CreateBucket(ctx, &bDst); err != nil {
		t.Fatal(err)
	}

	sessionAllPermsCtx := pcontext.SetAuthorizer(context.Background(), &platform.Session{
		UserID:      u.ID,
		Permissions: platform.OperPermissions(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	})

	newHandler := func(t *testing.T, ts *mock.TaskService) *TaskHandler {
		return NewTaskHandler(&TaskBackend{
			HTTPErrorHandler: ErrorHandler(0),
			Logger:           zaptest.NewLogger(t),

			TaskService:                ts,
			AuthorizationService:       i,
			OrganizationService:        i,
			UserResourceMappingService: i,
			LabelService:               i,
			UserService:                i,
			BucketService:              i,
		})
	}

	t.Run("get runs for a task", func(t *testing.T) {
		// Unique authorization to associate with our fake task.
		taskAuth := &platform.Authorization{OrgID: o.ID, UserID: u.ID}
		if err := i.CreateAuthorization(ctx, taskAuth); err != nil {
			t.Fatal(err)
		}

		const taskID = platform.ID(12345)
		const runID = platform.ID(9876)

		var findRunsCtx context.Context
		ts := &mock.TaskService{
			FindRunsFn: func(ctx context.Context, f platform.RunFilter) ([]*platform.Run, int, error) {
				findRunsCtx = ctx
				if f.Task != taskID {
					t.Fatalf("expected task ID %v, got %v", taskID, f.Task)
				}

				return []*platform.Run{
					{ID: runID, TaskID: taskID},
				}, 1, nil
			},

			FindTaskByIDFn: func(ctx context.Context, id platform.ID) (*platform.Task, error) {
				if id != taskID {
					return nil, platform.ErrTaskNotFound
				}

				return &platform.Task{
					ID:              taskID,
					OrganizationID:  o.ID,
					AuthorizationID: taskAuth.ID,
				}, nil
			},
		}

		h := newHandler(t, ts)
		url := fmt.Sprintf("http://localhost:9999/api/v2/tasks/%s/runs", taskID)
		valCtx := context.WithValue(sessionAllPermsCtx, httprouter.ParamsKey, httprouter.Params{{Key: "id", Value: taskID.String()}})
		r := httptest.NewRequest("GET", url, nil).WithContext(valCtx)
		w := httptest.NewRecorder()
		h.handleGetRuns(w, r)

		res := w.Result()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Logf("response body: %s", body)
			t.Fatalf("expected status OK, got %v", res.StatusCode)
		}

		authr, err := pcontext.GetAuthorizer(findRunsCtx)
		if err != nil {
			t.Fatal(err)
		}
		if authr.Kind() != platform.AuthorizationKind {
			t.Fatalf("expected context's authorizer to be of kind %q, got %q", platform.AuthorizationKind, authr.Kind())
		}

		orgID := authr.(*platform.Authorization).OrgID

		if orgID != o.ID {
			t.Fatalf("expected context's authorizer org ID to be %v, got %v", o.ID, orgID)
		}

		// Other user without permissions on the task or authorization should be disallowed.
		otherUser := &platform.User{Name: "other-" + t.Name()}
		if err := i.CreateUser(ctx, otherUser); err != nil {
			t.Fatal(err)
		}

		valCtx = pcontext.SetAuthorizer(valCtx, &platform.Session{
			UserID:    otherUser.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})

		r = httptest.NewRequest("GET", url, nil).WithContext(valCtx)
		w = httptest.NewRecorder()
		h.handleGetRuns(w, r)

		res = w.Result()
		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusUnauthorized {
			t.Logf("response body: %s", body)
			t.Fatalf("expected status unauthorized, got %v", res.StatusCode)
		}
	})

	t.Run("get single run for a task", func(t *testing.T) {
		// Unique authorization to associate with our fake task.
		taskAuth := &platform.Authorization{OrgID: o.ID, UserID: u.ID}
		if err := i.CreateAuthorization(ctx, taskAuth); err != nil {
			t.Fatal(err)
		}

		const taskID = platform.ID(12345)
		const runID = platform.ID(9876)

		var findRunByIDCtx context.Context
		ts := &mock.TaskService{
			FindRunByIDFn: func(ctx context.Context, tid, rid platform.ID) (*platform.Run, error) {
				findRunByIDCtx = ctx
				if tid != taskID {
					t.Fatalf("expected task ID %v, got %v", taskID, tid)
				}
				if rid != runID {
					t.Fatalf("expected run ID %v, got %v", runID, rid)
				}

				return &platform.Run{ID: runID, TaskID: taskID}, nil
			},

			FindTaskByIDFn: func(ctx context.Context, id platform.ID) (*platform.Task, error) {
				if id != taskID {
					return nil, platform.ErrTaskNotFound
				}

				return &platform.Task{
					ID:              taskID,
					OrganizationID:  o.ID,
					AuthorizationID: taskAuth.ID,
				}, nil
			},
		}

		h := newHandler(t, ts)
		url := fmt.Sprintf("http://localhost:9999/api/v2/tasks/%s/runs/%s", taskID, runID)
		valCtx := context.WithValue(sessionAllPermsCtx, httprouter.ParamsKey, httprouter.Params{
			{Key: "id", Value: taskID.String()},
			{Key: "rid", Value: runID.String()},
		})
		r := httptest.NewRequest("GET", url, nil).WithContext(valCtx)
		w := httptest.NewRecorder()
		h.handleGetRun(w, r)

		res := w.Result()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Logf("response body: %s", body)
			t.Fatalf("expected status OK, got %v", res.StatusCode)
		}

		// The context passed to TaskService.FindRunByID must be a valid authorization (not a session).
		authr, err := pcontext.GetAuthorizer(findRunByIDCtx)
		if err != nil {
			t.Fatal(err)
		}
		if authr.Kind() != platform.AuthorizationKind {
			t.Fatalf("expected context's authorizer to be of kind %q, got %q", platform.AuthorizationKind, authr.Kind())
		}
		if authr.Identifier() != taskAuth.ID {
			t.Fatalf("expected context's authorizer ID to be %v, got %v", taskAuth.ID, authr.Identifier())
		}

		// Other user without permissions on the task or authorization should be disallowed.
		otherUser := &platform.User{Name: "other-" + t.Name()}
		if err := i.CreateUser(ctx, otherUser); err != nil {
			t.Fatal(err)
		}

		valCtx = pcontext.SetAuthorizer(valCtx, &platform.Session{
			UserID:    otherUser.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})

		r = httptest.NewRequest("GET", url, nil).WithContext(valCtx)
		w = httptest.NewRecorder()
		h.handleGetRuns(w, r)

		res = w.Result()
		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusUnauthorized {
			t.Logf("response body: %s", body)
			t.Fatalf("expected status unauthorized, got %v", res.StatusCode)
		}
	})

	t.Run("get logs for a run", func(t *testing.T) {
		// Unique authorization to associate with our fake task.
		taskAuth := &platform.Authorization{OrgID: o.ID, UserID: u.ID}
		if err := i.CreateAuthorization(ctx, taskAuth); err != nil {
			t.Fatal(err)
		}

		const taskID = platform.ID(12345)
		const runID = platform.ID(9876)

		var findLogsCtx context.Context
		ts := &mock.TaskService{
			FindLogsFn: func(ctx context.Context, f platform.LogFilter) ([]*platform.Log, int, error) {
				findLogsCtx = ctx
				if f.Task != taskID {
					t.Fatalf("expected task ID %v, got %v", taskID, f.Task)
				}
				if *f.Run != runID {
					t.Fatalf("expected run ID %v, got %v", runID, *f.Run)
				}

				line := platform.Log{Time: "time", Message: "a log line"}
				return []*platform.Log{&line}, 1, nil
			},

			FindTaskByIDFn: func(ctx context.Context, id platform.ID) (*platform.Task, error) {
				if id != taskID {
					return nil, platform.ErrTaskNotFound
				}

				return &platform.Task{
					ID:              taskID,
					OrganizationID:  o.ID,
					AuthorizationID: taskAuth.ID,
				}, nil
			},
		}

		h := newHandler(t, ts)
		url := fmt.Sprintf("http://localhost:9999/api/v2/tasks/%s/runs/%s/logs", taskID, runID)
		valCtx := context.WithValue(sessionAllPermsCtx, httprouter.ParamsKey, httprouter.Params{
			{Key: "id", Value: taskID.String()},
			{Key: "rid", Value: runID.String()},
		})
		r := httptest.NewRequest("GET", url, nil).WithContext(valCtx)
		w := httptest.NewRecorder()
		h.handleGetLogs(w, r)

		res := w.Result()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Logf("response body: %s", body)
			t.Fatalf("expected status OK, got %v", res.StatusCode)
		}

		// The context passed to TaskService.FindLogs must be a valid authorization (not a session).
		authr, err := pcontext.GetAuthorizer(findLogsCtx)
		if err != nil {
			t.Fatal(err)
		}
		if authr.Kind() != platform.AuthorizationKind {
			t.Fatalf("expected context's authorizer to be of kind %q, got %q", platform.AuthorizationKind, authr.Kind())
		}
		if authr.Identifier() != taskAuth.ID {
			t.Fatalf("expected context's authorizer ID to be %v, got %v", taskAuth.ID, authr.Identifier())
		}

		// Other user without permissions on the task or authorization should be disallowed.
		otherUser := &platform.User{Name: "other-" + t.Name()}
		if err := i.CreateUser(ctx, otherUser); err != nil {
			t.Fatal(err)
		}

		valCtx = pcontext.SetAuthorizer(valCtx, &platform.Session{
			UserID:    otherUser.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})

		r = httptest.NewRequest("GET", url, nil).WithContext(valCtx)
		w = httptest.NewRecorder()
		h.handleGetRuns(w, r)

		res = w.Result()
		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusUnauthorized {
			t.Logf("response body: %s", body)
			t.Fatalf("expected status unauthorized, got %v", res.StatusCode)
		}
	})

	t.Run("retry a run", func(t *testing.T) {
		// Unique authorization to associate with our fake task.
		taskAuth := &platform.Authorization{OrgID: o.ID, UserID: u.ID}
		if err := i.CreateAuthorization(ctx, taskAuth); err != nil {
			t.Fatal(err)
		}

		const taskID = platform.ID(12345)
		const runID = platform.ID(9876)

		var retryRunCtx context.Context
		ts := &mock.TaskService{
			RetryRunFn: func(ctx context.Context, tid, rid platform.ID) (*platform.Run, error) {
				retryRunCtx = ctx
				if tid != taskID {
					t.Fatalf("expected task ID %v, got %v", taskID, tid)
				}
				if rid != runID {
					t.Fatalf("expected run ID %v, got %v", runID, rid)
				}

				return &platform.Run{ID: 10 * runID, TaskID: taskID}, nil
			},

			FindTaskByIDFn: func(ctx context.Context, id platform.ID) (*platform.Task, error) {
				if id != taskID {
					return nil, platform.ErrTaskNotFound
				}

				return &platform.Task{
					ID:              taskID,
					OrganizationID:  o.ID,
					AuthorizationID: taskAuth.ID,
				}, nil
			},
		}

		h := newHandler(t, ts)
		url := fmt.Sprintf("http://localhost:9999/api/v2/tasks/%s/runs/%s/retry", taskID, runID)
		valCtx := context.WithValue(sessionAllPermsCtx, httprouter.ParamsKey, httprouter.Params{
			{Key: "id", Value: taskID.String()},
			{Key: "rid", Value: runID.String()},
		})
		r := httptest.NewRequest("POST", url, nil).WithContext(valCtx)
		w := httptest.NewRecorder()
		h.handleRetryRun(w, r)

		res := w.Result()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Logf("response body: %s", body)
			t.Fatalf("expected status OK, got %v", res.StatusCode)
		}

		// The context passed to TaskService.RetryRun must be a valid authorization (not a session).
		authr, err := pcontext.GetAuthorizer(retryRunCtx)
		if err != nil {
			t.Fatal(err)
		}
		if authr.Kind() != platform.AuthorizationKind {
			t.Fatalf("expected context's authorizer to be of kind %q, got %q", platform.AuthorizationKind, authr.Kind())
		}
		if authr.Identifier() != taskAuth.ID {
			t.Fatalf("expected context's authorizer ID to be %v, got %v", taskAuth.ID, authr.Identifier())
		}

		// Other user without permissions on the task or authorization should be disallowed.
		otherUser := &platform.User{Name: "other-" + t.Name()}
		if err := i.CreateUser(ctx, otherUser); err != nil {
			t.Fatal(err)
		}

		valCtx = pcontext.SetAuthorizer(valCtx, &platform.Session{
			UserID:    otherUser.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})

		r = httptest.NewRequest("POST", url, nil).WithContext(valCtx)
		w = httptest.NewRecorder()
		h.handleGetRuns(w, r)

		res = w.Result()
		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusUnauthorized {
			t.Logf("response body: %s", body)
			t.Fatalf("expected status unauthorized, got %v", res.StatusCode)
		}
	})
}
