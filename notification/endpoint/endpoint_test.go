package endpoint_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/mock"
	"github.com/influxdata/influxdb/notification/endpoint"
	influxTesting "github.com/influxdata/influxdb/testing"
)

const (
	id1 = "020f755c3c082000"
	id3 = "020f755c3c082002"
)

var goodBase = endpoint.Base{
	ID:          influxTesting.MustIDBase16(id1),
	Name:        "name1",
	OrgID:       influxTesting.MustIDBase16(id3),
	Status:      influxdb.Active,
	Description: "desc1",
}

func TestValidEndpoint(t *testing.T) {
	cases := []struct {
		name string
		src  influxdb.NotificationEndpoint
		err  error
	}{
		{
			name: "invalid endpoint id",
			src:  &endpoint.Slack{},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "Notification Endpoint ID is invalid",
			},
		},
		{
			name: "invalid status",
			src: &endpoint.PagerDuty{
				Base: endpoint.Base{
					ID:    influxTesting.MustIDBase16(id1),
					Name:  "name1",
					OrgID: influxTesting.MustIDBase16(id3),
				},
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "invalid status",
			},
		},
		{
			name: "empty name",
			src: &endpoint.PagerDuty{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
				},
				ClientURL:  "https://events.pagerduty.com/v2/enqueue",
				RoutingKey: influxdb.SecretField{Key: id1 + "-routing-key"},
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "Notification Endpoint Name can't be empty",
			},
		},
		{
			name: "empty slack url and token",
			src: &endpoint.Slack{
				Base: goodBase,
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "slack endpoint URL and token are empty",
			},
		},
		{
			name: "invalid slack url",
			src: &endpoint.Slack{
				Base: goodBase,
				URL:  "posts://er:{DEf1=ghi@:5432/db?ssl",
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "slack endpoint URL is invalid: parse posts://er:{DEf1=ghi@:5432/db?ssl: net/url: invalid userinfo",
			},
		},
		{
			name: "empty slack token",
			src: &endpoint.Slack{
				Base: goodBase,
				URL:  "localhost",
			},
			err: nil,
		},
		{
			name: "invalid slack token",
			src: &endpoint.Slack{
				Base:  goodBase,
				URL:   "localhost",
				Token: influxdb.SecretField{Key: "bad-key"},
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "slack endpoint token is invalid",
			},
		},
		{
			name: "empty pagerduty url",
			src: &endpoint.PagerDuty{
				Base: goodBase,
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "pagerduty endpoint ClientURL is empty",
			},
		},
		{
			name: "invalid routine key",
			src: &endpoint.PagerDuty{
				Base:       goodBase,
				ClientURL:  "localhost",
				RoutingKey: influxdb.SecretField{Key: "bad-key"},
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "pagerduty routing key is invalid",
			},
		},
		{
			name: "empty http http method",
			src: &endpoint.HTTP{
				Base: goodBase,
				URL:  "localhost",
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "invalid http http method",
			},
		},
		{
			name: "empty http token",
			src: &endpoint.HTTP{
				Base:       goodBase,
				URL:        "localhost",
				Method:     "GET",
				AuthMethod: "bearer",
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "invalid http token for bearer auth",
			},
		},
		{
			name: "empty http username",
			src: &endpoint.HTTP{
				Base:       goodBase,
				URL:        "localhost",
				Method:     http.MethodGet,
				AuthMethod: "basic",
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "invalid http username/password for basic auth",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.src.Valid()
			influxTesting.ErrorsEqual(t, got, c.err)
		})
	}
}

var timeGen1 = mock.TimeGenerator{FakeValue: time.Date(2006, time.July, 13, 4, 19, 10, 0, time.UTC)}
var timeGen2 = mock.TimeGenerator{FakeValue: time.Date(2006, time.July, 14, 5, 23, 53, 10, time.UTC)}

func TestJSON(t *testing.T) {
	cases := []struct {
		name string
		src  influxdb.NotificationEndpoint
	}{
		{
			name: "simple Slack",
			src: &endpoint.Slack{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				URL:   "https://slack.com/api/chat.postMessage",
				Token: influxdb.SecretField{Key: "token-key-1"},
			},
		},
		{
			name: "Slack without token",
			src: &endpoint.Slack{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				URL: "https://hooks.slack.com/services/x/y/z",
			},
		},
		{
			name: "simple pagerduty",
			src: &endpoint.PagerDuty{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				ClientURL:  "https://events.pagerduty.com/v2/enqueue",
				RoutingKey: influxdb.SecretField{Key: "pagerduty-routing-key"},
			},
		},
		{
			name: "simple http",
			src: &endpoint.HTTP{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				Headers: map[string]string{
					"x-header-1": "header 1",
					"x-header-2": "header 2",
				},
				AuthMethod: "basic",
				URL:        "http://example.com",
				Username:   influxdb.SecretField{Key: "username-key"},
				Password:   influxdb.SecretField{Key: "password-key"},
			},
		},
	}
	for _, c := range cases {
		b, err := json.Marshal(c.src)
		if err != nil {
			t.Fatalf("%s marshal failed, err: %s", c.name, err.Error())
		}
		got, err := endpoint.UnmarshalJSON(b)
		if err != nil {
			t.Fatalf("%s unmarshal failed, err: %s", c.name, err.Error())
		}
		if diff := cmp.Diff(got, c.src); diff != "" {
			t.Errorf("failed %s, NotificationEndpoint are different -got/+want\ndiff %s", c.name, diff)
		}
	}
}

func TestBackFill(t *testing.T) {
	cases := []struct {
		name   string
		src    influxdb.NotificationEndpoint
		target influxdb.NotificationEndpoint
	}{
		{
			name: "simple Slack",
			src: &endpoint.Slack{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				URL: "https://slack.com/api/chat.postMessage",
				Token: influxdb.SecretField{
					Value: strPtr("token-value"),
				},
			},
			target: &endpoint.Slack{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				URL: "https://slack.com/api/chat.postMessage",
				Token: influxdb.SecretField{
					Key:   id1 + "-token",
					Value: strPtr("token-value"),
				},
			},
		},
		{
			name: "simple pagerduty",
			src: &endpoint.PagerDuty{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				ClientURL: "https://events.pagerduty.com/v2/enqueue",
				RoutingKey: influxdb.SecretField{
					Value: strPtr("routing-key-value"),
				},
			},
			target: &endpoint.PagerDuty{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				ClientURL: "https://events.pagerduty.com/v2/enqueue",
				RoutingKey: influxdb.SecretField{
					Key:   id1 + "-routing-key",
					Value: strPtr("routing-key-value"),
				},
			},
		},
		{
			name: "http with token",
			src: &endpoint.HTTP{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				AuthMethod: "basic",
				URL:        "http://example.com",
				Username: influxdb.SecretField{
					Value: strPtr("username1"),
				},
				Password: influxdb.SecretField{
					Value: strPtr("password1"),
				},
			},
			target: &endpoint.HTTP{
				Base: endpoint.Base{
					ID:     influxTesting.MustIDBase16(id1),
					Name:   "name1",
					OrgID:  influxTesting.MustIDBase16(id3),
					Status: influxdb.Active,
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				AuthMethod: "basic",
				URL:        "http://example.com",
				Username: influxdb.SecretField{
					Key:   id1 + "-username",
					Value: strPtr("username1"),
				},
				Password: influxdb.SecretField{
					Key:   id1 + "-password",
					Value: strPtr("password1"),
				},
			},
		},
	}
	for _, c := range cases {
		c.src.BackfillSecretKeys()
		if diff := cmp.Diff(c.target, c.src); diff != "" {
			t.Errorf("failed %s, NotificationEndpoint are different -got/+want\ndiff %s", c.name, diff)
		}
	}
}

func strPtr(s string) *string {
	ss := new(string)
	*ss = s
	return ss
}
