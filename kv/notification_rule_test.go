package kv_test

import (
	"context"
	"testing"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/kv"
	influxdbtesting "github.com/influxdata/influxdb/testing"
	"go.uber.org/zap/zaptest"
)

func TestBoltNotificationRuleStore(t *testing.T) {
	influxdbtesting.NotificationRuleStore(initBoltNotificationRuleStore, t)
}

func initBoltNotificationRuleStore(f influxdbtesting.NotificationRuleFields, t *testing.T) (influxdb.NotificationRuleStore, func()) {
	s, closeBolt, err := NewTestBoltStore(t)
	if err != nil {
		t.Fatalf("failed to create new kv store: %v", err)
	}

	svc, closeSvc := initNotificationRuleStore(s, f, t)
	return svc, func() {
		closeSvc()
		closeBolt()
	}
}

func initNotificationRuleStore(s kv.Store, f influxdbtesting.NotificationRuleFields, t *testing.T) (influxdb.NotificationRuleStore, func()) {
	svc := kv.NewService(zaptest.NewLogger(t), s)
	svc.IDGenerator = f.IDGenerator
	svc.TimeGenerator = f.TimeGenerator
	if f.TimeGenerator == nil {
		svc.TimeGenerator = influxdb.RealTimeGenerator{}
	}

	ctx := context.Background()
	if err := svc.Initialize(ctx); err != nil {
		t.Fatalf("error initializing user service: %v", err)
	}

	for _, o := range f.Orgs {
		if err := svc.PutOrganization(ctx, o); err != nil {
			t.Fatalf("failed to populate org: %v", err)
		}
	}

	for _, m := range f.UserResourceMappings {
		if err := svc.CreateUserResourceMapping(ctx, m); err != nil {
			t.Fatalf("failed to populate user resource mapping: %v", err)
		}
	}

	for _, e := range f.Endpoints {
		if err := svc.CreateNotificationEndpoint(ctx, e, 1); err != nil {
			t.Fatalf("failed to populate notification endpoint: %v", err)
		}
	}

	for _, nr := range f.NotificationRules {
		nrc := influxdb.NotificationRuleCreate{
			NotificationRule: nr,
			Status:           influxdb.Active,
		}
		if err := svc.PutNotificationRule(ctx, nrc); err != nil {
			t.Fatalf("failed to populate notification rule: %v", err)
		}
	}

	for _, c := range f.Tasks {
		if _, err := svc.CreateTask(ctx, c); err != nil {
			t.Fatalf("failed to populate task: %v", err)
		}
	}

	return svc, func() {
		for _, nr := range f.NotificationRules {
			if err := svc.DeleteNotificationRule(ctx, nr.GetID()); err != nil {
				t.Logf("failed to remove notification rule: %v", err)
			}
		}
		for _, urm := range f.UserResourceMappings {
			if err := svc.DeleteUserResourceMapping(ctx, urm.ResourceID, urm.UserID); err != nil && influxdb.ErrorCode(err) != influxdb.ENotFound {
				t.Logf("failed to remove urm rule: %v", err)
			}
		}
		for _, o := range f.Orgs {
			if err := svc.DeleteOrganization(ctx, o.ID); err != nil {
				t.Fatalf("failed to remove org: %v", err)
			}
		}
	}
}
