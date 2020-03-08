package influxdb_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	platform "github.com/influxdata/influxdb"
	_ "github.com/influxdata/influxdb/query/builtin"
	"github.com/influxdata/influxdb/task/options"
)

func TestOptionsMarshal(t *testing.T) {
	tu := &platform.TaskUpdate{}
	// this is to make sure that string durations are properly marshaled into durations
	if err := json.Unmarshal([]byte(`{"every":"10s", "offset":"1h"}`), tu); err != nil {
		t.Fatal(err)
	}
	if tu.Options.Every.String() != "10s" {
		t.Fatalf("option.every not properly unmarshaled, expected 10s got %s", tu.Options.Every)
	}
	if tu.Options.Offset.String() != "1h" {
		t.Fatalf("option.every not properly unmarshaled, expected 1h got %s", tu.Options.Offset)
	}

	tu = &platform.TaskUpdate{}
	// this is to make sure that string durations are properly marshaled into durations
	if err := json.Unmarshal([]byte(`{"flux":"option task = {\n\tname: \"task #99\",\n\tcron: \"* * * * *\",\n\toffset: 5s,\n\tconcurrency: 100,\n}\nfrom(bucket:\"b\") |\u003e toHTTP(url:\"http://example.com\")"}`), tu); err != nil {
		t.Fatal(err)
	}

	if tu.Flux == nil {
		t.Fatalf("flux not properly unmarshaled, expected not nil but got nil")
	}
}

func TestOptionsEdit(t *testing.T) {
	tu := &platform.TaskUpdate{}
	tu.Options.Every = *(options.MustParseDuration("10s"))
	if err := tu.UpdateFlux(`option task = {every: 20s, name: "foo"} from(bucket:"x") |> range(start:-1h)`); err != nil {
		t.Fatal(err)
	}
	t.Run("zeroing", func(t *testing.T) {
		if !tu.Options.Every.IsZero() {
			t.Errorf("expected Every to be zeroed but it was not")
		}
	})
	t.Run("fmt string", func(t *testing.T) {
		expected := `option task = {every: 10s, name: "foo"}

from(bucket: "x")
	|> range(start: -1h)`
		if *tu.Flux != expected {
			t.Errorf("got the wrong task back, expected %s,\n got %s\n diff: %s", expected, *tu.Flux, cmp.Diff(expected, *tu.Flux))
		}
	})
	t.Run("replacement", func(t *testing.T) {
		op, err := options.FromScript(*tu.Flux)
		if err != nil {
			t.Error(err)
		}
		if op.Every.String() != "10s" {
			t.Logf("expected every to be 10s but was %s", op.Every)
			t.Fail()
		}
	})
	t.Run("add new option", func(t *testing.T) {
		tu := &platform.TaskUpdate{}
		tu.Options.Offset = options.MustParseDuration("30s")
		if err := tu.UpdateFlux(`option task = {every: 20s, name: "foo"} from(bucket:"x") |> range(start:-1h)`); err != nil {
			t.Fatal(err)
		}
		op, err := options.FromScript(*tu.Flux)
		if err != nil {
			t.Error(err)
		}
		if op.Offset == nil || op.Offset.String() != "30s" {
			t.Fatalf("expected every to be 30s but was %s", op.Every)
		}
	})
	t.Run("switching from every to cron", func(t *testing.T) {
		tu := &platform.TaskUpdate{}
		tu.Options.Cron = "* * * * *"
		if err := tu.UpdateFlux(`option task = {every: 20s, name: "foo"} from(bucket:"x") |> range(start:-1h)`); err != nil {
			t.Fatal(err)
		}
		op, err := options.FromScript(*tu.Flux)
		if err != nil {
			t.Error(err)
		}
		if !op.Every.IsZero() {
			t.Fatalf("expected every to be 0 but was %s", op.Every)
		}
		if op.Cron != "* * * * *" {
			t.Fatalf("expected Cron to be \"* * * * *\" but was %s", op.Cron)
		}
	})
	t.Run("switching from cron to every", func(t *testing.T) {
		tu := &platform.TaskUpdate{}
		tu.Options.Every = *(options.MustParseDuration("10s"))
		if err := tu.UpdateFlux(`option task = {cron: "* * * * *", name: "foo"} from(bucket:"x") |> range(start:-1h)`); err != nil {
			t.Fatal(err)
		}
		op, err := options.FromScript(*tu.Flux)
		if err != nil {
			t.Error(err)
		}
		if op.Every.String() != "10s" {
			t.Fatalf("expected every to be 10s but was %s", op.Every)
		}
		if op.Cron != "" {
			t.Fatalf("expected Cron to be \"\" but was %s", op.Cron)
		}
	})
	t.Run("delete deletable option", func(t *testing.T) {
		tu := &platform.TaskUpdate{}
		tu.Options.Offset = &options.Duration{}
		expscript := `option task = {cron: "* * * * *", name: "foo"}

from(bucket: "x")
	|> range(start: -1h)`
		if err := tu.UpdateFlux(`option task = {cron: "* * * * *", name: "foo", offset: 10s} from(bucket:"x") |> range(start:-1h)`); err != nil {
			t.Fatal(err)
		}
		op, err := options.FromScript(*tu.Flux)
		if err != nil {
			t.Error(err)
		}
		if !op.Every.IsZero() {
			t.Fatalf("expected every to be 0s but was %s", op.Every)
		}
		if op.Cron != "* * * * *" {
			t.Fatalf("expected Cron to be \"\" but was %s", op.Cron)
		}
		if !cmp.Equal(*tu.Flux, expscript) {
			t.Fatalf(cmp.Diff(*tu.Flux, expscript))
		}
	})

}

func TestRun(t *testing.T) {
	t.Run("ScheduledForTime", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		r := platform.Run{
			ScheduledFor: now.Format(time.RFC3339),
		}
		schedFor, err := r.ScheduledForTime()
		if err != nil {
			t.Fatal(err)
		}
		if !now.Equal(schedFor) {
			t.Fatalf("expected scheduled for to match now time: exp: %s, got: %s", now.String(), schedFor.String())
		}
		r = platform.Run{
			ScheduledFor: "ugly time",
		}
		schedFor, err = r.ScheduledForTime()
		if err == nil {
			t.Fatal("failed to error with a bad time")
		}
	})
}
