package launcher_test

import (
	"fmt"
	"io/ioutil"
	nethttp "net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/cmd/influxd/launcher"
	"github.com/influxdata/influxdb/http"
	"github.com/influxdata/influxdb/toml"
	"github.com/influxdata/influxdb/tsdb/tsm1"
)

func TestStorage_WriteAndQuery(t *testing.T) {
	l := launcher.RunTestLauncherOrFail(t, ctx)

	org1 := l.OnBoardOrFail(t, &influxdb.OnboardingRequest{
		User:     "USER-1",
		Password: "PASSWORD-1",
		Org:      "ORG-01",
		Bucket:   "BUCKET",
	})
	org2 := l.OnBoardOrFail(t, &influxdb.OnboardingRequest{
		User:     "USER-2",
		Password: "PASSWORD-1",
		Org:      "ORG-02",
		Bucket:   "BUCKET",
	})

	defer l.ShutdownOrFail(t, ctx)

	// Execute single write against the server.
	l.WriteOrFail(t, org1, `m,k=v1 f=100i 946684800000000000`)
	l.WriteOrFail(t, org2, `m,k=v2 f=200i 946684800000000000`)

	qs := `from(bucket:"BUCKET") |> range(start:2000-01-01T00:00:00Z,stop:2000-01-02T00:00:00Z)`

	exp := `,result,table,_start,_stop,_time,_value,_field,_measurement,k` + "\r\n" +
		`,_result,0,2000-01-01T00:00:00Z,2000-01-02T00:00:00Z,2000-01-01T00:00:00Z,100,f,m,v1` + "\r\n\r\n"
	if got := l.FluxQueryOrFail(t, org1.Org, org1.Auth.Token, qs); !cmp.Equal(got, exp) {
		t.Errorf("unexpected query results -got/+exp\n%s", cmp.Diff(got, exp))
	}

	exp = `,result,table,_start,_stop,_time,_value,_field,_measurement,k` + "\r\n" +
		`,_result,0,2000-01-01T00:00:00Z,2000-01-02T00:00:00Z,2000-01-01T00:00:00Z,200,f,m,v2` + "\r\n\r\n"
	if got := l.FluxQueryOrFail(t, org2.Org, org2.Auth.Token, qs); !cmp.Equal(got, exp) {
		t.Errorf("unexpected query results -got/+exp\n%s", cmp.Diff(got, exp))
	}
}

func TestLauncher_WriteAndQuery(t *testing.T) {
	l := launcher.RunTestLauncherOrFail(t, ctx)
	l.SetupOrFail(t)
	defer l.ShutdownOrFail(t, ctx)

	// Execute single write against the server.
	resp, err := nethttp.DefaultClient.Do(l.MustNewHTTPRequest("POST", fmt.Sprintf("/api/v2/write?org=%s&bucket=%s", l.Org.ID, l.Bucket.ID), `m,k=v f=100i 946684800000000000`))
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != nethttp.StatusNoContent {
		t.Fatalf("unexpected status code: %d, body: %s, headers: %v", resp.StatusCode, body, resp.Header)
	}

	// Query server to ensure write persists.
	qs := `from(bucket:"BUCKET") |> range(start:2000-01-01T00:00:00Z,stop:2000-01-02T00:00:00Z)`
	exp := `,result,table,_start,_stop,_time,_value,_field,_measurement,k` + "\r\n" +
		`,_result,0,2000-01-01T00:00:00Z,2000-01-02T00:00:00Z,2000-01-01T00:00:00Z,100,f,m,v` + "\r\n\r\n"

	buf, err := http.SimpleQuery(l.URL(), qs, l.Org.Name, l.Auth.Token)
	if err != nil {
		t.Fatalf("unexpected error querying server: %v", err)
	}
	if diff := cmp.Diff(string(buf), exp); diff != "" {
		t.Fatal(diff)
	}
}

func TestLauncher_BucketDelete(t *testing.T) {
	l := launcher.RunTestLauncherOrFail(t, ctx)
	l.SetupOrFail(t)
	defer l.ShutdownOrFail(t, ctx)

	// Execute single write against the server.
	resp, err := nethttp.DefaultClient.Do(l.MustNewHTTPRequest("POST", fmt.Sprintf("/api/v2/write?org=%s&bucket=%s", l.Org.ID, l.Bucket.ID), `m,k=v f=100i 946684800000000000`))
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != nethttp.StatusNoContent {
		t.Fatalf("unexpected status code: %d, body: %s, headers: %v", resp.StatusCode, body, resp.Header)
	}

	// Query server to ensure write persists.
	qs := `from(bucket:"BUCKET") |> range(start:2000-01-01T00:00:00Z,stop:2000-01-02T00:00:00Z)`
	exp := `,result,table,_start,_stop,_time,_value,_field,_measurement,k` + "\r\n" +
		`,_result,0,2000-01-01T00:00:00Z,2000-01-02T00:00:00Z,2000-01-01T00:00:00Z,100,f,m,v` + "\r\n\r\n"

	buf, err := http.SimpleQuery(l.URL(), qs, l.Org.Name, l.Auth.Token)
	if err != nil {
		t.Fatalf("unexpected error querying server: %v", err)
	}
	if diff := cmp.Diff(string(buf), exp); diff != "" {
		t.Fatal(diff)
	}

	// Verify the cardinality in the engine.
	engine := l.Launcher.Engine()
	if got, exp := engine.SeriesCardinality(), int64(1); got != exp {
		t.Fatalf("got %d, exp %d", got, exp)
	}

	// Delete the bucket.
	if resp, err = nethttp.DefaultClient.Do(l.MustNewHTTPRequest("DELETE", fmt.Sprintf("/api/v2/buckets/%s", l.Bucket.ID), "")); err != nil {
		t.Fatal(err)
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}

	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != nethttp.StatusNoContent {
		t.Fatalf("unexpected status code: %d, body: %s, headers: %v", resp.StatusCode, body, resp.Header)
	}

	// Verify that the data has been removed from the storage engine.
	if got, exp := engine.SeriesCardinality(), int64(0); got != exp {
		t.Fatalf("after bucket delete got %d, exp %d", got, exp)
	}
}

func TestStorage_CacheSnapshot_Size(t *testing.T) {
	l := launcher.NewTestLauncher()
	l.StorageConfig.Engine.Cache.SnapshotMemorySize = 10
	l.StorageConfig.Engine.Cache.SnapshotAgeDuration = toml.Duration(time.Hour)
	defer l.ShutdownOrFail(t, ctx)

	if err := l.Run(ctx); err != nil {
		t.Fatal(err)
	}

	l.SetupOrFail(t)

	org1 := l.OnBoardOrFail(t, &influxdb.OnboardingRequest{
		User:     "USER-1",
		Password: "PASSWORD-1",
		Org:      "ORG-01",
		Bucket:   "BUCKET",
	})

	// Execute single write against the server.
	l.WriteOrFail(t, org1, `m,k=v1 f=100i 946684800000000000`)
	l.WriteOrFail(t, org1, `m,k=v2 f=101i 946684800000000000`)
	l.WriteOrFail(t, org1, `m,k=v3 f=102i 946684800000000000`)
	l.WriteOrFail(t, org1, `m,k=v4 f=103i 946684800000000000`)
	l.WriteOrFail(t, org1, `m,k=v5 f=104i 946684800000000000`)

	// Wait for cache to snapshot. This should take no longer than one second.
	time.Sleep(time.Second * 5)

	// Check there is TSM data.
	report := tsm1.Report{
		Dir:   filepath.Join(l.Path, "/engine/data"),
		Exact: true,
	}

	summary, err := report.Run(false)
	if err != nil {
		t.Fatal(err)
	}

	// Five series should be in the snapshot
	if got, exp := summary.Total, uint64(5); got != exp {
		t.Fatalf("got %d series in TSM files, expected %d", got, exp)
	}
}

func TestStorage_CacheSnapshot_Age(t *testing.T) {
	l := launcher.NewTestLauncher()
	l.StorageConfig.Engine.Cache.SnapshotAgeDuration = toml.Duration(time.Second)
	defer l.ShutdownOrFail(t, ctx)

	if err := l.Run(ctx); err != nil {
		t.Fatal(err)
	}

	l.SetupOrFail(t)

	org1 := l.OnBoardOrFail(t, &influxdb.OnboardingRequest{
		User:     "USER-1",
		Password: "PASSWORD-1",
		Org:      "ORG-01",
		Bucket:   "BUCKET",
	})

	// Execute single write against the server.
	l.WriteOrFail(t, org1, `m,k=v1 f=100i 946684800000000000`)
	l.WriteOrFail(t, org1, `m,k=v2 f=101i 946684800000000000`)
	l.WriteOrFail(t, org1, `m,k=v3 f=102i 946684800000000000`)
	l.WriteOrFail(t, org1, `m,k=v4 f=102i 946684800000000000`)
	l.WriteOrFail(t, org1, `m,k=v5 f=102i 946684800000000000`)

	// Wait for cache to snapshot. This should take no longer than one second.
	time.Sleep(time.Second * 5)

	// Check there is TSM data.
	report := tsm1.Report{
		Dir:   filepath.Join(l.Path, "/engine/data"),
		Exact: true,
	}

	summary, err := report.Run(false)
	if err != nil {
		t.Fatal(err)
	}

	// Five series should be in the snapshot
	if got, exp := summary.Total, uint64(5); got != exp {
		t.Fatalf("got %d series in TSM files, expected %d", got, exp)
	}
}
