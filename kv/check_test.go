package kv_test

import (
	"context"
	"testing"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/kv"
	influxdbtesting "github.com/influxdata/influxdb/testing"
)

func TestBoltCheckService(t *testing.T) {
	influxdbtesting.CheckService(initBoltCheckService, t)
}

func TestInmemCheckService(t *testing.T) {
	influxdbtesting.CheckService(initInmemCheckService, t)
}

func initBoltCheckService(f influxdbtesting.CheckFields, t *testing.T) (influxdb.CheckService, string, func()) {
	s, closeBolt, err := NewTestBoltStore()
	if err != nil {
		t.Fatalf("failed to create new kv store: %v", err)
	}

	svc, op, closeSvc := initCheckService(s, f, t)
	return svc, op, func() {
		closeSvc()
		closeBolt()
	}
}

func initInmemCheckService(f influxdbtesting.CheckFields, t *testing.T) (influxdb.CheckService, string, func()) {
	s, closeBolt, err := NewTestInmemStore()
	if err != nil {
		t.Fatalf("failed to create new kv store: %v", err)
	}

	svc, op, closeSvc := initCheckService(s, f, t)
	return svc, op, func() {
		closeSvc()
		closeBolt()
	}
}

func initCheckService(s kv.Store, f influxdbtesting.CheckFields, t *testing.T) (influxdb.CheckService, string, func()) {
	svc := kv.NewService(s)
	svc.IDGenerator = f.IDGenerator
	svc.TimeGenerator = f.TimeGenerator
	if f.TimeGenerator == nil {
		svc.TimeGenerator = influxdb.RealTimeGenerator{}
	}

	ctx := context.Background()
	if err := svc.Initialize(ctx); err != nil {
		t.Fatalf("error initializing check service: %v", err)
	}
	for _, m := range f.UserResourceMappings {
		if err := svc.CreateUserResourceMapping(ctx, m); err != nil {
			t.Fatalf("failed to populate user resource mapping: %v", err)
		}
	}
	for _, o := range f.Organizations {
		if err := svc.PutOrganization(ctx, o); err != nil {
			t.Fatalf("failed to populate organizations")
		}
	}
	for _, c := range f.Checks {
		if err := svc.PutCheck(ctx, c); err != nil {
			t.Fatalf("failed to populate checks")
		}
	}
	return svc, kv.OpPrefix, func() {
		for _, o := range f.Organizations {
			if err := svc.DeleteOrganization(ctx, o.ID); err != nil {
				t.Logf("failed to remove organization: %v", err)
			}
		}
		for _, urm := range f.UserResourceMappings {
			if err := svc.DeleteUserResourceMapping(ctx, urm.ResourceID, urm.UserID); err != nil && influxdb.ErrorCode(err) != influxdb.ENotFound {
				t.Logf("failed to remove urm rule: %v", err)
			}
		}
		for _, c := range f.Checks {
			if err := svc.DeleteCheck(ctx, c.GetID()); err != nil {
				t.Logf("failed to remove check: %v", err)
			}
		}
	}
}
