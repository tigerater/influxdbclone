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

	"go.uber.org/zap"

	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/inmem"
	"github.com/influxdata/influxdb/kv"
	"github.com/influxdata/influxdb/mock"
	platformtesting "github.com/influxdata/influxdb/testing"
)

// NewMockOrgBackend returns a OrgBackend with mock services.
func NewMockOrgBackend() *OrgBackend {
	return &OrgBackend{
		Logger: zap.NewNop().With(zap.String("handler", "org")),

		OrganizationService:             mock.NewOrganizationService(),
		OrganizationOperationLogService: mock.NewOrganizationOperationLogService(),
		UserResourceMappingService:      mock.NewUserResourceMappingService(),
		SecretService:                   mock.NewSecretService(),
		LabelService:                    mock.NewLabelService(),
		UserService:                     mock.NewUserService(),
	}
}

func initOrganizationService(f platformtesting.OrganizationFields, t *testing.T) (platform.OrganizationService, string, func()) {
	t.Helper()
	svc := kv.NewService(inmem.NewKVStore())
	svc.IDGenerator = f.IDGenerator
	svc.OrgBucketIDs = f.OrgBucketIDs
	svc.TimeGenerator = f.TimeGenerator
	if f.TimeGenerator == nil {
		svc.TimeGenerator = platform.RealTimeGenerator{}
	}

	ctx := context.Background()
	if err := svc.Initialize(ctx); err != nil {
		t.Fatal(err)
	}

	for _, o := range f.Organizations {
		if err := svc.PutOrganization(ctx, o); err != nil {
			t.Fatalf("failed to populate organizations")
		}
	}

	orgBackend := NewMockOrgBackend()
	orgBackend.HTTPErrorHandler = ErrorHandler(0)
	orgBackend.OrganizationService = svc
	handler := NewOrgHandler(orgBackend)
	server := httptest.NewServer(handler)
	client := OrganizationService{
		Addr:     server.URL,
		OpPrefix: inmem.OpPrefix,
	}
	done := server.Close

	return &client, inmem.OpPrefix, done
}
func TestOrganizationService(t *testing.T) {

	t.Parallel()
	platformtesting.OrganizationService(initOrganizationService, t)
}

func TestSecretService_handleGetSecrets(t *testing.T) {
	type fields struct {
		SecretService platform.SecretService
	}
	type args struct {
		orgID platform.ID
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
			name: "get basic secrets",
			fields: fields{
				&mock.SecretService{
					GetSecretKeysFn: func(ctx context.Context, orgID platform.ID) ([]string, error) {
						return []string{"hello", "world"}, nil
					},
				},
			},
			args: args{
				orgID: 1,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "org": "/api/v2/orgs/0000000000000001",
    "self": "/api/v2/orgs/0000000000000001/secrets"
  },
  "secrets": [
    "hello",
    "world"
  ]
}
`,
			},
		},
		{
			name: "get secrets when there are none",
			fields: fields{
				&mock.SecretService{
					GetSecretKeysFn: func(ctx context.Context, orgID platform.ID) ([]string, error) {
						return []string{}, nil
					},
				},
			},
			args: args{
				orgID: 1,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `
{
  "links": {
    "org": "/api/v2/orgs/0000000000000001",
    "self": "/api/v2/orgs/0000000000000001/secrets"
  },
  "secrets": []
}
`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgBackend := NewMockOrgBackend()
			orgBackend.HTTPErrorHandler = ErrorHandler(0)
			orgBackend.SecretService = tt.fields.SecretService
			h := NewOrgHandler(orgBackend)

			u := fmt.Sprintf("http://any.url/api/v2/orgs/%s/secrets", tt.args.orgID)
			r := httptest.NewRequest("GET", u, nil)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("handleGetSecrets() = %v, want %v", res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("handleGetSecrets() = %v, want %v", content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handleGetSecrets(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handleGetSecrets() = ***%s***", tt.name, diff)
				}
			}

		})
	}
}

func TestSecretService_handlePatchSecrets(t *testing.T) {
	type fields struct {
		SecretService platform.SecretService
	}
	type args struct {
		orgID   platform.ID
		secrets map[string]string
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
			name: "get basic secrets",
			fields: fields{
				&mock.SecretService{
					PatchSecretsFn: func(ctx context.Context, orgID platform.ID, s map[string]string) error {
						return nil
					},
				},
			},
			args: args{
				orgID: 1,
				secrets: map[string]string{
					"abc": "123",
				},
			},
			wants: wants{
				statusCode: http.StatusNoContent,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgBackend := NewMockOrgBackend()
			orgBackend.HTTPErrorHandler = ErrorHandler(0)
			orgBackend.SecretService = tt.fields.SecretService
			h := NewOrgHandler(orgBackend)

			b, err := json.Marshal(tt.args.secrets)
			if err != nil {
				t.Fatalf("failed to marshal secrets: %v", err)
			}

			buf := bytes.NewReader(b)
			u := fmt.Sprintf("http://any.url/api/v2/orgs/%s/secrets", tt.args.orgID)
			r := httptest.NewRequest("PATCH", u, buf)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("handlePatchSecrets() = %v, want %v", res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("handlePatchSecrets() = %v, want %v", content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handlePatchSecrets(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handlePatchSecrets() = ***%s***", tt.name, diff)
				}
			}

		})
	}
}

func TestSecretService_handleDeleteSecrets(t *testing.T) {
	type fields struct {
		SecretService platform.SecretService
	}
	type args struct {
		orgID   platform.ID
		secrets []string
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
			name: "get basic secrets",
			fields: fields{
				&mock.SecretService{
					DeleteSecretFn: func(ctx context.Context, orgID platform.ID, s ...string) error {
						return nil
					},
				},
			},
			args: args{
				orgID: 1,
				secrets: []string{
					"abc",
				},
			},
			wants: wants{
				statusCode: http.StatusNoContent,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgBackend := NewMockOrgBackend()
			orgBackend.HTTPErrorHandler = ErrorHandler(0)
			orgBackend.SecretService = tt.fields.SecretService
			h := NewOrgHandler(orgBackend)

			b, err := json.Marshal(deleteSecretsRequest{
				Secrets: tt.args.secrets,
			})
			if err != nil {
				t.Fatalf("failed to marshal secrets: %v", err)
			}

			buf := bytes.NewReader(b)
			u := fmt.Sprintf("http://any.url/api/v2/orgs/%s/secrets/delete", tt.args.orgID)
			r := httptest.NewRequest("POST", u, buf)
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("handleDeleteSecrets() = %v, want %v", res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("handleDeleteSecrets() = %v, want %v", content, tt.wants.contentType)
			}
			if tt.wants.body != "" {
				if eq, diff, err := jsonEqual(string(body), tt.wants.body); err != nil {
					t.Errorf("%q, handleDeleteSecrets(). error unmarshaling json %v", tt.name, err)
				} else if !eq {
					t.Errorf("%q. handleDeleteSecrets() = ***%s***", tt.name, diff)
				}
			}

		})
	}
}
