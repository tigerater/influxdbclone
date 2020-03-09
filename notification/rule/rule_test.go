package rule_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/influxdata/influxdb/mock"

	"github.com/google/go-cmp/cmp"
	"github.com/influxdata/influxdb/notification"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/notification/rule"
	influxTesting "github.com/influxdata/influxdb/testing"
)

func lvlPtr(l notification.CheckLevel) *notification.CheckLevel {
	return &l
}

const (
	id1 = "020f755c3c082000"
	id2 = "020f755c3c082001"
	id3 = "020f755c3c082002"
)

var goodBase = rule.Base{
	ID:         influxTesting.MustIDBase16(id1),
	Name:       "name1",
	OwnerID:    influxTesting.MustIDBase16(id2),
	OrgID:      influxTesting.MustIDBase16(id3),
	EndpointID: 1,
}

func TestValidRule(t *testing.T) {
	cases := []struct {
		name string
		src  influxdb.NotificationRule
		err  error
	}{
		{
			name: "invalid rule id",
			src:  &rule.Slack{},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "Notification Rule ID is invalid",
			},
		},
		{
			name: "empty name",
			src: &rule.PagerDuty{
				Base: rule.Base{
					ID: influxTesting.MustIDBase16(id1),
				},
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "Notification Rule Name can't be empty",
			},
		},
		{
			name: "invalid auth id",
			src: &rule.PagerDuty{
				Base: rule.Base{
					ID:   influxTesting.MustIDBase16(id1),
					Name: "name1",
				},
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "Notification Rule OwnerID is invalid",
			},
		},
		{
			name: "invalid org id",
			src: &rule.PagerDuty{
				Base: rule.Base{
					ID:      influxTesting.MustIDBase16(id1),
					Name:    "name1",
					OwnerID: influxTesting.MustIDBase16(id2),
				},
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "Notification Rule OrgID is invalid",
			},
		},
		{
			name: "invalid org id",
			src: &rule.Slack{
				Base: rule.Base{
					ID:         influxTesting.MustIDBase16(id1),
					Name:       "name1",
					OwnerID:    influxTesting.MustIDBase16(id2),
					OrgID:      influxTesting.MustIDBase16(id3),
					EndpointID: 0,
				},
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "Notification Rule EndpointID is invalid",
			},
		},
		{
			name: "offset greater then interval",
			src: &rule.Slack{
				Base: rule.Base{
					ID:         influxTesting.MustIDBase16(id1),
					Name:       "name1",
					OwnerID:    influxTesting.MustIDBase16(id2),
					OrgID:      influxTesting.MustIDBase16(id3),
					EndpointID: 1,
					Every:      mustDuration("1m"),
					Offset:     mustDuration("2m"),
				},
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "Offset should not be equal or greater than the interval",
			},
		},
		{
			name: "empty slack message",
			src: &rule.Slack{
				Base:    goodBase,
				Channel: "channel1",
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "slack msg template is empty",
			},
		},
		{
			name: "empty pagerDuty message",
			src: &rule.PagerDuty{
				Base: goodBase,
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  "pagerduty invalid message template",
			},
		},
		{
			name: "bad tag rule",
			src: &rule.PagerDuty{
				Base: rule.Base{
					ID:         influxTesting.MustIDBase16(id1),
					OwnerID:    influxTesting.MustIDBase16(id2),
					Name:       "name1",
					OrgID:      influxTesting.MustIDBase16(id3),
					EndpointID: 1,
					TagRules: []notification.TagRule{
						{
							Tag: influxdb.Tag{
								Key:   "k1",
								Value: "v1",
							},
							Operator: notification.Operator("bad"),
						},
					},
				},
				MessageTemplate: "body {var2}",
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  `Operator "bad" is invalid`,
			},
		},
		{
			name: "bad limit",
			src: &rule.PagerDuty{
				Base: rule.Base{
					ID:         influxTesting.MustIDBase16(id1),
					OwnerID:    influxTesting.MustIDBase16(id2),
					OrgID:      influxTesting.MustIDBase16(id3),
					EndpointID: 1,
					Name:       "name1",
					TagRules: []notification.TagRule{
						{
							Tag: influxdb.Tag{
								Key:   "k1",
								Value: "v1",
							},
							Operator: notification.RegexEqual,
						},
					},
					Limit: &influxdb.Limit{
						Rate: 3,
					},
				},
				MessageTemplate: "body {var2}",
			},
			err: &influxdb.Error{
				Code: influxdb.EInvalid,
				Msg:  `if limit is set, limit and limitEvery must be larger than 0`,
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
var time3 = time.Date(2006, time.July, 15, 5, 23, 53, 10, time.UTC)

func TestJSON(t *testing.T) {
	cases := []struct {
		name string
		src  influxdb.NotificationRule
	}{
		{
			name: "simple slack",
			src: &rule.Slack{
				Base: rule.Base{
					ID:          influxTesting.MustIDBase16(id1),
					OwnerID:     influxTesting.MustIDBase16(id2),
					Name:        "name1",
					OrgID:       influxTesting.MustIDBase16(id3),
					RunbookLink: "runbooklink1",
					SleepUntil:  &time3,
					Every:       mustDuration("1h"),
					TagRules: []notification.TagRule{
						{
							Tag: influxdb.Tag{
								Key:   "k1",
								Value: "v1",
							},
							Operator: notification.NotEqual,
						},
						{
							Tag: influxdb.Tag{
								Key:   "k2",
								Value: "v2",
							},
							Operator: notification.RegexEqual,
						},
					},
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				Channel:         "channel1",
				MessageTemplate: "msg1",
			},
		},
		{
			name: "simple smtp",
			src: &rule.PagerDuty{
				Base: rule.Base{
					ID:          influxTesting.MustIDBase16(id1),
					Name:        "name1",
					OwnerID:     influxTesting.MustIDBase16(id2),
					OrgID:       influxTesting.MustIDBase16(id3),
					RunbookLink: "runbooklink1",
					SleepUntil:  &time3,
					Every:       mustDuration("1h"),
					TagRules: []notification.TagRule{
						{
							Tag: influxdb.Tag{
								Key:   "k1",
								Value: "v1",
							},
							Operator: notification.NotEqual,
						},
						{
							Tag: influxdb.Tag{
								Key:   "k2",
								Value: "v2",
							},
							Operator: notification.RegexEqual,
						},
					},
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				MessageTemplate: "msg1",
			},
		},
		{
			name: "simple pagerDuty",
			src: &rule.PagerDuty{
				Base: rule.Base{
					ID:          influxTesting.MustIDBase16(id1),
					Name:        "name1",
					OwnerID:     influxTesting.MustIDBase16(id2),
					OrgID:       influxTesting.MustIDBase16(id3),
					RunbookLink: "runbooklink1",
					SleepUntil:  &time3,
					Every:       mustDuration("1h"),
					TagRules: []notification.TagRule{
						{
							Tag: influxdb.Tag{
								Key:   "k1",
								Value: "v1",
							},
							Operator: notification.NotEqual,
						},
					},
					StatusRules: []notification.StatusRule{
						{
							CurrentLevel:  notification.Warn,
							PreviousLevel: lvlPtr(notification.Critical),
						},
						{
							CurrentLevel: notification.Critical,
						},
					},
					CRUDLog: influxdb.CRUDLog{
						CreatedAt: timeGen1.Now(),
						UpdatedAt: timeGen2.Now(),
					},
				},
				MessageTemplate: "msg1",
			},
		},
	}
	for _, c := range cases {
		b, err := json.Marshal(c.src)
		if err != nil {
			t.Fatalf("%s marshal failed, err: %s", c.name, err.Error())
		}
		got, err := rule.UnmarshalJSON(b)
		if err != nil {
			t.Fatalf("%s unmarshal failed, err: %s", c.name, err.Error())
		}
		if diff := cmp.Diff(got, c.src); diff != "" {
			t.Errorf("failed %s, notification rule are different -got/+want\ndiff %s", c.name, diff)
		}
	}
}
