package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/influxdata/flux"
	"github.com/influxdata/flux/csv"
	"github.com/influxdata/flux/lang"
	"github.com/influxdata/influxdb"
	platform "github.com/influxdata/influxdb"
	icontext "github.com/influxdata/influxdb/context"
	"github.com/influxdata/influxdb/http/metric"
	"github.com/influxdata/influxdb/inmem"
	"github.com/influxdata/influxdb/kit/check"
	"github.com/influxdata/influxdb/query"
	"github.com/influxdata/influxdb/query/mock"
	"go.uber.org/zap/zaptest"
)

func TestFluxService_Query(t *testing.T) {
	orgID, err := influxdb.IDFromString("abcdabcdabcdabcd")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		token   string
		ctx     context.Context
		r       *query.ProxyRequest
		status  int
		want    flux.Statistics
		wantW   string
		wantErr bool
	}{
		{
			name:  "query",
			ctx:   context.Background(),
			token: "mytoken",
			r: &query.ProxyRequest{
				Request: query.Request{
					OrganizationID: *orgID,
					Compiler: lang.FluxCompiler{
						Query: "from()",
					},
				},
				Dialect: csv.DefaultDialect(),
			},
			status: http.StatusOK,
			want:   flux.Statistics{},
			wantW:  "howdy\n",
		},
		{
			name:  "missing org id",
			ctx:   context.Background(),
			token: "mytoken",
			r: &query.ProxyRequest{
				Request: query.Request{
					Compiler: lang.FluxCompiler{
						Query: "from()",
					},
				},
				Dialect: csv.DefaultDialect(),
			},
			wantErr: true,
		},
		{
			name:  "error status",
			token: "mytoken",
			ctx:   context.Background(),
			r: &query.ProxyRequest{
				Request: query.Request{
					OrganizationID: *orgID,
					Compiler: lang.FluxCompiler{
						Query: "from()",
					},
				},
				Dialect: csv.DefaultDialect(),
			},
			status:  http.StatusUnauthorized,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if reqID := r.URL.Query().Get(OrgID); reqID == "" {
					if name := r.URL.Query().Get(OrgName); name == "" {
						// Request must have org or orgID.
						ErrorHandler(0).HandleHTTPError(context.TODO(), influxdb.ErrInvalidOrgFilter, w)
						return
					}
				}
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprintln(w, "howdy")
			}))
			defer ts.Close()
			s := &FluxService{
				Addr:  ts.URL,
				Token: tt.token,
			}

			w := &bytes.Buffer{}
			got, err := s.Query(tt.ctx, w, tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("FluxService.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("FluxService.Query() = -want/+got: %v", diff)
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("FluxService.Query() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestFluxQueryService_Query(t *testing.T) {
	var orgID platform.ID
	orgID.DecodeFromString("aaaaaaaaaaaaaaaa")
	tests := []struct {
		name    string
		token   string
		ctx     context.Context
		r       *query.Request
		csv     string
		status  int
		want    string
		wantErr bool
	}{
		{
			name:  "error status",
			token: "mytoken",
			ctx:   context.Background(),
			r: &query.Request{
				OrganizationID: orgID,
				Compiler: lang.FluxCompiler{
					Query: "from()",
				},
			},
			status:  http.StatusUnauthorized,
			wantErr: true,
		},
		{
			name:  "returns csv",
			token: "mytoken",
			ctx:   context.Background(),
			r: &query.Request{
				OrganizationID: orgID,
				Compiler: lang.FluxCompiler{
					Query: "from()",
				},
			},
			status: http.StatusOK,
			csv: `#datatype,string,long,dateTime:RFC3339,double,long,string,boolean,string,string,string
#group,false,false,false,false,false,false,false,true,true,true
#default,_result,,,,,,,,,
,result,table,_time,usage_user,test,mystr,this,cpu,host,_measurement
,,0,2018-08-29T13:08:47Z,10.2,10,yay,true,cpu-total,a,cpui
`,
			want: toCRLF(`,_result,0,2018-08-29T13:08:47Z,10.2,10,yay,true,cpu-total,a,cpui

`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var orgIDStr string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				orgIDStr = r.URL.Query().Get(OrgID)
				w.WriteHeader(tt.status)
				fmt.Fprintln(w, tt.csv)
			}))
			s := &FluxQueryService{
				Addr:  ts.URL,
				Token: tt.token,
			}
			res, err := s.Query(tt.ctx, tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("FluxQueryService.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if res != nil && res.Err() != nil {
				t.Errorf("FluxQueryService.Query() result error = %v", res.Err())
				return
			}
			if tt.wantErr {
				return
			}
			defer res.Release()

			enc := csv.NewMultiResultEncoder(csv.ResultEncoderConfig{
				NoHeader:  true,
				Delimiter: ',',
			})
			b := bytes.Buffer{}
			n, err := enc.Encode(&b, res)
			if err != nil {
				t.Errorf("FluxQueryService.Query() encode error = %v", err)
				return
			}
			if n != int64(len(tt.want)) {
				t.Errorf("FluxQueryService.Query() encode result = %d, want %d", n, len(tt.want))
			}
			if orgIDStr == "" {
				t.Error("FluxQueryService.Query() encoded orgID is empty")
			}
			if got, want := orgIDStr, tt.r.OrganizationID.String(); got != want {
				t.Errorf("FluxQueryService.Query() encoded orgID = %s, want %s", got, want)
			}

			got := b.String()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FluxQueryService.Query() =\n%s\n%s", got, tt.want)
			}
		})
	}
}

func TestFluxHandler_postFluxAST(t *testing.T) {
	tests := []struct {
		name   string
		w      *httptest.ResponseRecorder
		r      *http.Request
		want   string
		status int
	}{
		{
			name: "get ast from()",
			w:    httptest.NewRecorder(),
			r:    httptest.NewRequest("POST", "/api/v2/query/ast", bytes.NewBufferString(`{"query": "from()"}`)),
			want: `{"ast":{"type":"Package","package":"main","files":[{"type":"File","location":{"start":{"line":1,"column":1},"end":{"line":1,"column":7},"source":"from()"},"package":null,"imports":null,"body":[{"type":"ExpressionStatement","location":{"start":{"line":1,"column":1},"end":{"line":1,"column":7},"source":"from()"},"expression":{"type":"CallExpression","location":{"start":{"line":1,"column":1},"end":{"line":1,"column":7},"source":"from()"},"callee":{"type":"Identifier","location":{"start":{"line":1,"column":1},"end":{"line":1,"column":5},"source":"from"},"name":"from"}}}]}]}}
`,
			status: http.StatusOK,
		},
		{
			name:   "error from bad json",
			w:      httptest.NewRecorder(),
			r:      httptest.NewRequest("POST", "/api/v2/query/ast", bytes.NewBufferString(`error!`)),
			want:   `{"code":"invalid","message":"invalid json","error":"invalid character 'e' looking for beginning of value"}`,
			status: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &FluxHandler{
				HTTPErrorHandler: ErrorHandler(0),
			}
			h.postFluxAST(tt.w, tt.r)
			if got := tt.w.Body.String(); got != tt.want {
				t.Errorf("http.postFluxAST = got\n%vwant\n%v", got, tt.want)
			}
			if got := tt.w.Code; got != tt.status {
				t.Errorf("http.postFluxAST = got %d\nwant %d", got, tt.status)
			}
		})
	}
}

func TestFluxService_Check(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(HealthHandler))
	defer ts.Close()
	s := &FluxService{
		Addr: ts.URL,
	}
	got := s.Check(context.Background())
	want := check.Response{
		Name:    "influxdb",
		Status:  "pass",
		Message: "ready for queries and writes",
		Checks:  check.Responses{},
	}
	if !cmp.Equal(want, got) {
		t.Errorf("unexpected response -want/+got: " + cmp.Diff(want, got))
	}
}

func TestFluxQueryService_Check(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(HealthHandler))
	defer ts.Close()
	s := &FluxQueryService{
		Addr: ts.URL,
	}
	got := s.Check(context.Background())
	want := check.Response{
		Name:    "influxdb",
		Status:  "pass",
		Message: "ready for queries and writes",
		Checks:  check.Responses{},
	}
	if !cmp.Equal(want, got) {
		t.Errorf("unexpected response -want/+got: " + cmp.Diff(want, got))
	}
}

var crlfPattern = regexp.MustCompile(`\r?\n`)

func toCRLF(data string) string {
	return crlfPattern.ReplaceAllString(data, "\r\n")
}

type noopEventRecorder struct{}

func (noopEventRecorder) Record(context.Context, metric.Event) {}

var _ metric.EventRecorder = noopEventRecorder{}

// Certain error cases must be encoded as influxdb.Error so they can be properly decoded clientside.
func TestFluxHandler_PostQuery_Errors(t *testing.T) {
	i := inmem.NewService()
	b := &FluxBackend{
		HTTPErrorHandler:    ErrorHandler(0),
		Logger:              zaptest.NewLogger(t),
		QueryEventRecorder:  noopEventRecorder{},
		OrganizationService: i,
		ProxyQueryService: &mock.ProxyQueryService{
			QueryF: func(ctx context.Context, w io.Writer, req *query.ProxyRequest) (flux.Statistics, error) {
				return flux.Statistics{}, &influxdb.Error{
					Code: influxdb.EInvalid,
					Msg:  "some query error",
				}
			},
		},
	}
	h := NewFluxHandler(b)

	t.Run("missing authorizer", func(t *testing.T) {
		ts := httptest.NewServer(h)
		defer ts.Close()

		resp, err := http.Post(ts.URL+"/api/v2/query", "application/json", strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected unauthorized status, got %d", resp.StatusCode)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var ierr influxdb.Error
		if err := json.Unmarshal(body, &ierr); err != nil {
			t.Logf("failed to json unmarshal into influxdb.error: %q", body)
			t.Fatal(err)
		}

		if !strings.Contains(ierr.Msg, "authorization is") {
			t.Fatalf("expected error to mention authorization, got %s", ierr.Msg)
		}
	})

	t.Run("authorizer but syntactically invalid JSON request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/api/v2/query", strings.NewReader("oops"))
		if err != nil {
			t.Fatal(err)
		}
		authz := &influxdb.Authorization{}
		req = req.WithContext(icontext.SetAuthorizer(req.Context(), authz))

		h.handleQuery(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected bad request status, got %d", w.Code)
		}

		body := w.Body.Bytes()
		var ierr influxdb.Error
		if err := json.Unmarshal(body, &ierr); err != nil {
			t.Logf("failed to json unmarshal into influxdb.error: %q", body)
			t.Fatal(err)
		}

		if !strings.Contains(ierr.Msg, "decode request body") {
			t.Fatalf("expected error to mention decoding, got %s", ierr.Msg)
		}
	})

	t.Run("valid request but executing query results in client error", func(t *testing.T) {
		org := influxdb.Organization{Name: t.Name()}
		if err := i.CreateOrganization(context.Background(), &org); err != nil {
			t.Fatal(err)
		}

		req, err := http.NewRequest("POST", "/api/v2/query?orgID="+org.ID.String(), bytes.NewReader([]byte("buckets()")))
		if err != nil {
			t.Fatal(err)
		}
		authz := &influxdb.Authorization{}
		req = req.WithContext(icontext.SetAuthorizer(req.Context(), authz))
		req.Header.Set("Content-Type", "application/vnd.flux")

		w := httptest.NewRecorder()
		h.handleQuery(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected bad request status, got %d", w.Code)
		}

		body := w.Body.Bytes()
		t.Logf("%s", body)
		var ierr influxdb.Error
		if err := json.Unmarshal(body, &ierr); err != nil {
			t.Logf("failed to json unmarshal into influxdb.error: %q", body)
			t.Fatal(err)
		}

		if got, want := ierr.Code, influxdb.EInvalid; got != want {
			t.Fatalf("unexpected error code -want/+got:\n\t- %v\n\t+ %v", want, got)
		}
		if ierr.Msg != "some query error" {
			t.Fatalf("expected error message to mention 'some query error', got %s", ierr.Err.Error())
		}
	})
}
