package testing

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/influxdata/flux/parser"
	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/mock"
	"github.com/influxdata/influxdb/notification"
	"github.com/influxdata/influxdb/notification/check"
)

func mustDuration(d string) *notification.Duration {
	dur, err := parser.ParseDuration(d)
	if err != nil {
		panic(err)
	}

	return (*notification.Duration)(dur)
}

const (
	checkOneID = "020f755c3c082000"
	checkTwoID = "020f755c3c082001"
)

var script = `data = from(bucket: "telegraf") |> range(start: -1m)`

var deadman1 = &check.Deadman{
	Base: check.Base{
		Name:        "name1",
		ID:          MustIDBase16(checkOneID),
		OrgID:       MustIDBase16(orgOneID),
		OwnerID:     MustIDBase16(sixID),
		Description: "desc1",
		TaskID:      1,
		Query: influxdb.DashboardQuery{
			Text: script,
			BuilderConfig: influxdb.BuilderConfig{
				Tags: []struct {
					Key    string   `json:"key"`
					Values []string `json:"values"`
				}{
					{
						Key:    "_field",
						Values: []string{"usage_user"},
					},
				},
			},
		},
		Every:                 mustDuration("1m"),
		StatusMessageTemplate: "msg1",
		Tags: []influxdb.Tag{
			{Key: "k1", Value: "v1"},
			{Key: "k2", Value: "v2"},
		},
		CRUDLog: influxdb.CRUDLog{
			CreatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
			UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
		},
	},
	TimeSince:  mustDuration("21s"),
	StaleTime:  mustDuration("1h"),
	ReportZero: true,
	Level:      notification.Critical,
}

var threshold1 = &check.Threshold{
	Base: check.Base{
		Name:                  "name2",
		ID:                    MustIDBase16(checkTwoID),
		OrgID:                 MustIDBase16(orgTwoID),
		OwnerID:               MustIDBase16(sixID),
		TaskID:                1,
		Description:           "desc2",
		StatusMessageTemplate: "msg2",
		Every:                 mustDuration("1m"),
		Query: influxdb.DashboardQuery{
			Text: script,
			BuilderConfig: influxdb.BuilderConfig{
				Tags: []struct {
					Key    string   `json:"key"`
					Values []string `json:"values"`
				}{
					{
						Key:    "_field",
						Values: []string{"usage_user"},
					},
				},
			},
		},
		Tags: []influxdb.Tag{
			{Key: "k11", Value: "v11"},
		},
		CRUDLog: influxdb.CRUDLog{
			CreatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
			UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
		},
	},
	Thresholds: []check.ThresholdConfig{
		&check.Lesser{
			ThresholdConfigBase: check.ThresholdConfigBase{
				Level: notification.Ok,
			},
			Value: 1000,
		},
		&check.Greater{
			ThresholdConfigBase: check.ThresholdConfigBase{
				Level: notification.Warn,
			},
			Value: 2000,
		},
		&check.Range{
			ThresholdConfigBase: check.ThresholdConfigBase{
				Level: notification.Info,
			},
			Min:    1500,
			Max:    1900,
			Within: true,
		},
	},
}

var checkCmpOptions = cmp.Options{
	cmp.Comparer(func(x, y []byte) bool {
		return bytes.Equal(x, y)
	}),
	cmpopts.IgnoreFields(check.Base{}, "TaskID"),
	cmp.Transformer("Sort", func(in []influxdb.Check) []influxdb.Check {
		out := append([]influxdb.Check(nil), in...) // Copy input to avoid mutating it
		sort.Slice(out, func(i, j int) bool {
			return out[i].GetID() > out[j].GetID()
		})
		return out
	}),
}

// CheckFields will include the IDGenerator, and checks
type CheckFields struct {
	IDGenerator          influxdb.IDGenerator
	TimeGenerator        influxdb.TimeGenerator
	Checks               []influxdb.Check
	Organizations        []*influxdb.Organization
	UserResourceMappings []*influxdb.UserResourceMapping
	Tasks                []influxdb.TaskCreate
}

type checkServiceF func(
	init func(CheckFields, *testing.T) (influxdb.CheckService, string, func()),
	t *testing.T,
)

// CheckService tests all the service functions.
func CheckService(
	init func(CheckFields, *testing.T) (influxdb.CheckService, string, func()),
	t *testing.T,
) {
	tests := []struct {
		name string
		fn   checkServiceF
	}{
		{
			name: "CreateCheck",
			fn:   CreateCheck,
		},
		{
			name: "FindCheckByID",
			fn:   FindCheckByID,
		},
		{
			name: "FindChecks",
			fn:   FindChecks,
		},
		{
			name: "FindCheck",
			fn:   FindCheck,
		},
		{
			name: "PatchCheck",
			fn:   PatchCheck,
		},
		{
			name: "UpdateCheck",
			fn:   UpdateCheck,
		},
		{
			name: "DeleteCheck",
			fn:   DeleteCheck,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(init, t)
		})
	}
}

// CreateCheck testing
func CreateCheck(
	init func(CheckFields, *testing.T) (influxdb.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		userID influxdb.ID
		check  influxdb.Check
	}
	type wants struct {
		err    error
		checks []influxdb.Check
		urms   []*influxdb.UserResourceMapping
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "create checks with empty set",
			fields: CheckFields{
				IDGenerator:   mock.NewIDGenerator(checkOneID, t),
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Checks:        []influxdb.Check{},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				UserResourceMappings: []*influxdb.UserResourceMapping{
					{
						ResourceID:   MustIDBase16(orgOneID),
						ResourceType: influxdb.OrgsResourceType,
						UserID:       MustIDBase16(fiveID),
						UserType:     influxdb.Owner,
					},
					{
						ResourceID:   MustIDBase16(orgOneID),
						ResourceType: influxdb.OrgsResourceType,
						UserID:       MustIDBase16(sixID),
						UserType:     influxdb.Owner,
					},
				},
			},
			args: args{
				userID: MustIDBase16(twoID),
				check: &check.Deadman{
					Base: check.Base{
						Name:        "name1",
						OrgID:       MustIDBase16(orgOneID),
						Description: "desc1",
						Query: influxdb.DashboardQuery{
							Text: script,
							BuilderConfig: influxdb.BuilderConfig{
								Tags: []struct {
									Key    string   `json:"key"`
									Values []string `json:"values"`
								}{
									{
										Key:    "_field",
										Values: []string{"usage_user"},
									},
								},
							},
						},
						Every:                 mustDuration("1m"),
						StatusMessageTemplate: "msg1",
						Tags: []influxdb.Tag{
							{Key: "k1", Value: "v1"},
							{Key: "k2", Value: "v2"},
						},
					},
					TimeSince:  mustDuration("21s"),
					StaleTime:  mustDuration("1h"),
					ReportZero: true,
					Level:      notification.Critical,
				},
			},
			wants: wants{
				urms: []*influxdb.UserResourceMapping{
					{
						ResourceID:   MustIDBase16(checkOneID),
						ResourceType: influxdb.ChecksResourceType,
						UserID:       MustIDBase16(fiveID),
						UserType:     influxdb.Owner,
					},
					{
						ResourceID:   MustIDBase16(checkOneID),
						ResourceType: influxdb.ChecksResourceType,
						UserID:       MustIDBase16(sixID),
						UserType:     influxdb.Owner,
					},
				},
				checks: []influxdb.Check{
					&check.Deadman{
						Base: check.Base{
							Name:    "name1",
							ID:      MustIDBase16(checkOneID),
							OrgID:   MustIDBase16(orgOneID),
							OwnerID: MustIDBase16(twoID),
							Query: influxdb.DashboardQuery{
								Text: script,
								BuilderConfig: influxdb.BuilderConfig{
									Tags: []struct {
										Key    string   `json:"key"`
										Values []string `json:"values"`
									}{
										{
											Key:    "_field",
											Values: []string{"usage_user"},
										},
									},
								},
							},
							Every:                 mustDuration("1m"),
							Description:           "desc1",
							StatusMessageTemplate: "msg1",
							Tags: []influxdb.Tag{
								{Key: "k1", Value: "v1"},
								{Key: "k2", Value: "v2"},
							},
							CRUDLog: influxdb.CRUDLog{
								CreatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
								UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
							},
						},
						TimeSince:  mustDuration("21s"),
						StaleTime:  mustDuration("1h"),
						ReportZero: true,
						Level:      notification.Critical,
					},
				},
			},
		},
		{
			name: "basic create check",
			fields: CheckFields{
				IDGenerator: &mock.IDGenerator{
					IDFn: func() influxdb.ID {
						return MustIDBase16(checkTwoID)
					},
				},
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Checks: []influxdb.Check{
					deadman1,
				},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
			},
			args: args{
				userID: MustIDBase16(sixID),
				check: &check.Threshold{
					Base: check.Base{
						Name:                  "name2",
						OrgID:                 MustIDBase16(orgTwoID),
						OwnerID:               MustIDBase16(twoID),
						Description:           "desc2",
						StatusMessageTemplate: "msg2",
						Every:                 mustDuration("1m"),
						Query: influxdb.DashboardQuery{
							Text: script,
							BuilderConfig: influxdb.BuilderConfig{
								Tags: []struct {
									Key    string   `json:"key"`
									Values []string `json:"values"`
								}{
									{
										Key:    "_field",
										Values: []string{"usage_user"},
									},
								},
							},
						},
						Tags: []influxdb.Tag{
							{Key: "k11", Value: "v11"},
						},
					},
					Thresholds: []check.ThresholdConfig{
						&check.Lesser{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Ok,
							},
							Value: 1000,
						},
						&check.Greater{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Warn,
							},
							Value: 2000,
						},
						&check.Range{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Info,
							},
							Min:    1500,
							Max:    1900,
							Within: true,
						},
					},
				},
			},
			wants: wants{
				checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
		},
		{
			name: "names should be unique within an organization",
			fields: CheckFields{
				IDGenerator: &mock.IDGenerator{
					IDFn: func() influxdb.ID {
						return MustIDBase16(checkTwoID)
					},
				},
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Checks: []influxdb.Check{
					deadman1,
				},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
			},
			args: args{
				userID: MustIDBase16(twoID),
				check: &check.Threshold{
					Base: check.Base{
						Name:        "name1",
						OrgID:       MustIDBase16(orgOneID),
						Description: "desc1",
						Every:       mustDuration("1m"),
						Query: influxdb.DashboardQuery{
							Text: script,
							BuilderConfig: influxdb.BuilderConfig{
								Tags: []struct {
									Key    string   `json:"key"`
									Values []string `json:"values"`
								}{
									{
										Key:    "_field",
										Values: []string{"usage_user"},
									},
								},
							},
						},
						StatusMessageTemplate: "msg1",
						Tags: []influxdb.Tag{
							{Key: "k1", Value: "v1"},
							{Key: "k2", Value: "v2"},
						},
					},
				},
			},
			wants: wants{
				checks: []influxdb.Check{
					deadman1,
				},
				err: &influxdb.Error{
					Code: influxdb.EConflict,
					Op:   influxdb.OpCreateCheck,
					Msg:  fmt.Sprintf("check with name name1 already exists"),
				},
			},
		},
		{
			name: "names should not be unique across organizations",
			fields: CheckFields{
				IDGenerator: &mock.IDGenerator{
					IDFn: func() influxdb.ID {
						return MustIDBase16(checkTwoID)
					},
				},
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
				},
			},
			args: args{
				userID: MustIDBase16(twoID),
				check: &check.Threshold{
					Base: check.Base{
						Name:        "name1",
						OrgID:       MustIDBase16(orgTwoID),
						Description: "desc2",
						Every:       mustDuration("1m"),
						Query: influxdb.DashboardQuery{
							Text: script,
							BuilderConfig: influxdb.BuilderConfig{
								Tags: []struct {
									Key    string   `json:"key"`
									Values []string `json:"values"`
								}{
									{
										Key:    "_field",
										Values: []string{"usage_user"},
									},
								},
							},
						},
						StatusMessageTemplate: "msg2",
						Tags: []influxdb.Tag{
							{Key: "k11", Value: "v11"},
							{Key: "k22", Value: "v22"},
						},
					},
				},
			},
			wants: wants{
				checks: []influxdb.Check{
					deadman1,
					&check.Threshold{
						Base: check.Base{
							ID:    MustIDBase16(checkTwoID),
							Name:  "name1",
							OrgID: MustIDBase16(orgTwoID),
							Every: mustDuration("1m"),
							Query: influxdb.DashboardQuery{
								Text: script,
								BuilderConfig: influxdb.BuilderConfig{
									Tags: []struct {
										Key    string   `json:"key"`
										Values []string `json:"values"`
									}{
										{
											Key:    "_field",
											Values: []string{"usage_user"},
										},
									},
								},
							},
							OwnerID:               MustIDBase16(twoID),
							Description:           "desc2",
							StatusMessageTemplate: "msg2",
							Tags: []influxdb.Tag{
								{Key: "k11", Value: "v11"},
								{Key: "k22", Value: "v22"},
							},
							CRUDLog: influxdb.CRUDLog{
								CreatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
								UpdatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
							},
						},
					},
				},
			},
		},
		{
			name: "create check with orgID not exist",
			fields: CheckFields{
				IDGenerator:   mock.NewIDGenerator(checkOneID, t),
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Checks:        []influxdb.Check{},
				Organizations: []*influxdb.Organization{},
			},
			args: args{
				userID: MustIDBase16(twoID),
				check: &check.Threshold{
					Base: check.Base{
						Name:  "name1",
						OrgID: MustIDBase16(orgOneID),
						Every: mustDuration("1m"),
						Query: influxdb.DashboardQuery{
							Text: script,
							BuilderConfig: influxdb.BuilderConfig{
								Tags: []struct {
									Key    string   `json:"key"`
									Values []string `json:"values"`
								}{
									{
										Key:    "_field",
										Values: []string{"usage_user"},
									},
								},
							},
						},
						Description:           "desc2",
						StatusMessageTemplate: "msg2",
						Tags: []influxdb.Tag{
							{Key: "k11", Value: "v11"},
							{Key: "k22", Value: "v22"},
						},
					},
				},
			},
			wants: wants{
				checks: []influxdb.Check{},
				err: &influxdb.Error{
					Code: influxdb.ENotFound,
					Msg:  "organization not found",
					Op:   influxdb.OpCreateCheck,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()
			createCheck := influxdb.CheckCreate{Check: tt.args.check, Status: influxdb.Active}
			err := s.CreateCheck(ctx, createCheck, tt.args.userID)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			defer s.DeleteCheck(ctx, tt.args.check.GetID())

			checks, _, err := s.FindChecks(ctx, influxdb.CheckFilter{})
			if err != nil {
				t.Fatalf("failed to retrieve checks: %v", err)
			}
			if diff := cmp.Diff(checks, tt.wants.checks, checkCmpOptions...); diff != "" {
				t.Errorf("checks are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// FindCheckByID testing
func FindCheckByID(
	init func(CheckFields, *testing.T) (influxdb.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		id influxdb.ID
	}
	type wants struct {
		err   error
		check influxdb.Check
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "basic find check by id",
			fields: CheckFields{
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
			},
			args: args{
				id: MustIDBase16(checkTwoID),
			},
			wants: wants{
				check: threshold1,
			},
		},
		{
			name: "find check by id not exist",
			fields: CheckFields{
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
			},
			args: args{
				id: MustIDBase16(threeID),
			},
			wants: wants{
				err: &influxdb.Error{
					Code: influxdb.ENotFound,
					Op:   influxdb.OpFindCheckByID,
					Msg:  "check not found",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()

			check, err := s.FindCheckByID(ctx, tt.args.id)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			if diff := cmp.Diff(check, tt.wants.check, checkCmpOptions...); diff != "" {
				t.Errorf("check is different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// FindChecks testing
func FindChecks(
	init func(CheckFields, *testing.T) (influxdb.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		ID           influxdb.ID
		name         string
		organization string
		OrgID        influxdb.ID
		findOptions  influxdb.FindOptions
	}

	type wants struct {
		checks []influxdb.Check
		err    error
	}
	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "find all checks",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
			args: args{},
			wants: wants{
				checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
		},
		{
			name: "find all checks by offset and limit",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
			args: args{
				findOptions: influxdb.FindOptions{
					Offset: 1,
					Limit:  1,
				},
			},
			wants: wants{
				checks: []influxdb.Check{
					threshold1,
				},
			},
		},
		{
			name: "find all checks by descending",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
			args: args{
				findOptions: influxdb.FindOptions{
					Limit:      1,
					Descending: true,
				},
			},
			wants: wants{
				checks: []influxdb.Check{
					threshold1,
				},
			},
		},
		{
			name: "find checks by organization name",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
			args: args{
				organization: "theorg",
			},
			wants: wants{
				checks: []influxdb.Check{
					deadman1,
				},
			},
		},
		{
			name: "find checks by organization id",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "otherorg",
						ID:   MustIDBase16(orgTwoID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
			args: args{
				OrgID: MustIDBase16(orgTwoID),
			},
			wants: wants{
				checks: []influxdb.Check{
					threshold1,
				},
			},
		},
		{
			name: "find check by name",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
			args: args{
				name: "name2",
			},
			wants: wants{
				checks: []influxdb.Check{
					threshold1,
				},
			},
		},
		{
			name: "missing check returns no checks",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []influxdb.Check{},
			},
			args: args{
				name: "xyz",
			},
			wants: wants{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()

			filter := influxdb.CheckFilter{}
			if tt.args.ID.Valid() {
				filter.ID = &tt.args.ID
			}
			if tt.args.OrgID.Valid() {
				filter.OrgID = &tt.args.OrgID
			}
			if tt.args.organization != "" {
				filter.Org = &tt.args.organization
			}
			if tt.args.name != "" {
				filter.Name = &tt.args.name
			}

			checks, _, err := s.FindChecks(ctx, filter, tt.args.findOptions)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			if diff := cmp.Diff(checks, tt.wants.checks, checkCmpOptions...); diff != "" {
				t.Errorf("checks are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// DeleteCheck testing
func DeleteCheck(
	init func(CheckFields, *testing.T) (influxdb.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		ID string
	}
	type wants struct {
		err    error
		checks []influxdb.Check
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "delete checks using exist id",
			fields: CheckFields{
				IDGenerator: mock.NewIDGenerator("0000000000000001", t),
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Tasks: []influxdb.TaskCreate{
					{
						Flux: `option task = { every: 10s, name: "foo" }
data = from(bucket: "telegraf") |> range(start: -1m)`,
						OrganizationID: MustIDBase16(orgOneID),
						OwnerID:        MustIDBase16(sixID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
			args: args{
				ID: checkOneID,
			},
			wants: wants{
				checks: []influxdb.Check{
					threshold1,
				},
			},
		},
		{
			name: "delete checks using id that does not exist",
			fields: CheckFields{
				IDGenerator: mock.NewIDGenerator("0000000000000001", t),
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Tasks: []influxdb.TaskCreate{
					{
						Flux: `option task = { every: 10s, name: "foo" }
data = from(bucket: "telegraf") |> range(start: -1m)`,
						OrganizationID: MustIDBase16(orgOneID),
						OwnerID:        MustIDBase16(sixID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
			args: args{
				ID: "1234567890654321",
			},
			wants: wants{
				err: &influxdb.Error{
					Op:   influxdb.OpDeleteCheck,
					Msg:  "check not found",
					Code: influxdb.ENotFound,
				},
				checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()
			err := s.DeleteCheck(ctx, MustIDBase16(tt.args.ID))
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			filter := influxdb.CheckFilter{}
			checks, _, err := s.FindChecks(ctx, filter)
			if err != nil {
				t.Fatalf("failed to retrieve checks: %v", err)
			}
			if diff := cmp.Diff(checks, tt.wants.checks, checkCmpOptions...); diff != "" {
				t.Errorf("checks are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// FindCheck testing
func FindCheck(
	init func(CheckFields, *testing.T) (influxdb.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		name  string
		OrgID influxdb.ID
	}

	type wants struct {
		check influxdb.Check
		err   error
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "find check by name",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
					{
						Name: "theorg2",
						ID:   MustIDBase16(orgTwoID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					threshold1,
				},
			},
			args: args{
				name:  "name1",
				OrgID: MustIDBase16(orgOneID),
			},
			wants: wants{
				check: deadman1,
			},
		},
		{
			name: "mixed filter",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []influxdb.Check{},
			},
			args: args{
				name:  "name2",
				OrgID: MustIDBase16(orgOneID),
			},
			wants: wants{
				err: &influxdb.Error{
					Code: influxdb.ENotFound,
					Op:   influxdb.OpFindCheck,
					Msg:  "check not found",
				},
			},
		},
		{
			name: "missing check returns error",
			fields: CheckFields{
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []influxdb.Check{},
			},
			args: args{
				name:  "xyz",
				OrgID: MustIDBase16(orgOneID),
			},
			wants: wants{
				err: &influxdb.Error{
					Code: influxdb.ENotFound,
					Op:   influxdb.OpFindCheck,
					Msg:  "check not found",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()
			filter := influxdb.CheckFilter{}
			if tt.args.name != "" {
				filter.Name = &tt.args.name
			}
			if tt.args.OrgID.Valid() {
				filter.OrgID = &tt.args.OrgID
			}

			check, err := s.FindCheck(ctx, filter)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			if diff := cmp.Diff(check, tt.wants.check, checkCmpOptions...); diff != "" {
				t.Errorf("checks are different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// UpdateCheck testing
func UpdateCheck(
	init func(CheckFields, *testing.T) (influxdb.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		id    influxdb.ID
		check influxdb.Check
	}
	type wants struct {
		err   error
		check influxdb.Check
	}

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "mixed update",
			fields: CheckFields{
				IDGenerator:   mock.NewIDGenerator("0000000000000001", t),
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2007, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Tasks: []influxdb.TaskCreate{
					{
						Flux: `option task = { every: 10s, name: "foo" }
data = from(bucket: "telegraf") |> range(start: -1m)`,
						OrganizationID: MustIDBase16(orgOneID),
						OwnerID:        MustIDBase16(sixID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
				},
			},
			args: args{
				id: MustIDBase16(checkOneID),
				check: &check.Threshold{
					Base: check.Base{
						ID:      MustIDBase16(checkTwoID),
						OrgID:   MustIDBase16(orgOneID),
						OwnerID: MustIDBase16(twoID),
						Every:   mustDuration("1m"),
						Query: influxdb.DashboardQuery{
							Text: script,
							BuilderConfig: influxdb.BuilderConfig{
								Tags: []struct {
									Key    string   `json:"key"`
									Values []string `json:"values"`
								}{
									{
										Key:    "_field",
										Values: []string{"usage_user"},
									},
								},
							},
						},
						Name:                  "changed",
						Description:           "desc changed",
						StatusMessageTemplate: "msg2",
						TaskID:                1,
						Tags: []influxdb.Tag{
							{Key: "k11", Value: "v11"},
							{Key: "k22", Value: "v22"},
							{Key: "k33", Value: "v33"},
						},
						CRUDLog: influxdb.CRUDLog{
							CreatedAt: time.Date(2001, 5, 4, 1, 2, 3, 0, time.UTC),
							UpdatedAt: time.Date(2002, 5, 4, 1, 2, 3, 0, time.UTC),
						},
					},
					Thresholds: []check.ThresholdConfig{
						&check.Lesser{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Ok,
							},
							Value: 1000,
						},
						&check.Greater{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Warn,
							},
							Value: 2000,
						},
						&check.Range{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Info,
							},
							Min:    1500,
							Max:    1900,
							Within: true,
						},
					},
				},
			},
			wants: wants{
				check: &check.Threshold{
					Base: check.Base{
						ID:      MustIDBase16(checkOneID),
						OrgID:   MustIDBase16(orgOneID),
						Name:    "changed",
						Every:   mustDuration("1m"),
						OwnerID: MustIDBase16(sixID),
						Query: influxdb.DashboardQuery{
							Text: script,
							BuilderConfig: influxdb.BuilderConfig{
								Tags: []struct {
									Key    string   `json:"key"`
									Values []string `json:"values"`
								}{
									{
										Key:    "_field",
										Values: []string{"usage_user"},
									},
								},
							},
						},
						Description:           "desc changed",
						StatusMessageTemplate: "msg2",
						Tags: []influxdb.Tag{
							{Key: "k11", Value: "v11"},
							{Key: "k22", Value: "v22"},
							{Key: "k33", Value: "v33"},
						},
						TaskID: 1,
						CRUDLog: influxdb.CRUDLog{
							CreatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
							UpdatedAt: time.Date(2007, 5, 4, 1, 2, 3, 0, time.UTC),
						},
					},
					Thresholds: []check.ThresholdConfig{
						&check.Lesser{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Ok,
							},
							Value: 1000,
						},
						&check.Greater{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Warn,
							},
							Value: 2000,
						},
						&check.Range{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Info,
							},
							Min:    1500,
							Max:    1900,
							Within: true,
						},
					},
				},
			},
		},
		{
			name: "update name unique",
			fields: CheckFields{
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					&check.Deadman{
						Base: check.Base{
							ID:    MustIDBase16(checkTwoID),
							OrgID: MustIDBase16(orgOneID),
							Every: mustDuration("1m"),
							Query: influxdb.DashboardQuery{
								Text: script,
								BuilderConfig: influxdb.BuilderConfig{
									Tags: []struct {
										Key    string   `json:"key"`
										Values []string `json:"values"`
									}{
										{
											Key:    "_field",
											Values: []string{"usage_user"},
										},
									},
								},
							},
							TaskID:                1,
							Name:                  "check2",
							OwnerID:               MustIDBase16(twoID),
							StatusMessageTemplate: "msg1",
						},
					},
				},
			},
			args: args{
				id: MustIDBase16(checkOneID),
				check: &check.Deadman{
					Base: check.Base{
						OrgID:                 MustIDBase16(orgOneID),
						OwnerID:               MustIDBase16(twoID),
						Name:                  "check2",
						Description:           "desc changed",
						TaskID:                1,
						Every:                 mustDuration("1m"),
						StatusMessageTemplate: "msg2",
						Tags: []influxdb.Tag{
							{Key: "k11", Value: "v11"},
							{Key: "k22", Value: "v22"},
							{Key: "k33", Value: "v33"},
						},
						CRUDLog: influxdb.CRUDLog{
							CreatedAt: time.Date(2001, 5, 4, 1, 2, 3, 0, time.UTC),
							UpdatedAt: time.Date(2002, 5, 4, 1, 2, 3, 0, time.UTC),
						},
					},
					TimeSince:  mustDuration("12s"),
					StaleTime:  mustDuration("1h"),
					ReportZero: false,
					Level:      notification.Warn,
				},
			},
			wants: wants{
				err: &influxdb.Error{
					Code: influxdb.EConflict,
					Msg:  "check name is not unique",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()

			checkCreate := influxdb.CheckCreate{Check: tt.args.check, Status: influxdb.Active}

			check, err := s.UpdateCheck(ctx, tt.args.id, checkCreate)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			if diff := cmp.Diff(check, tt.wants.check, checkCmpOptions...); diff != "" {
				t.Errorf("check is different -got/+want\ndiff %s", diff)
			}
		})
	}
}

// PatchCheck testing
func PatchCheck(
	init func(CheckFields, *testing.T) (influxdb.CheckService, string, func()),
	t *testing.T,
) {
	type args struct {
		id  influxdb.ID
		upd influxdb.CheckUpdate
	}
	type wants struct {
		err   error
		check influxdb.Check
	}

	inactive := influxdb.Inactive

	tests := []struct {
		name   string
		fields CheckFields
		args   args
		wants  wants
	}{
		{
			name: "mixed patch",
			fields: CheckFields{
				IDGenerator:   mock.NewIDGenerator("0000000000000001", t),
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2007, 5, 4, 1, 2, 3, 0, time.UTC)},
				Tasks: []influxdb.TaskCreate{
					{
						Flux: `option task = { every: 10s, name: "foo" }
data = from(bucket: "telegraf") |> range(start: -1m)`,
						OrganizationID: MustIDBase16(orgOneID),
						OwnerID:        MustIDBase16(sixID),
					},
				},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
				},
			},
			args: args{
				id: MustIDBase16(checkOneID),
				upd: influxdb.CheckUpdate{
					Name:        strPtr("changed"),
					Description: strPtr("desc changed"),
					Status:      &inactive,
				},
			},
			wants: wants{
				check: &check.Deadman{
					Base: check.Base{
						ID:          MustIDBase16(checkOneID),
						OrgID:       MustIDBase16(orgOneID),
						Name:        "changed",
						OwnerID:     MustIDBase16(sixID),
						Every:       mustDuration("1m"),
						Description: "desc changed",
						Query: influxdb.DashboardQuery{
							Text: script,
							BuilderConfig: influxdb.BuilderConfig{
								Tags: []struct {
									Key    string   `json:"key"`
									Values []string `json:"values"`
								}{
									{
										Key:    "_field",
										Values: []string{"usage_user"},
									},
								},
							},
						},
						StatusMessageTemplate: "msg1",
						Tags: []influxdb.Tag{
							{Key: "k1", Value: "v1"},
							{Key: "k2", Value: "v2"},
						},
						CRUDLog: influxdb.CRUDLog{
							CreatedAt: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC),
							UpdatedAt: time.Date(2007, 5, 4, 1, 2, 3, 0, time.UTC),
						},
					},
					TimeSince:  mustDuration("21s"),
					StaleTime:  mustDuration("1h"),
					ReportZero: true,
					Level:      notification.Critical,
				},
			},
		},
		{
			name: "update name unique",
			fields: CheckFields{
				TimeGenerator: mock.TimeGenerator{FakeValue: time.Date(2006, 5, 4, 1, 2, 3, 0, time.UTC)},
				Organizations: []*influxdb.Organization{
					{
						Name: "theorg",
						ID:   MustIDBase16(orgOneID),
					},
				},
				Checks: []influxdb.Check{
					deadman1,
					&check.Deadman{
						Base: check.Base{
							ID:      MustIDBase16(checkTwoID),
							OrgID:   MustIDBase16(orgOneID),
							Every:   mustDuration("1m"),
							Name:    "check2",
							OwnerID: MustIDBase16(sixID),
							Query: influxdb.DashboardQuery{
								Text: script,
								BuilderConfig: influxdb.BuilderConfig{
									Tags: []struct {
										Key    string   `json:"key"`
										Values []string `json:"values"`
									}{
										{
											Key:    "_field",
											Values: []string{"usage_user"},
										},
									},
								},
							},
							StatusMessageTemplate: "msg1",
						},
					},
				},
			},
			args: args{
				id: MustIDBase16(checkOneID),
				upd: influxdb.CheckUpdate{
					Name: strPtr("check2"),
				},
			},
			wants: wants{
				err: &influxdb.Error{
					Code: influxdb.EConflict,
					Msg:  "check name is not unique",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, opPrefix, done := init(tt.fields, t)
			defer done()
			ctx := context.Background()

			check, err := s.PatchCheck(ctx, tt.args.id, tt.args.upd)
			diffPlatformErrors(tt.name, err, tt.wants.err, opPrefix, t)

			if diff := cmp.Diff(check, tt.wants.check, checkCmpOptions...); diff != "" {
				t.Errorf("check is different -got/+want\ndiff %s", diff)
			}
		})
	}
}
