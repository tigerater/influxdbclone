package kv_test

import (
	"context"
	"testing"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/kv"
	influxdbtesting "github.com/influxdata/influxdb/testing"
)

func TestBoltUserResourceMappingService(t *testing.T) {
	influxdbtesting.UserResourceMappingService(initBoltUserResourceMappingService, t)
}

func TestInmemUserResourceMappingService(t *testing.T) {
	influxdbtesting.UserResourceMappingService(initInmemUserResourceMappingService, t)
}

func initBoltUserResourceMappingService(f influxdbtesting.UserResourceFields, t *testing.T) (influxdb.UserResourceMappingService, func()) {
	s, closeBolt, err := NewTestBoltStore()
	if err != nil {
		t.Fatalf("failed to create new kv store: %v", err)
	}

	svc, closeSvc := initUserResourceMappingService(s, f, t)
	return svc, func() {
		closeSvc()
		closeBolt()
	}
}

func initInmemUserResourceMappingService(f influxdbtesting.UserResourceFields, t *testing.T) (influxdb.UserResourceMappingService, func()) {
	s, closeBolt, err := NewTestInmemStore()
	if err != nil {
		t.Fatalf("failed to create new kv store: %v", err)
	}

	svc, closeSvc := initUserResourceMappingService(s, f, t)
	return svc, func() {
		closeSvc()
		closeBolt()
	}
}

func initUserResourceMappingService(s kv.Store, f influxdbtesting.UserResourceFields, t *testing.T) (influxdb.UserResourceMappingService, func()) {
	svc := kv.NewService(s)

	ctx := context.Background()
	if err := svc.Initialize(ctx); err != nil {
		t.Fatalf("error initializing urm service: %v", err)
	}

	for _, m := range f.UserResourceMappings {
		if err := svc.CreateUserResourceMapping(ctx, m); err != nil {
			t.Fatalf("failed to populate mappings")
		}
	}

	return svc, func() {
		for _, m := range f.UserResourceMappings {
			if err := svc.DeleteUserResourceMapping(ctx, m.ResourceID, m.UserID); err != nil {
				t.Logf("failed to remove user resource mapping: %v", err)
			}
		}
	}
}
