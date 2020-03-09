package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/influxdata/influxdb"
	pcontext "github.com/influxdata/influxdb/context"
	"github.com/influxdata/influxdb/mock"
	"github.com/influxdata/influxdb/notification/endpoint"
	influxTesting "github.com/influxdata/influxdb/testing"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// NewMockNotificationEndpointBackend returns a NotificationEndpointBackend with mock services.
func NewMockNotificationEndpointBackend() *NotificationEndpointBackend {
	return &NotificationEndpointBackend{
		Logger: zap.NewNop().With(zap.String("handler", "notification endpoint")),

		NotificationEndpointService: &mock.NotificationEndpointService{},
		UserResourceMappingService:  mock.NewUserResourceMappingService(),
		LabelService:                mock.NewLabelService(),
		UserService:                 mock.NewUserService(),
		OrganizationService:         mock.NewOrganizationService(),
		SecretService:               mock.NewSecretService(),
	}
}

func TestService_handleGetNotificationEndpoints(t *testing.T) {
	type fields struct {
		NotificationEndpointService influxdb.NotificationEndpointService
		LabelService                influxdb.LabelService
	}
	type args struct {
		queryParams map[string][]string
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
			name: "get all notification endpoints",
			fields: fields{
				&mock.NotificationEndpointService{
					FindNotificationEndpointsF: func(ctx context.Context, filter influxdb.NotificationEndpointFilter, opts ...influxdb.FindOptions) ([]influxdb.NotificationEndpoint, int, error) {
						return []influxdb.NotificationEndpoint{
							&endpoint.Slack{
								Base: endpoint.Base{
									ID:     influxTesting.MustIDBase16("0b501e7e557ab1ed"),
									Name:   "hello",
									OrgID:  influxTesting.MustIDBase16("50f7ba1150f7ba11"),
									Status: influxdb.Active,
								},
								URL: "http://example.com",
							},
							&endpoint.HTTP{
								Base: endpoint.Base{
									ID:     influxTesting.MustIDBase16("c0175f0077a77005"),
									Name:   "example",
									OrgID:  influxTesting.MustIDBase16("7e55e118dbabb1ed"),
									Status: influxdb.Inactive,
								},
								URL:             "example.com",
								Username:        influxdb.SecretField{Key: "http-user-key"},
								Password:        influxdb.SecretField{Key: "http-password-key"},
								AuthMethod:      "basic",
								Method:          "POST",
								ContentTemplate: "template",
								Headers: map[string]string{
									"x-header-1": "header 1",
									"x-header-2": "header 2",
								},
							},
						}, 2, nil
					},
				},
				&mock.LabelService{
					FindResourceLabelsFn: func(ctx context.Context, f influxdb.LabelMappingFilter) ([]*influxdb.Label, error) {
						labels := []*influxdb.Label{
							{
								ID:   influxTesting.MustIDBase16("fc3dc670a4be9b9a"),
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
			args: args{
				map[string][]string{
					"limit": {"1"},
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
		{
		  "links": {
		    "self": "/api/v2/notificationEndpoints?descending=false&limit=1&offset=0",
		    "next": "/api/v2/notificationEndpoints?descending=false&limit=1&offset=1"
		  },
		  "notificationEndpoints": [
		   {
		     "createdAt": "0001-01-01T00:00:00Z",
		     "id": "0b501e7e557ab1ed",
		     "labels": [
		       {
		         "id": "fc3dc670a4be9b9a",
		         "name": "label",
		         "properties": {
		           "color": "fff000"
		         }
		       }
		     ],
		     "links": {
		       "labels": "/api/v2/notificationEndpoints/0b501e7e557ab1ed/labels",
		       "members": "/api/v2/notificationEndpoints/0b501e7e557ab1ed/members",
		       "owners": "/api/v2/notificationEndpoints/0b501e7e557ab1ed/owners",
		       "self": "/api/v2/notificationEndpoints/0b501e7e557ab1ed"
		     },
		     "name": "hello",
		     "orgID": "50f7ba1150f7ba11",
		     "status": "active",
			 "type": "slack",
			 "token": "",
		     "updatedAt": "0001-01-01T00:00:00Z",
		     "url": "http://example.com"
		   },
		   {
		     "createdAt": "0001-01-01T00:00:00Z",
		     "url": "example.com",
		     "id": "c0175f0077a77005",
		     "labels": [
		       {
		         "id": "fc3dc670a4be9b9a",
		         "name": "label",
		         "properties": {
		           "color": "fff000"
		         }
		       }
		     ],
		     "links": {
		       "labels": "/api/v2/notificationEndpoints/c0175f0077a77005/labels",
		       "members": "/api/v2/notificationEndpoints/c0175f0077a77005/members",
		       "owners": "/api/v2/notificationEndpoints/c0175f0077a77005/owners",
		       "self": "/api/v2/notificationEndpoints/c0175f0077a77005"
		     },
		     "name": "example",
			 "orgID": "7e55e118dbabb1ed",
			 "authMethod": "basic",
             "contentTemplate": "template",
			 "password": "secret: http-password-key",
			 "token":"",
  			 "method": "POST",
		     "status": "inactive",
			 "type": "http",
			 "headers": {
				"x-header-1": "header 1",
				"x-header-2": "header 2"
			 },
		     "updatedAt": "0001-01-01T00:00:00Z",
		     "username": "secret: http-user-key"
		   }
		   ]
		}`,
			},
		},
		{
			name: "get all notification endpoints when there are none",
			fields: fields{
				&mock.NotificationEndpointService{
					FindNotificationEndpointsF: func(ctx context.Context, filter influxdb.NotificationEndpointFilter, opts ...influxdb.FindOptions) ([]influxdb.NotificationEndpoint, int, error) {
						return []influxdb.NotificationEndpoint{}, 0, nil
					},
				},
				&mock.LabelService{},
			},
			args: args{
				map[string][]string{
					"limit": {"1"},
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/notificationEndpoints?descending=false&limit=1&offset=0"
  },
  "notificationEndpoints": []
}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notificationEndpointBackend := NewMockNotificationEndpointBackend()
			notificationEndpointBackend.NotificationEndpointService = tt.fields.NotificationEndpointService
			notificationEndpointBackend.LabelService = tt.fields.LabelService
			h := NewNotificationEndpointHandler(notificationEndpointBackend)

			r := httptest.NewRequest("GET", "http://any.url", nil)

			qp := r.URL.Query()
			for k, vs := range tt.args.queryParams {
				for _, v := range vs {
					qp.Add(k, v)
				}
			}
			r.URL.RawQuery = qp.Encode()

			w := httptest.NewRecorder()

			h.handleGetNotificationEndpoints(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetNotificationEndpoints() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetNotificationEndpoints() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil || tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetNotificationEndpoints() = ***%v***", tt.name, diff)
			}
		})
	}
}

func TestService_handleGetNotificationEndpoint(t *testing.T) {
	type fields struct {
		NotificationEndpointService influxdb.NotificationEndpointService
	}
	type args struct {
		id string
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
			name: "get a notification endpoint by id",
			fields: fields{
				&mock.NotificationEndpointService{
					FindNotificationEndpointByIDF: func(ctx context.Context, id influxdb.ID) (influxdb.NotificationEndpoint, error) {
						if id == influxTesting.MustIDBase16("020f755c3c082000") {
							return &endpoint.HTTP{
								Base: endpoint.Base{
									ID:     influxTesting.MustIDBase16("020f755c3c082000"),
									OrgID:  influxTesting.MustIDBase16("020f755c3c082000"),
									Name:   "hello",
									Status: influxdb.Active,
								},
								URL:             "example.com",
								Username:        influxdb.SecretField{Key: "http-user-key"},
								Password:        influxdb.SecretField{Key: "http-password-key"},
								AuthMethod:      "basic",
								Method:          "POST",
								ContentTemplate: "template",
							}, nil
						}
						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
		{
		  "links": {
		    "self": "/api/v2/notificationEndpoints/020f755c3c082000",
		    "labels": "/api/v2/notificationEndpoints/020f755c3c082000/labels",
		    "members": "/api/v2/notificationEndpoints/020f755c3c082000/members",
		    "owners": "/api/v2/notificationEndpoints/020f755c3c082000/owners"
		  },
		  "labels": [],
		  "authMethod": "basic",
		  "method": "POST",
		  "contentTemplate": "template",
		  "createdAt": "0001-01-01T00:00:00Z",
		  "updatedAt": "0001-01-01T00:00:00Z",
		  "id": "020f755c3c082000",
		  "url": "example.com",
		  "username": "secret: http-user-key",
		  "password": "secret: http-password-key",
		  "token":"",
		  "status": "active",
          "type": "http",
		  "orgID": "020f755c3c082000",
		  "name": "hello"
		}
		`,
			},
		},
		{
			name: "not found",
			fields: fields{
				&mock.NotificationEndpointService{
					FindNotificationEndpointByIDF: func(ctx context.Context, id influxdb.ID) (influxdb.NotificationEndpoint, error) {
						return nil, &influxdb.Error{
							Code: influxdb.ENotFound,
							Msg:  "notification endpoint not found",
						}
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notificationEndpointBackend := NewMockNotificationEndpointBackend()
			notificationEndpointBackend.HTTPErrorHandler = ErrorHandler(0)
			notificationEndpointBackend.NotificationEndpointService = tt.fields.NotificationEndpointService
			h := NewNotificationEndpointHandler(notificationEndpointBackend)

			r := httptest.NewRequest("GET", "http://any.url", nil)

			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handleGetNotificationEndpoint(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)
			t.Logf(res.Header.Get("X-Influx-Error"))

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetNotificationEndpoint() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetNotificationEndpoint() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handleGetNotificationEndpoint(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handleGetNotificationEndpoint() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestService_handlePostNotificationEndpoint(t *testing.T) {
	type fields struct {
		Secrets                     map[string]string
		SecretService               influxdb.SecretService
		NotificationEndpointService influxdb.NotificationEndpointService
		OrganizationService         influxdb.OrganizationService
	}
	type args struct {
		endpoint interface{}
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
		secrets     map[string]string
	}

	var secrets map[string]string

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "create a new notification endpoint",
			fields: fields{
				Secrets: map[string]string{},
				SecretService: &mock.SecretService{
					PutSecretFn: func(ctx context.Context, orgID influxdb.ID, k string, v string) error {
						secrets[orgID.String()+"-"+k] = v
						return nil
					},
				},
				NotificationEndpointService: &mock.NotificationEndpointService{
					CreateNotificationEndpointF: func(ctx context.Context, edp influxdb.NotificationEndpoint, userID influxdb.ID) error {
						edp.SetID(influxTesting.MustIDBase16("020f755c3c082000"))
						edp.BackfillSecretKeys()
						return nil
					},
				},
				OrganizationService: &mock.OrganizationService{
					FindOrganizationF: func(ctx context.Context, f influxdb.OrganizationFilter) (*influxdb.Organization, error) {
						return &influxdb.Organization{ID: influxTesting.MustIDBase16("6f626f7274697320")}, nil
					},
				},
			},
			args: args{
				endpoint: map[string]interface{}{
					"name":            "hello",
					"type":            "http",
					"orgID":           "6f626f7274697320",
					"description":     "desc1",
					"status":          "active",
					"url":             "example.com",
					"username":        "user1",
					"password":        "password1",
					"authMethod":      "basic",
					"method":          "POST",
					"contentTemplate": "template",
				},
			},
			wants: wants{
				secrets: map[string]string{
					"6f626f7274697320-020f755c3c082000-password": "password1",
					"6f626f7274697320-020f755c3c082000-username": "user1",
				},
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "self": "/api/v2/notificationEndpoints/020f755c3c082000",
    "labels": "/api/v2/notificationEndpoints/020f755c3c082000/labels",
    "members": "/api/v2/notificationEndpoints/020f755c3c082000/members",
    "owners": "/api/v2/notificationEndpoints/020f755c3c082000/owners"
  },
  "url": "example.com",
  "status": "active",
  "username": "secret: 020f755c3c082000-username",
  "password": "secret: 020f755c3c082000-password",
  "token":"",
  "authMethod": "basic",
  "contentTemplate": "template",
  "type": "http",
  "method": "POST",
  "createdAt": "0001-01-01T00:00:00Z",
  "updatedAt": "0001-01-01T00:00:00Z",
  "id": "020f755c3c082000",
  "orgID": "6f626f7274697320",
  "name": "hello",
  "description": "desc1",
  "labels": []
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets = tt.fields.Secrets
			notificationEndpointBackend := NewMockNotificationEndpointBackend()
			notificationEndpointBackend.NotificationEndpointService = tt.fields.NotificationEndpointService
			notificationEndpointBackend.OrganizationService = tt.fields.OrganizationService
			notificationEndpointBackend.SecretService = tt.fields.SecretService
			h := NewNotificationEndpointHandler(notificationEndpointBackend)

			b, err := json.Marshal(tt.args.endpoint)
			if err != nil {
				t.Fatalf("failed to unmarshal endpoint: %v", err)
			}
			r := httptest.NewRequest("GET", "http://any.url?org=30", bytes.NewReader(b))
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), &influxdb.Session{UserID: user1ID}))
			w := httptest.NewRecorder()

			h.handlePostNotificationEndpoint(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostNotificationEndpoint() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostNotificationEndpoint() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handlePostNotificationEndpoint(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handlePostNotificationEndpoint() = ***%s***", tt.name, diff)
				}
			}
			if diff := cmp.Diff(secrets, tt.wants.secrets); diff != "" {
				t.Errorf("%q. handlePostNotificationEndpoint secrets are different ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handleDeleteNotificationEndpoint(t *testing.T) {
	var secrets map[string]string
	type fields struct {
		Secrets                     map[string]string
		SecretService               influxdb.SecretService
		NotificationEndpointService influxdb.NotificationEndpointService
	}
	type args struct {
		id string
	}
	type wants struct {
		secrets     map[string]string
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
			name: "remove a notification endpoint by id",
			fields: fields{
				Secrets: map[string]string{
					"020f755c3c082001-k1": "v1",
					"020f755c3c082001-k2": "v2",
				},
				SecretService: &mock.SecretService{
					DeleteSecretFn: func(ctx context.Context, orgID influxdb.ID, ks ...string) error {
						for _, k := range ks {
							delete(secrets, orgID.String()+"-"+k)
						}
						return nil
					},
				},
				NotificationEndpointService: &mock.NotificationEndpointService{
					DeleteNotificationEndpointF: func(ctx context.Context, id influxdb.ID) ([]influxdb.SecretField, influxdb.ID, error) {
						if id == influxTesting.MustIDBase16("020f755c3c082000") {
							return []influxdb.SecretField{
								{Key: "k1"},
							}, influxTesting.MustIDBase16("020f755c3c082001"), nil
						}

						return nil, 0, fmt.Errorf("wrong id")
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				secrets: map[string]string{
					"020f755c3c082001-k2": "v2",
				},
				statusCode: http.StatusNoContent,
			},
		},
		{
			name: "notification endpoint not found",
			fields: fields{
				NotificationEndpointService: &mock.NotificationEndpointService{
					DeleteNotificationEndpointF: func(ctx context.Context, id influxdb.ID) ([]influxdb.SecretField, influxdb.ID, error) {
						return nil, 0, &influxdb.Error{
							Code: influxdb.ENotFound,
							Msg:  "notification endpoint not found",
						}
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets = tt.fields.Secrets

			notificationEndpointBackend := NewMockNotificationEndpointBackend()
			notificationEndpointBackend.HTTPErrorHandler = ErrorHandler(0)
			notificationEndpointBackend.NotificationEndpointService = tt.fields.NotificationEndpointService
			notificationEndpointBackend.SecretService = tt.fields.SecretService
			h := NewNotificationEndpointHandler(notificationEndpointBackend)

			r := httptest.NewRequest("GET", "http://any.url", nil)

			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handleDeleteNotificationEndpoint(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleDeleteNotificationEndpoint() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleDeleteNotificationEndpoint() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handleDeleteNotificationEndpoint(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handleDeleteNotificationEndpoint() = ***%s***", tt.name, diff)
				}
			}

			if diff := cmp.Diff(secrets, tt.wants.secrets); diff != "" {
				t.Errorf("%q. handlePostNotificationEndpoint secrets are different ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handlePatchNotificationEndpoint(t *testing.T) {
	type fields struct {
		NotificationEndpointService influxdb.NotificationEndpointService
	}
	type args struct {
		id   string
		name string
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
			name: "update a notification endpoint name",
			fields: fields{
				&mock.NotificationEndpointService{
					PatchNotificationEndpointF: func(ctx context.Context, id influxdb.ID, upd influxdb.NotificationEndpointUpdate) (influxdb.NotificationEndpoint, error) {
						if id == influxTesting.MustIDBase16("020f755c3c082000") {
							d := &endpoint.Slack{
								Base: endpoint.Base{
									ID:     influxTesting.MustIDBase16("020f755c3c082000"),
									Name:   "hello",
									OrgID:  influxTesting.MustIDBase16("020f755c3c082000"),
									Status: influxdb.Active,
								},
								URL: "http://example.com",
							}

							if upd.Name != nil {
								d.Name = *upd.Name
							}

							return d, nil
						}

						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id:   "020f755c3c082000",
				name: "example",
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
		{
		  "links": {
		    "self": "/api/v2/notificationEndpoints/020f755c3c082000",
		    "labels": "/api/v2/notificationEndpoints/020f755c3c082000/labels",
		    "members": "/api/v2/notificationEndpoints/020f755c3c082000/members",
		    "owners": "/api/v2/notificationEndpoints/020f755c3c082000/owners"
		  },
		  "createdAt": "0001-01-01T00:00:00Z",
		  "updatedAt": "0001-01-01T00:00:00Z",
		  "id": "020f755c3c082000",
		  "orgID": "020f755c3c082000",
		  "url": "http://example.com",
		  "name": "example",
		  "status": "active",
		  "type": "slack",
		  "token": "",
		  "labels": []
		}
		`,
			},
		},
		{
			name: "notification endpoint not found",
			fields: fields{
				&mock.NotificationEndpointService{
					PatchNotificationEndpointF: func(ctx context.Context, id influxdb.ID, upd influxdb.NotificationEndpointUpdate) (influxdb.NotificationEndpoint, error) {
						return nil, &influxdb.Error{
							Code: influxdb.ENotFound,
							Msg:  "notification endpoint not found",
						}
					},
				},
			},
			args: args{
				id:   "020f755c3c082000",
				name: "hello",
			},
			wants: wants{
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notificationEndpointBackend := NewMockNotificationEndpointBackend()
			notificationEndpointBackend.HTTPErrorHandler = ErrorHandler(0)
			notificationEndpointBackend.NotificationEndpointService = tt.fields.NotificationEndpointService
			h := NewNotificationEndpointHandler(notificationEndpointBackend)

			upd := influxdb.NotificationEndpointUpdate{}
			if tt.args.name != "" {
				upd.Name = &tt.args.name
			}

			b, err := json.Marshal(upd)
			if err != nil {
				t.Fatalf("failed to unmarshal notification endpoint update: %v", err)
			}

			r := httptest.NewRequest("GET", "http://any.url", bytes.NewReader(b))
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), &influxdb.Session{UserID: user1ID}))

			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))

			w := httptest.NewRecorder()

			h.handlePatchNotificationEndpoint(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePatchNotificationEndpoint() = %v, want %v %v", tt.name, res.StatusCode, tt.wants.statusCode, w.Header())
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePatchNotificationEndpoint() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handlePatchNotificationEndpoint(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handlePatchNotificationEndpoint() = ***%s***", tt.name, diff)
				}
			}
		})
	}
}

func TestService_handleUpdateNotificationEndpoint(t *testing.T) {
	var secrets map[string]string
	type fields struct {
		Secrets                     map[string]string
		SecretService               influxdb.SecretService
		NotificationEndpointService influxdb.NotificationEndpointService
	}
	type args struct {
		id  string
		edp map[string]interface{}
	}
	type wants struct {
		secrets     map[string]string
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
			name: "update a notification endpoint name",
			fields: fields{
				SecretService: &mock.SecretService{
					PutSecretFn: func(ctx context.Context, orgID influxdb.ID, k string, v string) error {
						secrets[orgID.String()+"-"+k] = v
						return nil
					},
				},
				NotificationEndpointService: &mock.NotificationEndpointService{
					UpdateNotificationEndpointF: func(ctx context.Context, id influxdb.ID, edp influxdb.NotificationEndpoint, userID influxdb.ID) (influxdb.NotificationEndpoint, error) {
						if id == influxTesting.MustIDBase16("020f755c3c082000") {
							edp.SetID(id)
							edp.BackfillSecretKeys()
							return edp, nil
						}

						return nil, fmt.Errorf("not found")
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
				edp: map[string]interface{}{
					"name":   "example",
					"status": "active",
					"orgID":  "020f755c3c082001",
					"type":   "slack",
					"token":  "",
					"url":    "example.com",
				},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
		{
		  "links": {
		    "self": "/api/v2/notificationEndpoints/020f755c3c082000",
		    "labels": "/api/v2/notificationEndpoints/020f755c3c082000/labels",
		    "members": "/api/v2/notificationEndpoints/020f755c3c082000/members",
		    "owners": "/api/v2/notificationEndpoints/020f755c3c082000/owners"
		  },
		  "createdAt": "0001-01-01T00:00:00Z",
		  "updatedAt": "0001-01-01T00:00:00Z",
		  "id": "020f755c3c082000",
		  "orgID": "020f755c3c082001",
		  "name": "example",
		  "url": "example.com",
          "type": "slack",
		  "status": "active",
		  "token": "",
          "labels": []
		}
		`,
			},
		},
		{
			name: "notification endpoint not found",
			fields: fields{
				Secrets: map[string]string{},
				NotificationEndpointService: &mock.NotificationEndpointService{
					UpdateNotificationEndpointF: func(ctx context.Context, id influxdb.ID, edp influxdb.NotificationEndpoint, userID influxdb.ID) (influxdb.NotificationEndpoint, error) {
						return nil, &influxdb.Error{
							Code: influxdb.ENotFound,
							Msg:  "notification endpoint not found",
						}
					},
				},
			},
			args: args{
				id: "020f755c3c082000",
				edp: map[string]interface{}{
					"type": "slack",
					"name": "example",
				},
			},
			wants: wants{
				secrets:    map[string]string{},
				statusCode: http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets = tt.fields.Secrets
			notificationEndpointBackend := NewMockNotificationEndpointBackend()
			notificationEndpointBackend.HTTPErrorHandler = ErrorHandler(0)
			notificationEndpointBackend.NotificationEndpointService = tt.fields.NotificationEndpointService
			notificationEndpointBackend.SecretService = tt.fields.SecretService
			h := NewNotificationEndpointHandler(notificationEndpointBackend)

			b, err := json.Marshal(tt.args.edp)
			if err != nil {
				t.Fatalf("failed to unmarshal notification endpoint update: %v", err)
			}

			r := httptest.NewRequest("PUT", "http://any.url", bytes.NewReader(b))
			r = r.WithContext(context.WithValue(
				context.Background(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "id",
						Value: tt.args.id,
					},
				}))
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), &influxdb.Session{UserID: user1ID}))
			w := httptest.NewRecorder()

			h.handlePutNotificationEndpoint(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePutNotificationEndpoint() = %v, want %v %v", tt.name, res.StatusCode, tt.wants.statusCode, w.Header())
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePutNotificationEndpoint() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handlePutNotificationEndpoint(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handlePutNotificationEndpoint() = ***%s***", tt.name, diff)
				}
			}
			if diff := cmp.Diff(secrets, tt.wants.secrets); diff != "" {
				t.Errorf("%q. handlePostNotificationEndpoint secrets are different ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handlePostNotificationEndpointMember(t *testing.T) {
	type fields struct {
		UserService influxdb.UserService
	}
	type args struct {
		notificationEndpointID string
		user                   *influxdb.User
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
			name: "add a notification endpoint member",
			fields: fields{
				UserService: &mock.UserService{
					FindUserByIDFn: func(ctx context.Context, id influxdb.ID) (*influxdb.User, error) {
						return &influxdb.User{
							ID:     id,
							Name:   "name",
							Status: influxdb.Active,
						}, nil
					},
				},
			},
			args: args{
				notificationEndpointID: "020f755c3c082000",
				user: &influxdb.User{
					ID: influxTesting.MustIDBase16("6f626f7274697320"),
				},
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "logs": "/api/v2/users/6f626f7274697320/logs",
    "self": "/api/v2/users/6f626f7274697320"
  },
  "role": "member",
  "id": "6f626f7274697320",
	"name": "name",
	"status": "active"
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notificationEndpointBackend := NewMockNotificationEndpointBackend()
			notificationEndpointBackend.UserService = tt.fields.UserService
			h := NewNotificationEndpointHandler(notificationEndpointBackend)

			b, err := json.Marshal(tt.args.user)
			if err != nil {
				t.Fatalf("failed to marshal user: %v", err)
			}

			path := fmt.Sprintf("/api/v2/notificationEndpoints/%s/members", tt.args.notificationEndpointID)
			r := httptest.NewRequest("POST", path, bytes.NewReader(b))
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), &influxdb.Session{UserID: user1ID}))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostNotificationEndpointMember() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostNotificationEndpointMember() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
				t.Errorf("%q, handlePostNotificationEndpointMember(). error unmarshaling json %v", tt.name, err)
			} else if tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePostNotificationEndpointMember() = ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handlePostNotificationEndpointOwner(t *testing.T) {
	type fields struct {
		UserService influxdb.UserService
	}
	type args struct {
		notificationEndpointID string
		user                   *influxdb.User
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	cases := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "add a notification endpoint owner",
			fields: fields{
				UserService: &mock.UserService{
					FindUserByIDFn: func(ctx context.Context, id influxdb.ID) (*influxdb.User, error) {
						return &influxdb.User{
							ID:     id,
							Name:   "name",
							Status: influxdb.Active,
						}, nil
					},
				},
			},
			args: args{
				notificationEndpointID: "020f755c3c082000",
				user: &influxdb.User{
					ID: influxTesting.MustIDBase16("6f626f7274697320"),
				},
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "logs": "/api/v2/users/6f626f7274697320/logs",
    "self": "/api/v2/users/6f626f7274697320"
  },
  "role": "owner",
  "id": "6f626f7274697320",
	"name": "name",
	"status": "active"
}
`,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			notificationEndpointBackend := NewMockNotificationEndpointBackend()
			notificationEndpointBackend.UserService = tt.fields.UserService
			h := NewNotificationEndpointHandler(notificationEndpointBackend)

			b, err := json.Marshal(tt.args.user)
			if err != nil {
				t.Fatalf("failed to marshal user: %v", err)
			}

			path := fmt.Sprintf("/api/v2/notificationEndpoints/%s/owners", tt.args.notificationEndpointID)
			r := httptest.NewRequest("POST", path, bytes.NewReader(b))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostNotificationEndpointOwner() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostNotificationEndpointOwner() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
				t.Errorf("%q, handlePostNotificationEndpointOwner(). error unmarshaling json %v", tt.name, err)
			} else if tt.wants.body != "" && !eq {
				t.Errorf("%q. handlePostNotificationEndpointOwner() = ***%s***", tt.name, diff)
			}
		})
	}
}
