package bolt_test

import (
	"context"
	"testing"

	"github.com/influxdata/influxdb/kv"
	platformtesting "github.com/influxdata/influxdb/testing"
)

func initKVStore(f platformtesting.KVStoreFields, t *testing.T) (kv.Store, func()) {
	s, closeFn, err := NewTestKVStore()
	if err != nil {
		t.Fatalf("failed to create new kv store: %v", err)
	}

	err = s.Update(context.Background(), func(tx kv.Tx) error {
		b, err := tx.Bucket(f.Bucket)
		if err != nil {
			return err
		}

		for _, p := range f.Pairs {
			if err := b.Put(p.Key, p.Value); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed to put keys: %v", err)
	}
	return s, func() {
		closeFn()
	}
}

func TestKVStore(t *testing.T) {
	platformtesting.KVStore(initKVStore, t)
}
