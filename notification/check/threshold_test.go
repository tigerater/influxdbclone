package check_test

import (
	"testing"

	"github.com/influxdata/flux/ast"
	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/notification"
	"github.com/influxdata/influxdb/notification/check"
)

func TestThreshold_GenerateFlux(t *testing.T) {
	type args struct {
		threshold check.Threshold
	}
	type wants struct {
		script string
	}

	var l float64 = 10
	var u float64 = 40

	tests := []struct {
		name  string
		args  args
		wants wants
	}{
		{
			name: "all levels",
			args: args{
				threshold: check.Threshold{
					Base: check.Base{
						ID:   10,
						Name: "moo",
						Tags: []notification.Tag{
							{Key: "aaa", Value: "vaaa"},
							{Key: "bbb", Value: "vbbb"},
						},
						Every:                 mustDuration("1h"),
						StatusMessageTemplate: "whoa! {check.yeah}",
						Query: influxdb.DashboardQuery{
							Text: `from(bucket: "foo") |> range(start: -1d) |> aggregateWindow(every: 1m, fn: mean)`,
						},
					},
					Thresholds: []check.ThresholdConfig{
						check.Greater{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Ok,
							},
							Value: l,
						},
						check.Lesser{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Info,
							},
							Value: u,
						},
						check.Range{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Warn,
							},
							Min:    l,
							Max:    u,
							Within: true,
						},
						check.Range{
							ThresholdConfigBase: check.ThresholdConfigBase{
								Level: notification.Critical,
							},
							Min:    l,
							Max:    u,
							Within: true,
						},
					},
				},
			},
			wants: wants{
				script: `package main
import "influxdata/influxdb/alerts"
import "influxdata/influxdb/v1"

data = from(bucket: "foo")
	|> range(start: -1h)
	|> aggregateWindow(every: 1h, fn: mean)

option task = {name: "moo", every: 1h}

check = {
	_check_id: "000000000000000a",
	_check_name: "moo",
	_check_type: "threshold",
	tags: {aaa: "vaaa", bbb: "vbbb"},
}
ok = (r) =>
	(r._value > 10.0)
info = (r) =>
	(r._value < 40.0)
warn = (r) =>
	(r._value < 40.0 and r._value > 10.0)
crit = (r) =>
	(r._value < 40.0 and r._value > 10.0)
messageFn = (r, check) =>
	("whoa! {check.yeah}")

data
	|> v1.fieldsAsCols()
	|> alerts.check(
		data: check,
		messageFn: messageFn,
		ok: ok,
		info: info,
		warn: warn,
		crit: crit,
	)`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO(desa): change this to GenerateFlux() when we don't need to code
			// around the alerts package not being available.
			p, err := tt.args.threshold.GenerateFluxASTReal()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if exp, got := tt.wants.script, ast.Format(p); exp != got {
				t.Errorf("expected:\n%v\n\ngot:\n%v\n", exp, got)
			}
		})
	}

}
