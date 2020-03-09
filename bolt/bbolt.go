package bolt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "github.com/coreos/bbolt"
	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/rand"
	"github.com/influxdata/influxdb/snowflake"
	"go.uber.org/zap"
)

// OpPrefix is the prefix for bolt ops
const OpPrefix = "bolt/"

func getOp(op string) string {
	return OpPrefix + op
}

// Client is a client for the boltDB data store.
type Client struct {
	Path string
	db   *bolt.DB
	log  *zap.Logger

	IDGenerator    platform.IDGenerator
	TokenGenerator platform.TokenGenerator
	platform.TimeGenerator
}

// NewClient returns an instance of a Client.
func NewClient(log *zap.Logger) *Client {
	return &Client{
		log:            log,
		IDGenerator:    snowflake.NewIDGenerator(),
		TokenGenerator: rand.NewTokenGenerator(64),
		TimeGenerator:  platform.RealTimeGenerator{},
	}
}

// DB returns the clients DB.
func (c *Client) DB() *bolt.DB {
	return c.db
}

// Open / create boltDB file.
func (c *Client) Open(ctx context.Context) error {
	// Ensure the required directory structure exists.
	if err := os.MkdirAll(filepath.Dir(c.Path), 0700); err != nil {
		return fmt.Errorf("unable to create directory %s: %v", c.Path, err)
	}

	if _, err := os.Stat(c.Path); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Open database file.
	db, err := bolt.Open(c.Path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return fmt.Errorf("unable to open boltdb; is there a chronograf already running?  %v", err)
	}
	c.db = db

	if err := c.initialize(ctx); err != nil {
		return err
	}

	c.log.Info("Resources opened", zap.String("path", c.Path))
	return nil
}

// initialize creates Buckets that are missing
func (c *Client) initialize(ctx context.Context) error {
	if err := c.db.Update(func(tx *bolt.Tx) error {
		// Always create ID bucket.
		if err := c.initializeID(tx); err != nil {
			return err
		}

		// Always create Buckets bucket.
		if err := c.initializeBuckets(ctx, tx); err != nil {
			return err
		}

		// Always create Organizations bucket.
		if err := c.initializeOrganizations(ctx, tx); err != nil {
			return err
		}

		// Always create Dashboards bucket.
		if err := c.initializeDashboards(ctx, tx); err != nil {
			return err
		}

		// Always create User bucket.
		if err := c.initializeUsers(ctx, tx); err != nil {
			return err
		}

		// Always create Authorization bucket.
		if err := c.initializeAuthorizations(ctx, tx); err != nil {
			return err
		}

		// Always create Onboarding bucket.
		if err := c.initializeOnboarding(ctx, tx); err != nil {
			return err
		}

		// Always create Telegraf Config bucket.
		if err := c.initializeTelegraf(ctx, tx); err != nil {
			return err
		}

		// Always create Source bucket.
		if err := c.initializeSources(ctx, tx); err != nil {
			return err
		}

		// Always create Variables bucket.
		if err := c.initializeVariables(ctx, tx); err != nil {
			return err
		}

		// Always create Scraper bucket.
		if err := c.initializeScraperTargets(ctx, tx); err != nil {
			return err
		}

		// Always create UserResourceMapping bucket.
		if err := c.initializeUserResourceMappings(ctx, tx); err != nil {
			return err
		}

		// Always create labels bucket.
		if err := c.initializeLabels(ctx, tx); err != nil {
			return err
		}

		// Always create Session bucket.
		if err := c.initializeSessions(ctx, tx); err != nil {
			return err
		}

		// Always create KeyValueLog bucket.
		if err := c.initializeKeyValueLog(ctx, tx); err != nil {
			return err
		}

		// Always create SecretService bucket.
		if err := c.initializeSecretService(ctx, tx); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// Close the connection to the bolt database
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
