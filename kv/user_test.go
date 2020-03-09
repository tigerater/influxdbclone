package kv_test

import (
	"context"
	"testing"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/kv"
	influxdbtesting "github.com/influxdata/influxdb/testing"
)

func TestBoltUserService(t *testing.T) {
	influxdbtesting.UserService(initBoltUserService, t)
}

func TestInmemUserService(t *testing.T) {
	influxdbtesting.UserService(initInmemUserService, t)
}

func initBoltUserService(f influxdbtesting.UserFields, t *testing.T) (influxdb.UserService, string, func()) {
	s, closeBolt, err := NewTestBoltStore()
	if err != nil {
		t.Fatalf("failed to create new kv store: %v", err)
	}

	svc, op, closeSvc := initUserService(s, f, t)
	return svc, op, func() {
		closeSvc()
		closeBolt()
	}
}

func initInmemUserService(f influxdbtesting.UserFields, t *testing.T) (influxdb.UserService, string, func()) {
	s, closeBolt, err := NewTestInmemStore()
	if err != nil {
		t.Fatalf("failed to create new kv store: %v", err)
	}

	svc, op, closeSvc := initUserService(s, f, t)
	return svc, op, func() {
		closeSvc()
		closeBolt()
	}
}

func initUserService(s kv.Store, f influxdbtesting.UserFields, t *testing.T) (influxdb.UserService, string, func()) {
	svc := kv.NewService(s)
	svc.IDGenerator = f.IDGenerator

	ctx := context.Background()
	if err := svc.Initialize(ctx); err != nil {
		t.Fatalf("error initializing user service: %v", err)
	}

	for _, u := range f.Users {
		if err := svc.PutUser(ctx, u); err != nil {
			t.Fatalf("failed to populate users")
		}
	}

	return svc, kv.OpPrefix, func() {
		for _, u := range f.Users {
			if err := svc.DeleteUser(ctx, u.ID); err != nil {
				t.Logf("failed to remove users: %v", err)
			}
		}
	}
}
