package pkger

import (
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/notification"
	icheck "github.com/influxdata/influxdb/notification/check"
	"github.com/influxdata/influxdb/notification/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("pkg with a bucket", func(t *testing.T) {
		t.Run("with valid bucket pkg should be valid", func(t *testing.T) {
			testfileRunner(t, "testdata/bucket", func(t *testing.T, pkg *Pkg) {
				buckets := pkg.Summary().Buckets
				require.Len(t, buckets, 1)

				actual := buckets[0]
				expectedBucket := SummaryBucket{
					Name:              "rucket_11",
					Description:       "bucket 1 description",
					RetentionPeriod:   time.Hour,
					LabelAssociations: []SummaryLabel{},
				}
				assert.Equal(t, expectedBucket, actual)
			})
		})

		t.Run("handles bad config", func(t *testing.T) {
			tests := []testPkgResourceError{
				{
					name:           "missing name",
					validationErrs: 1,
					valFields:      []string{fieldName},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
spec:
`,
				},
				{
					name:           "mixed valid and missing name",
					validationErrs: 1,
					valFields:      []string{fieldName},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
  name:  rucket_11
---
apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
spec:
`,
				},
				{
					name:           "mixed valid and multiple bad names",
					resourceErrs:   2,
					validationErrs: 1,
					valFields:      []string{fieldName},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
  name:  rucket_11
---
apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
spec:
---
apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
spec:
`,
				},
				{
					name:           "duplicate bucket names",
					resourceErrs:   1,
					validationErrs: 1,
					valFields:      []string{fieldName},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
  name:  valid name
---
apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
  name:  valid name
`,
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, KindBucket, tt)
			}
		})
	})

	t.Run("pkg with a label", func(t *testing.T) {
		t.Run("with valid label pkg should be valid", func(t *testing.T) {
			testfileRunner(t, "testdata/label", func(t *testing.T, pkg *Pkg) {
				labels := pkg.labels()
				require.Len(t, labels, 2)

				expectedLabel1 := label{
					name:        "label_1",
					Description: "label 1 description",
					Color:       "#FFFFFF",
				}
				assert.Equal(t, expectedLabel1, *labels[0])

				expectedLabel2 := label{
					name:        "label_2",
					Description: "label 2 description",
					Color:       "#000000",
				}
				assert.Equal(t, expectedLabel2, *labels[1])
			})
		})

		t.Run("with missing label name should error", func(t *testing.T) {
			tests := []testPkgResourceError{
				{
					name:           "missing name",
					validationErrs: 1,
					valFields:      []string{"name"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
spec:
`,
				},
				{
					name:           "mixed valid and missing name",
					validationErrs: 1,
					valFields:      []string{"name"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: valid name
spec:
---
apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: a
spec:
`,
				},
				{
					name:           "multiple labels with missing name",
					resourceErrs:   2,
					validationErrs: 1,
					valFields:      []string{"name"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
---
apiVersion: influxdata.com/v2alpha1
kind: Label

`,
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, KindLabel, tt)
			}
		})
	})

	t.Run("pkg with buckets and labels associated", func(t *testing.T) {
		t.Run("happy path", func(t *testing.T) {
			testfileRunner(t, "testdata/bucket_associates_label", func(t *testing.T, pkg *Pkg) {
				sum := pkg.Summary()
				require.Len(t, sum.Labels, 2)

				bkts := sum.Buckets
				require.Len(t, bkts, 3)

				expectedLabels := []struct {
					bktName string
					labels  []string
				}{
					{
						bktName: "rucket_1",
						labels:  []string{"label_1"},
					},
					{
						bktName: "rucket_2",
						labels:  []string{"label_2"},
					},
					{
						bktName: "rucket_3",
						labels:  []string{"label_1", "label_2"},
					},
				}
				for i, expected := range expectedLabels {
					bkt := bkts[i]
					require.Len(t, bkt.LabelAssociations, len(expected.labels))

					for j, label := range expected.labels {
						assert.Equal(t, label, bkt.LabelAssociations[j].Name)
					}
				}

				expectedMappings := []SummaryLabelMapping{
					{
						ResourceName: "rucket_1",
						LabelName:    "label_1",
					},
					{
						ResourceName: "rucket_2",
						LabelName:    "label_2",
					},
					{
						ResourceName: "rucket_3",
						LabelName:    "label_1",
					},
					{
						ResourceName: "rucket_3",
						LabelName:    "label_2",
					},
				}

				require.Len(t, sum.LabelMappings, len(expectedMappings))
				for i, expected := range expectedMappings {
					expected.ResourceType = influxdb.BucketsResourceType
					assert.Equal(t, expected, sum.LabelMappings[i])
				}
			})
		})

		t.Run("association doesn't exist then provides an error", func(t *testing.T) {
			tests := []testPkgResourceError{
				{
					name:    "no labels provided",
					assErrs: 1,
					assIdxs: []int{0},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
  name: rucket_1
spec:
  associations:
    - kind: Label
      name: label_1
`,
				},
				{
					name:    "mixed found and not found",
					assErrs: 1,
					assIdxs: []int{1},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: label_1
---
apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
  name: rucket_3
spec:
  associations:
    - kind: Label
      name: label_1
    - kind: Label
      name: NOT TO BE FOUND
`,
				},
				{
					name:    "multiple not found",
					assErrs: 1,
					assIdxs: []int{0, 1},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
  name: rucket_3
spec:
  associations:
    - kind: Label
      name: label_1
    - kind: Label
      name: label_2
`,
				},
				{
					name:    "duplicate valid nested labels",
					assErrs: 1,
					assIdxs: []int{1},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: label_1
---
apiVersion: influxdata.com/v2alpha1
kind: Bucket
metadata:
  name: rucket_3
spec:
  associations:
    - kind: Label
      name: label_1
    - kind: Label
      name: label_1
`,
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, KindBucket, tt)
			}
		})
	})

	t.Run("pkg with checks", func(t *testing.T) {
		t.Run("happy path", func(t *testing.T) {
			testfileRunner(t, "testdata/checks", func(t *testing.T, pkg *Pkg) {
				sum := pkg.Summary()
				require.Len(t, sum.Checks, 2)

				check1 := sum.Checks[0]
				thresholdCheck, ok := check1.Check.(*icheck.Threshold)
				require.Truef(t, ok, "got: %#v", check1)

				expectedBase := icheck.Base{
					Name:                  "check_0",
					Description:           "desc_0",
					Every:                 mustDuration(t, time.Minute),
					Offset:                mustDuration(t, 15*time.Second),
					StatusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }",
					Tags: []influxdb.Tag{
						{Key: "tag_1", Value: "val_1"},
						{Key: "tag_2", Value: "val_2"},
					},
				}
				expectedBase.Query.Text = "from(bucket: \"rucket_1\")\n  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n  |> filter(fn: (r) => r._measurement == \"cpu\")\n  |> filter(fn: (r) => r._field == \"usage_idle\")\n  |> aggregateWindow(every: 1m, fn: mean)\n  |> yield(name: \"mean\")"
				assert.Equal(t, expectedBase, thresholdCheck.Base)

				expectedThresholds := []icheck.ThresholdConfig{
					icheck.Greater{
						ThresholdConfigBase: icheck.ThresholdConfigBase{
							AllValues: true,
							Level:     notification.Critical,
						},
						Value: 50.0,
					},
					icheck.Lesser{
						ThresholdConfigBase: icheck.ThresholdConfigBase{Level: notification.Warn},
						Value:               49.9,
					},
					icheck.Range{
						ThresholdConfigBase: icheck.ThresholdConfigBase{Level: notification.Info},
						Within:              true,
						Min:                 30.0,
						Max:                 45.0,
					},
					icheck.Range{
						ThresholdConfigBase: icheck.ThresholdConfigBase{Level: notification.Ok},
						Min:                 30.0,
						Max:                 35.0,
					},
				}
				assert.Equal(t, expectedThresholds, thresholdCheck.Thresholds)
				assert.Equal(t, influxdb.Inactive, check1.Status)
				assert.Len(t, check1.LabelAssociations, 1)

				check2 := sum.Checks[1]
				deadmanCheck, ok := check2.Check.(*icheck.Deadman)
				require.Truef(t, ok, "got: %#v", check2)

				expectedBase = icheck.Base{
					Name:                  "check_1",
					Description:           "desc_1",
					Every:                 mustDuration(t, 5*time.Minute),
					Offset:                mustDuration(t, 10*time.Second),
					StatusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }",
					Tags: []influxdb.Tag{
						{Key: "tag_1", Value: "val_1"},
						{Key: "tag_2", Value: "val_2"},
					},
				}
				expectedBase.Query.Text = "from(bucket: \"rucket_1\")\n  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)\n  |> filter(fn: (r) => r._measurement == \"cpu\")\n  |> filter(fn: (r) => r._field == \"usage_idle\")\n  |> aggregateWindow(every: 1m, fn: mean)\n  |> yield(name: \"mean\")"
				assert.Equal(t, expectedBase, deadmanCheck.Base)
				assert.Equal(t, influxdb.Active, check2.Status)
				assert.Equal(t, mustDuration(t, 10*time.Minute), deadmanCheck.StaleTime)
				assert.Equal(t, mustDuration(t, 90*time.Second), deadmanCheck.TimeSince)
				assert.True(t, deadmanCheck.ReportZero)
				assert.Len(t, check2.LabelAssociations, 1)

				containsLabelMappings(t, sum.LabelMappings,
					labelMapping{
						labelName: "label_1",
						resName:   "check_0",
						resType:   influxdb.ChecksResourceType,
					},
					labelMapping{
						labelName: "label_1",
						resName:   "check_1",
						resType:   influxdb.ChecksResourceType,
					},
				)
			})
		})

		t.Run("handles bad config", func(t *testing.T) {
			tests := []struct {
				kind   Kind
				resErr testPkgResourceError
			}{
				{
					kind: KindCheckDeadman,
					resErr: testPkgResourceError{
						name:           "duplicate name",
						validationErrs: 1,
						valFields:      []string{fieldName},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckDeadman
metadata:
  name: check_1
spec:
  every: 5m
  level: cRiT
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
---
apiVersion: influxdata.com/v2alpha1
kind: CheckDeadman
metadata:
  name: check_1
spec:
  every: 5m
  level: cRiT
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
`,
					},
				},
				{
					kind: KindCheckThreshold,
					resErr: testPkgResourceError{
						name:           "missing every duration",
						validationErrs: 1,
						valFields:      []string{fieldEvery},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckThreshold
metadata:
  name: check_0
spec:
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  thresholds:
    - type: outside_range
      level: ok
      min: 30.0
      max: 35.0

`,
					},
				},
				{
					kind: KindCheckThreshold,
					resErr: testPkgResourceError{
						name:           "invalid threshold value provided",
						validationErrs: 1,
						valFields:      []string{fieldLevel},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckThreshold
metadata:
  name: check_0
spec:
  every: 1m
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  thresholds:
    - type: greater
      level: RANDO
      value: 50.0
`,
					},
				},
				{
					kind: KindCheckThreshold,
					resErr: testPkgResourceError{
						name:           "invalid threshold type provided",
						validationErrs: 1,
						valFields:      []string{fieldType},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckThreshold
metadata:
  name: check_0
spec:
  every: 1m
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  thresholds:
    - type: RANDO_TYPE
      level: CRIT
      value: 50.0
`,
					},
				},
				{
					kind: KindCheckThreshold,
					resErr: testPkgResourceError{
						name:           "invalid min for inside range",
						validationErrs: 1,
						valFields:      []string{fieldMin},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckThreshold
metadata:
  name: check_0
spec:
  every: 1m
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  thresholds:
    - type: inside_range
      level: INfO
      min: 45.0
      max: 30.0
`,
					},
				},
				{
					kind: KindCheckThreshold,
					resErr: testPkgResourceError{
						name:           "no threshold values provided",
						validationErrs: 1,
						valFields:      []string{fieldCheckThresholds},
						pkgStr: `---
apiVersion: influxdata.com/v2alpha1
kind: CheckThreshold
metadata:
  name: check_0
spec:
  every: 1m
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  thresholds:
`,
					},
				},
				{
					kind: KindCheckThreshold,
					resErr: testPkgResourceError{
						name:           "threshold missing query",
						validationErrs: 1,
						valFields:      []string{fieldQuery},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckThreshold
metadata:
  name: check_0
spec:
  every: 1m
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  thresholds:
    - type: greater
      level: CRIT
      value: 50.0
`,
					},
				},
				{
					kind: KindCheckThreshold,
					resErr: testPkgResourceError{
						name:           "invalid status provided",
						validationErrs: 1,
						valFields:      []string{fieldStatus},
						pkgStr: `---
apiVersion: influxdata.com/v2alpha1
kind: CheckThreshold
metadata:
  name: check_0
spec:
  every: 1m
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  status: RANDO STATUS
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  thresholds:
    - type: greater
      level: CRIT
      value: 50.0
      allValues: true
`,
					},
				},
				{
					kind: KindCheckThreshold,
					resErr: testPkgResourceError{
						name:           "missing status message template",
						validationErrs: 1,
						valFields:      []string{fieldCheckStatusMessageTemplate},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckThreshold
metadata:
  name: check_0
spec:
  every: 1m
  query:  >
    from(bucket: "rucket_1")
  thresholds:
    - type: greater
      level: CRIT
      value: 50.0
`,
					},
				},
				{
					kind: KindCheckDeadman,
					resErr: testPkgResourceError{
						name:           "missing every",
						validationErrs: 1,
						valFields:      []string{fieldEvery},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckDeadman
metadata:
  name: check_1
spec:
  level: cRiT
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  timeSince: 90s
`,
					},
				},
				{
					kind: KindCheckDeadman,
					resErr: testPkgResourceError{
						name:           "deadman missing every",
						validationErrs: 1,
						valFields:      []string{fieldQuery},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckDeadman
metadata:
  name: check_1
spec:
  every: 5m
  level: cRiT
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  timeSince: 90s
`,
					},
				},
				{
					kind: KindCheckDeadman,
					resErr: testPkgResourceError{
						name:           "missing association label",
						validationErrs: 1,
						valFields:      []string{fieldAssociations},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: CheckDeadman
metadata:
  name: check_1
spec:
  every: 5m
  level: cRiT
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  timeSince: 90s
  associations:
    - kind: Label
      name: label_1
`,
					},
				},
				{
					kind: KindCheckDeadman,
					resErr: testPkgResourceError{
						name:           "duplicate association labels",
						validationErrs: 1,
						valFields:      []string{fieldAssociations},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: label_1
---
apiVersion: influxdata.com/v2alpha1
kind: CheckDeadman
metadata:
  name: check_1
spec:
  every: 5m
  level: cRiT
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  statusMessageTemplate: "Check: ${ r._check_name } is: ${ r._level }"
  timeSince: 90s
  associations:
    - kind: Label
      name: label_1
    - kind: Label
      name: label_1
`,
					},
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, tt.kind, tt.resErr)
			}
		})
	})

	t.Run("pkg with single dashboard and single chart", func(t *testing.T) {
		t.Run("single gauge chart", func(t *testing.T) {
			t.Run("happy path", func(t *testing.T) {
				testfileRunner(t, "testdata/dashboard_gauge", func(t *testing.T, pkg *Pkg) {
					sum := pkg.Summary()
					require.Len(t, sum.Dashboards, 1)

					actual := sum.Dashboards[0]
					assert.Equal(t, "dash_1", actual.Name)
					assert.Equal(t, "desc1", actual.Description)

					require.Len(t, actual.Charts, 1)
					actualChart := actual.Charts[0]
					assert.Equal(t, 3, actualChart.Height)
					assert.Equal(t, 6, actualChart.Width)
					assert.Equal(t, 1, actualChart.XPosition)
					assert.Equal(t, 2, actualChart.YPosition)

					props, ok := actualChart.Properties.(influxdb.GaugeViewProperties)
					require.True(t, ok)
					assert.Equal(t, "gauge", props.GetType())
					assert.Equal(t, "gauge note", props.Note)
					assert.True(t, props.ShowNoteWhenEmpty)

					require.Len(t, props.Queries, 1)
					q := props.Queries[0]
					queryText := `from(bucket: v.bucket)  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)  |> filter(fn: (r) => r._measurement == "boltdb_writes_total")  |> filter(fn: (r) => r._field == "counter")`
					assert.Equal(t, queryText, q.Text)
					assert.Equal(t, "advanced", q.EditMode)

					require.Len(t, props.ViewColors, 3)
					c := props.ViewColors[0]
					assert.Equal(t, "laser", c.Name)
					assert.Equal(t, "min", c.Type)
					assert.Equal(t, "#8F8AF4", c.Hex)
					assert.Equal(t, 0.0, c.Value)
				})
			})

			t.Run("handles invalid config", func(t *testing.T) {
				tests := []testPkgResourceError{
					{
						name:           "missing threshold color type",
						validationErrs: 1,
						valFields:      []string{"charts[0].colors"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   gauge
      name:   gauge
      note: gauge note
      noteOnEmpty: true
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)  |> filter(fn: (r) => r._measurement == "boltdb_writes_total")  |> filter(fn: (r) => r._field == "counter")
      colors:
        - name: laser
          type: min
          hex: "#8F8AF4"
          value: 0
        - name: laser
          type: max
          hex: "#8F8AF4"
          value: 5000
`,
					},
					{
						name:           "color mixing a hex value",
						validationErrs: 1,
						valFields:      []string{"charts[0].colors[0].hex"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   gauge
      name:   gauge
      note: gauge note
      noteOnEmpty: true
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)  |> filter(fn: (r) => r._measurement == "boltdb_writes_total")  |> filter(fn: (r) => r._field == "counter")
      colors:
        - name: laser
          type: min
          value: 0
        - name: laser
          type: threshold
          hex: "#8F8AF4"
          value: 700
        - name: laser
          type: max
          hex: "#8F8AF4"
          value: 5000
`,
					},
					{
						name:           "missing a query value",
						validationErrs: 1,
						valFields:      []string{"charts[0].queries[0].query"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   gauge
      name:   gauge
      note: gauge note
      noteOnEmpty: true
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      queries:
        - query:
      colors:
        - name: laser
          type: min
          hex: "#8F8AF4"
          value: 0
        - name: laser
          type: threshold
          hex: "#8F8AF4"
          value: 700
        - name: laser
          type: max
          hex: "#8F8AF4"
          value: 5000
`,
					},
				}

				for _, tt := range tests {
					testPkgErrors(t, KindDashboard, tt)
				}
			})
		})

		t.Run("single heatmap chart", func(t *testing.T) {
			t.Run("happy path", func(t *testing.T) {
				testfileRunner(t, "testdata/dashboard_heatmap", func(t *testing.T, pkg *Pkg) {
					sum := pkg.Summary()
					require.Len(t, sum.Dashboards, 1)

					actual := sum.Dashboards[0]
					assert.Equal(t, "dashboard w/ single heatmap chart", actual.Name)
					assert.Equal(t, "a dashboard w/ heatmap chart", actual.Description)

					require.Len(t, actual.Charts, 1)
					actualChart := actual.Charts[0]
					assert.Equal(t, 3, actualChart.Height)
					assert.Equal(t, 6, actualChart.Width)
					assert.Equal(t, 1, actualChart.XPosition)
					assert.Equal(t, 2, actualChart.YPosition)

					props, ok := actualChart.Properties.(influxdb.HeatmapViewProperties)
					require.True(t, ok)
					assert.Equal(t, "heatmap", props.GetType())
					assert.Equal(t, "heatmap note", props.Note)
					assert.Equal(t, int32(10), props.BinSize)
					assert.True(t, props.ShowNoteWhenEmpty)

					assert.Equal(t, []float64{0, 10}, props.XDomain)
					assert.Equal(t, []float64{0, 100}, props.YDomain)

					require.Len(t, props.Queries, 1)
					q := props.Queries[0]
					queryText := `from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")`
					assert.Equal(t, queryText, q.Text)
					assert.Equal(t, "advanced", q.EditMode)

					require.Len(t, props.ViewColors, 12)
					c := props.ViewColors[0]
					assert.Equal(t, "#000004", c)
				})
			})

			t.Run("handles invalid config", func(t *testing.T) {
				tests := []testPkgResourceError{
					{
						name:           "a color is missing a hex value",
						validationErrs: 1,
						valFields:      []string{"charts[0].colors[2].hex"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dashboard w/ single heatmap chart
spec:
  charts:
    - kind:   heatmap
      name:   heatmap
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      binSize: 10
      xCol: _time
      yCol: _value
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - hex: "#fbb61a"
        - hex: "#f4df53"
        - hex: ""
      axes:
        - name: "x"
          label: "x_label"
          prefix: "x_prefix"
          suffix: "x_suffix"
          domain:
            - 0
            - 10
        - name: "y"
          label: "y_label"
          prefix: "y_prefix"
          suffix: "y_suffix"
          domain:
            - 0
            - 100
`,
					},
					{
						name:           "missing axes",
						validationErrs: 1,
						valFields:      []string{"charts[0].axes"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dashboard w/ single heatmap chart
spec:
  charts:
    - kind:   heatmap
      name:   heatmap
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      binSize: 10
      xCol: _time
      yCol: _value
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - hex: "#000004"
`,
					},
					{
						name:           "missing a query value",
						validationErrs: 1,
						valFields:      []string{"charts[0].queries[0].query"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dashboard w/ single heatmap chart
spec:
  charts:
    - kind:   heatmap
      name:   heatmap
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      binSize: 10
      xCol: _time
      yCol: _value
      queries:
        - query:
      colors:
        - hex: "#000004"
      axes:
        - name: "x"
          label: "x_label"
          prefix: "x_prefix"
          suffix: "x_suffix"
          domain:
            - 0
            - 10
        - name: "y"
          label: "y_label"
          prefix: "y_prefix"
          suffix: "y_suffix"
          domain:
            - 0
            - 100
`,
					},
				}

				for _, tt := range tests {
					testPkgErrors(t, KindDashboard, tt)
				}
			})
		})

		t.Run("single histogram chart", func(t *testing.T) {
			t.Run("happy path", func(t *testing.T) {
				testfileRunner(t, "testdata/dashboard_histogram", func(t *testing.T, pkg *Pkg) {
					sum := pkg.Summary()
					require.Len(t, sum.Dashboards, 1)

					actual := sum.Dashboards[0]
					assert.Equal(t, "dashboard w/ single histogram chart", actual.Name)
					assert.Equal(t, "a dashboard w/ single histogram chart", actual.Description)

					require.Len(t, actual.Charts, 1)
					actualChart := actual.Charts[0]
					assert.Equal(t, 3, actualChart.Height)
					assert.Equal(t, 6, actualChart.Width)

					props, ok := actualChart.Properties.(influxdb.HistogramViewProperties)
					require.True(t, ok)
					assert.Equal(t, "histogram", props.GetType())
					assert.Equal(t, "histogram note", props.Note)
					assert.Equal(t, 30, props.BinCount)
					assert.True(t, props.ShowNoteWhenEmpty)
					assert.Equal(t, []float64{0, 10}, props.XDomain)
					assert.Equal(t, []string{}, props.FillColumns)
					require.Len(t, props.Queries, 1)
					q := props.Queries[0]
					queryText := `from(bucket: v.bucket) |> range(start: v.timeRangeStart, stop: v.timeRangeStop) |> filter(fn: (r) => r._measurement == "boltdb_reads_total") |> filter(fn: (r) => r._field == "counter")`
					assert.Equal(t, queryText, q.Text)
					assert.Equal(t, "advanced", q.EditMode)

					require.Len(t, props.ViewColors, 3)
					assert.Equal(t, "#8F8AF4", props.ViewColors[0].Hex)
					assert.Equal(t, "#F4CF31", props.ViewColors[1].Hex)
					assert.Equal(t, "#FFFFFF", props.ViewColors[2].Hex)
				})
			})

			t.Run("handles invalid config", func(t *testing.T) {
				tests := []testPkgResourceError{
					{
						name:           "missing x-axis",
						validationErrs: 1,
						valFields:      []string{"charts[0].axes"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dashboard w/ single histogram chart
spec:
  description: a dashboard w/ single histogram chart
  charts:
    - kind: Histogram
      name: histogram chart
      xCol: _value
      width:  6
      height: 3
      binCount: 30
      queries:
        - query: >
            from(bucket: v.bucket) |> range(start: v.timeRangeStart, stop: v.timeRangeStop) |> filter(fn: (r) => r._measurement == "boltdb_reads_total") |> filter(fn: (r) => r._field == "counter")
      colors:
        - hex: "#8F8AF4"
          type: scale
          value: 0
          name: mycolor
      axes:
`,
					},
					{
						name:           "missing a query value",
						validationErrs: 1,
						valFields:      []string{"charts[0].queries[0].query"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dashboard w/ single histogram chart
spec:
  description: a dashboard w/ single histogram chart
  charts:
    - kind: Histogram
      name: histogram chart
      xCol: _value
      width:  6
      height: 3
      binCount: 30
      queries:
        - query:
      colors:
        - hex: "#8F8AF4"
          type: scale
          value: 0
          name: mycolor
      axes:
        - name : "x"
          label: x_label
          domain:
            - 0
            - 10
`,
					},
				}

				for _, tt := range tests {
					testPkgErrors(t, KindDashboard, tt)
				}
			})
		})

		t.Run("single markdown chart", func(t *testing.T) {
			t.Run("happy path", func(t *testing.T) {
				testfileRunner(t, "testdata/dashboard_markdown", func(t *testing.T, pkg *Pkg) {
					sum := pkg.Summary()
					require.Len(t, sum.Dashboards, 1)

					actual := sum.Dashboards[0]
					assert.Equal(t, "dashboard w/ single markdown chart", actual.Name)
					assert.Equal(t, "a dashboard w/ single markdown chart", actual.Description)

					require.Len(t, actual.Charts, 1)
					actualChart := actual.Charts[0]

					props, ok := actualChart.Properties.(influxdb.MarkdownViewProperties)
					require.True(t, ok)
					assert.Equal(t, "markdown", props.GetType())
					assert.Equal(t, "## markdown note", props.Note)
				})
			})
		})

		t.Run("single scatter chart", func(t *testing.T) {
			t.Run("happy path", func(t *testing.T) {
				testfileRunner(t, "testdata/dashboard_scatter", func(t *testing.T, pkg *Pkg) {
					sum := pkg.Summary()
					require.Len(t, sum.Dashboards, 1)

					actual := sum.Dashboards[0]
					assert.Equal(t, "dashboard w/ single scatter chart", actual.Name)
					assert.Equal(t, "a dashboard w/ single scatter chart", actual.Description)

					require.Len(t, actual.Charts, 1)
					actualChart := actual.Charts[0]
					assert.Equal(t, 3, actualChart.Height)
					assert.Equal(t, 6, actualChart.Width)
					assert.Equal(t, 1, actualChart.XPosition)
					assert.Equal(t, 2, actualChart.YPosition)

					props, ok := actualChart.Properties.(influxdb.ScatterViewProperties)
					require.True(t, ok)
					assert.Equal(t, "scatter note", props.Note)
					assert.True(t, props.ShowNoteWhenEmpty)

					require.Len(t, props.Queries, 1)
					q := props.Queries[0]
					expectedQuery := `from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")`
					assert.Equal(t, expectedQuery, q.Text)
					assert.Equal(t, "advanced", q.EditMode)

					assert.Equal(t, []float64{0, 10}, props.XDomain)
					assert.Equal(t, []float64{0, 100}, props.YDomain)
					assert.Equal(t, "x_label", props.XAxisLabel)
					assert.Equal(t, "y_label", props.YAxisLabel)
					assert.Equal(t, "x_prefix", props.XPrefix)
					assert.Equal(t, "y_prefix", props.YPrefix)
					assert.Equal(t, "x_suffix", props.XSuffix)
					assert.Equal(t, "y_suffix", props.YSuffix)
				})
			})

			t.Run("handles invalid config", func(t *testing.T) {
				tests := []testPkgResourceError{
					{
						name:           "missing axes",
						validationErrs: 1,
						valFields:      []string{"charts[0].axes"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name:  dashboard w/ single scatter chart
spec:
  description: a dashboard w/ single scatter chart
  charts:
    - kind:   Scatter
      name:   scatter chart
      xPos:  1
      yPos:  2
      xCol: _time
      yCol: _value
      width:  6
      height: 3
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - hex: "#8F8AF4"
        - hex: "#F4CF31"
`,
					},
					{
						name:           "missing query value",
						validationErrs: 1,
						valFields:      []string{"charts[0].queries[0].query"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name:  dashboard w/ single scatter chart
spec:
  description: a dashboard w/ single scatter chart
  charts:
    - kind:   Scatter
      name:   scatter chart
      xPos:  1
      yPos:  2
      xCol: _time
      yCol: _value
      width:  6
      height: 3
      queries:
        - query:
      colors:
        - hex: "#8F8AF4"
        - hex: "#F4CF31"
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          domain:
            - 0
            - 10
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          domain:
            - 0
            - 100
`,
					},
					{
						name:           "no queries provided",
						validationErrs: 1,
						valFields:      []string{"charts[0].queries"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name:  dashboard w/ single scatter chart
spec:
  description: a dashboard w/ single scatter chart
  charts:
    - kind:   Scatter
      name:   scatter chart
      xPos:  1
      yPos:  2
      xCol: _time
      yCol: _value
      width:  6
      height: 3
      colors:
        - hex: "#8F8AF4"
        - hex: "#F4CF31"
        - hex: "#FFFFFF"
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          domain:
            - 0
            - 10
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          domain:
            - 0
            - 100
`,
					},
					{
						name:           "no width provided",
						validationErrs: 1,
						valFields:      []string{"charts[0].width"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name:  dashboard w/ single scatter chart
spec:
  description: a dashboard w/ single scatter chart
  charts:
    - kind:   Scatter
      name:   scatter chart
      xPos:  1
      yPos:  2
      xCol: _time
      yCol: _value
      height: 3
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - hex: "#8F8AF4"
        - hex: "#F4CF31"
        - hex: "#FFFFFF"
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          domain:
            - 0
            - 10
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          domain:
            - 0
            - 100
`,
					},
					{
						name:           "no height provided",
						validationErrs: 1,
						valFields:      []string{"charts[0].height"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name:  dashboard w/ single scatter chart
spec:
  description: a dashboard w/ single scatter chart
  charts:
    - kind:   Scatter
      name:   scatter chart
      xPos:  1
      yPos:  2
      xCol: _time
      yCol: _value
      width:  6
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - hex: "#8F8AF4"
        - hex: "#F4CF31"
        - hex: "#FFFFFF"
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          domain:
            - 0
            - 10
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          domain:
            - 0
            - 100
`,
					},
					{
						name:           "missing hex color",
						validationErrs: 1,
						valFields:      []string{"charts[0].colors[0].hex"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name:  dashboard w/ single scatter chart
spec:
  description: a dashboard w/ single scatter chart
  charts:
    - kind:   Scatter
      name:   scatter chart
      note: scatter note
      noteOnEmpty: true
      prefix: sumtin
      suffix: days
      xPos:  1
      yPos:  2
      xCol: _time
      yCol: _value
      width:  6
      height: 3
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - hex: ""
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          domain:
            - 0
            - 10
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          domain:
            - 0
            - 100
`,
					},
					{
						name:           "missing x axis",
						validationErrs: 1,
						valFields:      []string{"charts[0].axes"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name:  dashboard w/ single scatter chart
spec:
  description: a dashboard w/ single scatter chart
  charts:
    - kind:   Scatter
      name:   scatter chart
      xPos:  1
      yPos:  2
      xCol: _time
      yCol: _value
      width:  6
      height: 3
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - hex: "#8F8AF4"
        - hex: "#F4CF31"
        - hex: "#FFFFFF"
      axes:
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          domain:
            - 0
            - 100
`,
					},
					{
						name:           "missing y axis",
						validationErrs: 1,
						valFields:      []string{"charts[0].axes"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name:  dashboard w/ single scatter chart
spec:
  description: a dashboard w/ single scatter chart
  charts:
    - kind:   Scatter
      name:   scatter chart
      xPos:  1
      yPos:  2
      xCol: _time
      yCol: _value
      width:  6
      height: 3
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - hex: "#8F8AF4"
        - hex: "#F4CF31"
        - hex: "#FFFFFF"
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          domain:
            - 0
            - 10
`,
					},
				}

				for _, tt := range tests {
					testPkgErrors(t, KindDashboard, tt)
				}
			})
		})

		t.Run("single stat chart", func(t *testing.T) {
			t.Run("happy path", func(t *testing.T) {
				testfileRunner(t, "testdata/dashboard", func(t *testing.T, pkg *Pkg) {
					sum := pkg.Summary()
					require.Len(t, sum.Dashboards, 1)

					actual := sum.Dashboards[0]
					assert.Equal(t, "dash_1", actual.Name)
					assert.Equal(t, "desc1", actual.Description)

					require.Len(t, actual.Charts, 1)
					actualChart := actual.Charts[0]
					assert.Equal(t, 3, actualChart.Height)
					assert.Equal(t, 6, actualChart.Width)
					assert.Equal(t, 1, actualChart.XPosition)
					assert.Equal(t, 2, actualChart.YPosition)

					props, ok := actualChart.Properties.(influxdb.SingleStatViewProperties)
					require.True(t, ok)
					assert.Equal(t, "single-stat", props.GetType())
					assert.Equal(t, "single stat note", props.Note)
					assert.True(t, props.ShowNoteWhenEmpty)
					assert.True(t, props.DecimalPlaces.IsEnforced)
					assert.Equal(t, int32(1), props.DecimalPlaces.Digits)
					assert.Equal(t, "days", props.Suffix)
					assert.Equal(t, "true", props.TickSuffix)
					assert.Equal(t, "sumtin", props.Prefix)
					assert.Equal(t, "true", props.TickPrefix)

					require.Len(t, props.Queries, 1)
					q := props.Queries[0]
					queryText := `from(bucket: v.bucket) |> range(start: v.timeRangeStart) |> filter(fn: (r) => r._measurement == "processes") |> filter(fn: (r) => r._field == "running" or r._field == "blocked") |> aggregateWindow(every: v.windowPeriod, fn: max) |> yield(name: "max")`
					assert.Equal(t, queryText, q.Text)
					assert.Equal(t, "advanced", q.EditMode)

					require.Len(t, props.ViewColors, 1)
					c := props.ViewColors[0]
					assert.Equal(t, "laser", c.Name)
					assert.Equal(t, "text", c.Type)
					assert.Equal(t, "#8F8AF4", c.Hex)
					assert.Equal(t, 3.0, c.Value)
				})
			})

			t.Run("handles invalid config", func(t *testing.T) {
				tests := []testPkgResourceError{
					{
						name:           "color missing hex value",
						validationErrs: 1,
						valFields:      []string{"charts[0].colors[0].hex"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat
      name:   single stat
      xPos: 1
      yPos: 2
      width:  6
      height: 3
      decimalPlaces: 1
      shade: true
      queries:
        - query: "from(bucket: v.bucket) |> range(start: v.timeRangeStart) |> filter(fn: (r) => r._measurement == \"processes\") |> filter(fn: (r) => r._field == \"running\" or r._field == \"blocked\") |> aggregateWindow(every: v.windowPeriod, fn: max) |> yield(name: \"max\")"
      colors:
        - name: laser
          type: text
          value: 3
`,
					},
					{
						name:           "query missing text value",
						validationErrs: 1,
						valFields:      []string{"charts[0].queries[0].query"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat
      name:   single stat
      xPos: 1
      yPos: 2
      width:  6
      height: 3
      queries:
        - query:
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
`,
					},
					{
						name:           "no queries provided",
						validationErrs: 1,
						valFields:      []string{"charts[0].queries"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat
      name:   single stat
      xPos: 1
      yPos: 2
      width:  6
      height: 3
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
`,
					},
					{
						name:           "no width provided",
						validationErrs: 1,
						valFields:      []string{"charts[0].width"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat
      name:   single stat
      xPos: 1
      yPos: 2
      height: 3
      queries:
        - query: "from(bucket: v.bucket) |> range(start: v.timeRangeStart) |> filter(fn: (r) => r._measurement == \"processes\") |> filter(fn: (r) => r._field == \"running\" or r._field == \"blocked\") |> aggregateWindow(every: v.windowPeriod, fn: max) |> yield(name: \"max\")"
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
`,
					},
					{
						name:           "no height provided",
						validationErrs: 1,
						valFields:      []string{"charts[0].height"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat
      name:   single stat
      xPos: 1
      yPos: 2
      width: 3
      queries:
        - query: "from(bucket: v.bucket) |> range(start: v.timeRangeStart) |> filter(fn: (r) => r._measurement == \"processes\") |> filter(fn: (r) => r._field == \"running\" or r._field == \"blocked\") |> aggregateWindow(every: v.windowPeriod, fn: max) |> yield(name: \"max\")"
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
`,
					},
				}

				for _, tt := range tests {
					testPkgErrors(t, KindDashboard, tt)
				}
			})
		})

		t.Run("single stat plus line chart", func(t *testing.T) {
			t.Run("happy path", func(t *testing.T) {
				testfileRunner(t, "testdata/dashboard_single_stat_plus_line", func(t *testing.T, pkg *Pkg) {
					sum := pkg.Summary()
					require.Len(t, sum.Dashboards, 1)

					actual := sum.Dashboards[0]
					assert.Equal(t, "dash_1", actual.Name)
					assert.Equal(t, "desc1", actual.Description)

					require.Len(t, actual.Charts, 1)
					actualChart := actual.Charts[0]
					assert.Equal(t, 3, actualChart.Height)
					assert.Equal(t, 6, actualChart.Width)
					assert.Equal(t, 1, actualChart.XPosition)
					assert.Equal(t, 2, actualChart.YPosition)

					props, ok := actualChart.Properties.(influxdb.LinePlusSingleStatProperties)
					require.True(t, ok)
					assert.Equal(t, "single stat plus line note", props.Note)
					assert.True(t, props.ShowNoteWhenEmpty)
					assert.True(t, props.DecimalPlaces.IsEnforced)
					assert.Equal(t, int32(1), props.DecimalPlaces.Digits)
					assert.Equal(t, "days", props.Suffix)
					assert.Equal(t, "sumtin", props.Prefix)
					assert.Equal(t, "overlaid", props.Position)
					assert.Equal(t, "leg_type", props.Legend.Type)
					assert.Equal(t, "horizontal", props.Legend.Orientation)

					require.Len(t, props.Queries, 1)
					q := props.Queries[0]
					expectedQuery := `from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")`
					assert.Equal(t, expectedQuery, q.Text)
					assert.Equal(t, "advanced", q.EditMode)

					for _, key := range []string{"x", "y"} {
						xAxis, ok := props.Axes[key]
						require.True(t, ok, "key="+key)
						assert.Equal(t, "10", xAxis.Base, "key="+key)
						assert.Equal(t, key+"_label", xAxis.Label, "key="+key)
						assert.Equal(t, key+"_prefix", xAxis.Prefix, "key="+key)
						assert.Equal(t, "linear", xAxis.Scale, "key="+key)
						assert.Equal(t, key+"_suffix", xAxis.Suffix, "key="+key)
					}

					require.Len(t, props.ViewColors, 2)
					c := props.ViewColors[0]
					assert.Equal(t, "laser", c.Name)
					assert.Equal(t, "text", c.Type)
					assert.Equal(t, "#8F8AF4", c.Hex)
					assert.Equal(t, 3.0, c.Value)

					c = props.ViewColors[1]
					assert.Equal(t, "android", c.Name)
					assert.Equal(t, "scale", c.Type)
					assert.Equal(t, "#F4CF31", c.Hex)
					assert.Equal(t, 1.0, c.Value)
				})
			})

			t.Run("handles invalid config", func(t *testing.T) {
				tests := []testPkgResourceError{
					{
						name:           "color missing hex value",
						validationErrs: 1,
						valFields:      []string{"charts[0].colors[0].hex"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat_Plus_Line
      name:   single stat plus line
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      position: overlaid
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - name: laser
          type: text
          value: 3
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          base: 10
          scale: linear
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          base: 10
          scale: linear
`,
					},
					{
						name:           "missing query value",
						validationErrs: 1,
						valFields:      []string{"charts[0].queries[0].query"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat_Plus_Line
      name:   single stat plus line
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      position: overlaid
      queries:
        - query:
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
          value: 3
        - name: android
          type: scale
          hex: "#F4CF31"
          value: 1
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          base: 10
          scale: linear
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          base: 10
          scale: linear
`,
					},
					{
						name:           "no queries provided",
						validationErrs: 1,
						valFields:      []string{"charts[0].queries"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat_Plus_Line
      name:   single stat plus line
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      position: overlaid
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
          value: 3
        - name: android
          type: scale
          hex: "#F4CF31"
          value: 1
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          base: 10
          scale: linear
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          base: 10
          scale: linear
`,
					},
					{
						name:           "no width provided",
						validationErrs: 1,
						valFields:      []string{"charts[0].width"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat_Plus_Line
      name:   single stat plus line
      xPos:  1
      yPos:  2
      height: 3
      shade: true
      position: overlaid
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
          value: 3
        - name: android
          type: scale
          hex: "#F4CF31"
          value: 1
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          base: 10
          scale: linear
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          base: 10
          scale: linear
`,
					},
					{
						name:           "no height provided",
						validationErrs: 1,
						valFields:      []string{"charts[0].height"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat_Plus_Line
      name:   single stat plus line
      xPos:  1
      yPos:  2
      width:  6
      position: overlaid
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
          value: 3
        - name: android
          type: scale
          hex: "#F4CF31"
          value: 1
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          base: 10
          scale: linear
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          base: 10
          scale: linear
`,
					},
					{
						name:           "missing text color but has scale color",
						validationErrs: 1,
						valFields:      []string{"charts[0].colors"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat_Plus_Line
      name:   single stat plus line
      suffix: days
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      position: overlaid
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - name: android
          type: scale
          hex: "#F4CF31"
          value: 1
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          base: 10
          scale: linear
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          base: 10
          scale: linear
`,
					},
					{
						name:           "missing x axis",
						validationErrs: 1,
						valFields:      []string{"charts[0].axes"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat_Plus_Line
      name:   single stat plus line
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      position: overlaid
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
          value: 3
        - name: android
          type: scale
          hex: "#F4CF31"
          value: 1
      axes:
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          base: 10
          scale: linear
`,
					},
					{
						name:           "missing y axis",
						validationErrs: 1,
						valFields:      []string{"charts[0].axes"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   Single_Stat_Plus_Line
      name:   single stat plus line
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      position: overlaid
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart)  |> filter(fn: (r) => r._measurement == "mem")  |> filter(fn: (r) => r._field == "used_percent")  |> aggregateWindow(every: v.windowPeriod, fn: mean, createEmpty: false)  |> yield(name: "mean")
      colors:
        - name: laser
          type: text
          hex: "#8F8AF4"
          value: 3
        - name: android
          type: scale
          hex: "#F4CF31"
          value: 1
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          base: 10
          scale: linear
`,
					},
				}

				for _, tt := range tests {
					testPkgErrors(t, KindDashboard, tt)
				}
			})
		})

		t.Run("single xy chart", func(t *testing.T) {
			t.Run("happy path", func(t *testing.T) {
				testfileRunner(t, "testdata/dashboard_xy", func(t *testing.T, pkg *Pkg) {
					sum := pkg.Summary()
					require.Len(t, sum.Dashboards, 1)

					actual := sum.Dashboards[0]
					assert.Equal(t, "dash_1", actual.Name)
					assert.Equal(t, "desc1", actual.Description)

					require.Len(t, actual.Charts, 1)
					actualChart := actual.Charts[0]
					assert.Equal(t, 3, actualChart.Height)
					assert.Equal(t, 6, actualChart.Width)
					assert.Equal(t, 1, actualChart.XPosition)
					assert.Equal(t, 2, actualChart.YPosition)

					props, ok := actualChart.Properties.(influxdb.XYViewProperties)
					require.True(t, ok)
					assert.Equal(t, "xy", props.GetType())
					assert.Equal(t, true, props.ShadeBelow)
					assert.Equal(t, "xy chart note", props.Note)
					assert.True(t, props.ShowNoteWhenEmpty)
					assert.Equal(t, "stacked", props.Position)

					require.Len(t, props.Queries, 1)
					q := props.Queries[0]
					queryText := `from(bucket: v.bucket)  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)  |> filter(fn: (r) => r._measurement == "boltdb_writes_total")  |> filter(fn: (r) => r._field == "counter")`
					assert.Equal(t, queryText, q.Text)
					assert.Equal(t, "advanced", q.EditMode)

					require.Len(t, props.ViewColors, 1)
					c := props.ViewColors[0]
					assert.Equal(t, "laser", c.Name)
					assert.Equal(t, "scale", c.Type)
					assert.Equal(t, "#8F8AF4", c.Hex)
					assert.Equal(t, 3.0, c.Value)
				})
			})

			t.Run("handles invalid config", func(t *testing.T) {
				tests := []testPkgResourceError{
					{
						name:           "color missing hex value",
						validationErrs: 1,
						valFields:      []string{"charts[0].colors[0].hex"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   XY
      name:   xy chart
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      geom: line
      position: stacked
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)  |> filter(fn: (r) => r._measurement == "boltdb_writes_total")  |> filter(fn: (r) => r._field == "counter")
      colors:
        - name: laser
          type: scale
          value: 3
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          base: 10
          scale: linear
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          base: 10
          scale: linear
`,
					},
					{
						name:           "invalid geom flag",
						validationErrs: 1,
						valFields:      []string{"charts[0].geom"},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  description: desc1
  charts:
    - kind:   XY
      name:   xy chart
      xPos:  1
      yPos:  2
      width:  6
      height: 3
      position: stacked
      legend:
      queries:
        - query: >
            from(bucket: v.bucket)  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)  |> filter(fn: (r) => r._measurement == "boltdb_writes_total")  |> filter(fn: (r) => r._field == "counter")
      colors:
        - name: laser
          type: scale
          hex: "#8F8AF4"
          value: 3
      axes:
        - name : "x"
          label: x_label
          prefix: x_prefix
          suffix: x_suffix
          base: 10
          scale: linear
        - name: "y"
          label: y_label
          prefix: y_prefix
          suffix: y_suffix
          base: 10
          scale: linear
`,
					},
				}

				for _, tt := range tests {
					testPkgErrors(t, KindDashboard, tt)
				}
			})
		})
	})

	t.Run("pkg with dashboard and labels associated", func(t *testing.T) {
		t.Run("happy path", func(t *testing.T) {
			testfileRunner(t, "testdata/dashboard_associates_label", func(t *testing.T, pkg *Pkg) {
				sum := pkg.Summary()
				require.Len(t, sum.Dashboards, 1)

				actual := sum.Dashboards[0]
				assert.Equal(t, "dash_1", actual.Name)

				require.Len(t, actual.LabelAssociations, 1)
				actualLabel := actual.LabelAssociations[0]
				assert.Equal(t, "label_1", actualLabel.Name)

				expectedMappings := []SummaryLabelMapping{
					{
						ResourceName: "dash_1",
						LabelName:    "label_1",
					},
				}
				require.Len(t, sum.LabelMappings, len(expectedMappings))

				for i, expected := range expectedMappings {
					expected.ResourceType = influxdb.DashboardsResourceType
					assert.Equal(t, expected, sum.LabelMappings[i])
				}
			})
		})

		t.Run("association doesn't exist then provides an error", func(t *testing.T) {
			tests := []testPkgResourceError{
				{
					name:    "no labels provided",
					assErrs: 1,
					assIdxs: []int{0},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  associations:
    - kind: Label
      name: label_1
`,
				},
				{
					name:    "mixed found and not found",
					assErrs: 1,
					assIdxs: []int{1},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: label_1
---
apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  associations:
    - kind: Label
      name: label_1
    - kind: Label
      name: unfound label
`,
				},
				{
					name:    "multiple not found",
					assErrs: 1,
					assIdxs: []int{0, 1},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: label_1
---
apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  associations:
    - kind: Label
      name: not found 1
    - kind: Label
      name: unfound label
`,
				},
				{
					name:    "duplicate valid nested labels",
					assErrs: 1,
					assIdxs: []int{1},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: label_1
---
apiVersion: influxdata.com/v2alpha1
kind: Dashboard
metadata:
  name: dash_1
spec:
  associations:
    - kind: Label
      name: label_1
    - kind: Label
      name: label_1
`,
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, KindDashboard, tt)
			}
		})
	})

	t.Run("pkg with notification endpoints and labels associated", func(t *testing.T) {
		t.Run("happy path", func(t *testing.T) {
			testfileRunner(t, "testdata/notification_endpoint", func(t *testing.T, pkg *Pkg) {
				expectedEndpoints := []SummaryNotificationEndpoint{
					{
						NotificationEndpoint: &endpoint.HTTP{
							Base: endpoint.Base{
								Name:        "http_basic_auth_notification_endpoint",
								Description: "http basic auth desc",
								Status:      influxdb.TaskStatusInactive,
							},
							URL:        "https://www.example.com/endpoint/basicauth",
							AuthMethod: "basic",
							Method:     "POST",
							Username:   influxdb.SecretField{Value: strPtr("secret username")},
							Password:   influxdb.SecretField{Value: strPtr("secret password")},
						},
					},
					{
						NotificationEndpoint: &endpoint.HTTP{
							Base: endpoint.Base{
								Name:        "http_bearer_auth_notification_endpoint",
								Description: "http bearer auth desc",
								Status:      influxdb.TaskStatusActive,
							},
							URL:        "https://www.example.com/endpoint/bearerauth",
							AuthMethod: "bearer",
							Method:     "PUT",
							Token:      influxdb.SecretField{Value: strPtr("secret token")},
						},
					},
					{
						NotificationEndpoint: &endpoint.HTTP{
							Base: endpoint.Base{
								Name:        "http_none_auth_notification_endpoint",
								Description: "http none auth desc",
								Status:      influxdb.TaskStatusActive,
							},
							URL:        "https://www.example.com/endpoint/noneauth",
							AuthMethod: "none",
							Method:     "GET",
						},
					},
					{
						NotificationEndpoint: &endpoint.PagerDuty{
							Base: endpoint.Base{
								Name:        "pager_duty_notification_endpoint",
								Description: "pager duty desc",
								Status:      influxdb.TaskStatusActive,
							},
							ClientURL:  "http://localhost:8080/orgs/7167eb6719fa34e5/alert-history",
							RoutingKey: influxdb.SecretField{Value: strPtr("secret routing-key")},
						},
					},
					{
						NotificationEndpoint: &endpoint.Slack{
							Base: endpoint.Base{
								Name:        "slack_notification_endpoint",
								Description: "slack desc",
								Status:      influxdb.TaskStatusActive,
							},
							URL:   "https://hooks.slack.com/services/bip/piddy/boppidy",
							Token: influxdb.SecretField{Value: strPtr("tokenval")},
						},
					},
				}

				sum := pkg.Summary()
				endpoints := sum.NotificationEndpoints
				require.Len(t, endpoints, len(expectedEndpoints))
				require.Len(t, sum.LabelMappings, len(expectedEndpoints))

				for i := range expectedEndpoints {
					expected, actual := expectedEndpoints[i], endpoints[i]
					assert.Equalf(t, expected.NotificationEndpoint, actual.NotificationEndpoint, "index=%d", i)
					require.Len(t, actual.LabelAssociations, 1)
					assert.Equal(t, "label_1", actual.LabelAssociations[0].Name)

					containsLabelMappings(t, sum.LabelMappings, labelMapping{
						labelName: "label_1",
						resName:   expected.NotificationEndpoint.GetName(),
						resType:   influxdb.NotificationEndpointResourceType,
					})
				}
			})
		})

		t.Run("handles bad config", func(t *testing.T) {
			tests := []struct {
				kind   Kind
				resErr testPkgResourceError
			}{
				{
					kind: KindNotificationEndpointSlack,
					resErr: testPkgResourceError{
						name:           "missing slack url",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointURL},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointSlack
metadata:
  name: slack_notification_endpoint
spec:
`,
					},
				},
				{
					kind: KindNotificationEndpointPagerDuty,
					resErr: testPkgResourceError{
						name:           "missing pager duty url",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointURL},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointPagerDuty
metadata:
  name: pager_duty_notification_endpoint
spec:
`,
					},
				},
				{
					kind: KindNotificationEndpointHTTP,
					resErr: testPkgResourceError{
						name:           "missing http url",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointURL},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointHTTP
metadata:
  name: http_none_auth_notification_endpoint
spec:
  type: none
  method: get
`,
					},
				},
				{
					kind: KindNotificationEndpointHTTP,
					resErr: testPkgResourceError{
						name:           "bad url",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointURL},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointHTTP
metadata:
  name: http_none_auth_notification_endpoint
spec:
  type: none
  method: get
  url: d_____-_8**(*https://www.examples.coms
`,
					},
				},
				{
					kind: KindNotificationEndpointHTTP,
					resErr: testPkgResourceError{
						name:           "missing http method",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointHTTPMethod},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointHTTP
metadata:
  name: http_none_auth_notification_endpoint
spec:
  type: none
  url:  https://www.example.com/endpoint/noneauth
`,
					},
				},
				{
					kind: KindNotificationEndpointHTTP,
					resErr: testPkgResourceError{
						name:           "invalid http method",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointHTTPMethod},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointHTTP
metadata:
  name: http_none_auth_notification_endpoint
spec:
  type: none
  description: http none auth desc
  method: GHOST
  url:  https://www.example.com/endpoint/noneauth
`,
					},
				},
				{
					kind: KindNotificationEndpointHTTP,
					resErr: testPkgResourceError{
						name:           "missing basic username",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointUsername},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointHTTP
metadata:
  name: http_basic_auth_notification_endpoint
spec:
  type: basic
  method: POST
  url:  https://www.example.com/endpoint/basicauth
  password: "secret password"
`,
					},
				},
				{
					kind: KindNotificationEndpointHTTP,
					resErr: testPkgResourceError{
						name:           "missing basic password",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointPassword},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointHTTP
metadata:
  name: http_basic_auth_notification_endpoint
spec:
  type: basic
  method: POST
  url:  https://www.example.com/endpoint/basicauth
  username: username
`,
					},
				},
				{
					kind: KindNotificationEndpointHTTP,
					resErr: testPkgResourceError{
						name:           "missing basic password and username",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointPassword, fieldNotificationEndpointUsername},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointHTTP
metadata:
  name: http_basic_auth_notification_endpoint
spec:
  description: http basic auth desc
  type: basic
  method: pOsT
  url:  https://www.example.com/endpoint/basicauth
`,
					},
				},
				{
					kind: KindNotificationEndpointHTTP,
					resErr: testPkgResourceError{
						name:           "missing bearer token",
						validationErrs: 1,
						valFields:      []string{fieldNotificationEndpointToken},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointHTTP
metadata:
  name: http_bearer_auth_notification_endpoint
spec:
  description: http bearer auth desc
  type: bearer
  method: puT
  url:  https://www.example.com/endpoint/bearerauth
`,
					},
				},
				{
					kind: KindNotificationEndpointHTTP,
					resErr: testPkgResourceError{
						name:           "invalid http type",
						validationErrs: 1,
						valFields:      []string{fieldType},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointHTTP
metadata:
  name: http_none_auth_notification_endpoint
spec:
  type: RANDOM WRONG TYPE
  description: http none auth desc
  method: get
  url:  https://www.example.com/endpoint/noneauth
`,
					},
				},
				{
					kind: KindNotificationEndpointSlack,
					resErr: testPkgResourceError{
						name:           "duplicate endpoints",
						validationErrs: 1,
						valFields:      []string{fieldName},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointSlack
metadata:
  name: slack_notification_endpoint
spec:
  url: https://hooks.slack.com/services/bip/piddy/boppidy
---
apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointSlack
metadata:
  name: slack_notification_endpoint
spec:
  url: https://hooks.slack.com/services/bip/piddy/boppidy
`,
					},
				},
				{
					kind: KindNotificationEndpointSlack,
					resErr: testPkgResourceError{
						name:           "invalid status",
						validationErrs: 1,
						valFields:      []string{fieldStatus},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationEndpointSlack
metadata:
  name: slack_notification_endpoint
spec:
  description: slack desc
  url: https://hooks.slack.com/services/bip/piddy/boppidy
  status: RANDO STATUS
`,
					},
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, tt.kind, tt.resErr)
			}
		})
	})

	t.Run("pkg with notification rules", func(t *testing.T) {
		t.Run("happy path", func(t *testing.T) {
			testfileRunner(t, "testdata/notification_rule", func(t *testing.T, pkg *Pkg) {
				sum := pkg.Summary()
				rules := sum.NotificationRules
				require.Len(t, rules, 1)

				rule := rules[0]
				assert.Equal(t, "rule_0", rule.Name)
				assert.Equal(t, "endpoint_0", rule.EndpointName)
				assert.Equal(t, "desc_0", rule.Description)
				assert.Equal(t, (10 * time.Minute).String(), rule.Every)
				assert.Equal(t, (30 * time.Second).String(), rule.Offset)
				expectedMsgTempl := "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
				assert.Equal(t, expectedMsgTempl, rule.MessageTemplate)
				assert.Equal(t, influxdb.Active, rule.Status)

				expectedStatusRules := []SummaryStatusRule{
					{CurrentLevel: "CRIT", PreviousLevel: "OK"},
					{CurrentLevel: "WARN"},
				}
				assert.Equal(t, expectedStatusRules, rule.StatusRules)

				expectedTagRules := []SummaryTagRule{
					{Key: "k1", Value: "v1", Operator: "equal"},
					{Key: "k1", Value: "v2", Operator: "equal"},
				}
				assert.Equal(t, expectedTagRules, rule.TagRules)

				require.Len(t, sum.Labels, 1)
				require.Len(t, rule.LabelAssociations, 1)
			})
		})

		t.Run("handles bad config", func(t *testing.T) {
			tests := []struct {
				kind   Kind
				resErr testPkgResourceError
			}{
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "missing name",
						validationErrs: 1,
						valFields:      []string{fieldName},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
spec:
  endpointName: endpoint_0
  every: 10m
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
  statusRules:
    - currentLevel: WARN
`,
					},
				},
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "missing endpoint name",
						validationErrs: 1,
						valFields:      []string{fieldNotificationRuleEndpointName},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
  name: rule_0
spec:
  every: 10m
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
  statusRules:
    - currentLevel: WARN
`,
					},
				},
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "missing every",
						validationErrs: 1,
						valFields:      []string{fieldEvery},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
  name: rule_0
spec:
  endpointName: endpoint_0
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
  statusRules:
    - currentLevel: WARN
`,
					},
				},
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "missing status rules",
						validationErrs: 1,
						valFields:      []string{fieldNotificationRuleStatusRules},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
  name: rule_0
spec:
  every: 10m
  endpointName: 10m
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
`,
					},
				},
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "bad current status rule level",
						validationErrs: 1,
						valFields:      []string{fieldNotificationRuleStatusRules},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
  name: rule_0
spec:
  every: 10m
  endpointName: 10m
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
  statusRules:
    - currentLevel: WRONGO
`,
					},
				},
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "bad previous status rule level",
						validationErrs: 1,
						valFields:      []string{fieldNotificationRuleStatusRules},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
  name: rule_0
spec:
  endpointName: endpoint_0
  every: 10m
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
  statusRules:
    - currentLevel: CRIT
      previousLevel: WRONG
`,
					},
				},
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "bad tag rule operator",
						validationErrs: 1,
						valFields:      []string{fieldNotificationRuleTagRules},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
  name: rule_0
spec:
  endpointName: endpoint_0
  every: 10m
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
  statusRules:
    - currentLevel: WARN
  tagRules:
    - key: k1
      value: v2
      operator: WRONG
`,
					},
				},
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "bad status provided",
						validationErrs: 1,
						valFields:      []string{fieldStatus},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
  name: rule_0
spec:
  endpointName: endpoint_0
  every: 10m
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
  status: RANDO STATUS
  statusRules:
    - currentLevel: WARN
`,
					},
				},
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "label association does not exist",
						validationErrs: 1,
						valFields:      []string{fieldAssociations},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
  name: rule_0
spec:
  endpointName: endpoint_0
  every: 10m
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
  statusRules:
    - currentLevel: WARN
  associations:
    - kind: Label
      name: label_1
`,
					},
				},
				{
					kind: KindNotificationRule,
					resErr: testPkgResourceError{
						name:           "label association dupe",
						validationErrs: 1,
						valFields:      []string{fieldAssociations},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: label_1
---
apiVersion: influxdata.com/v2alpha1
kind: NotificationRule
metadata:
  name: rule_0
spec:
  endpointName: endpoint_0
  every: 10m
  messageTemplate: "Notification Rule: ${ r._notification_rule_name } triggered by check: ${ r._check_name }: ${ r._message }"
  statusRules:
    - currentLevel: WARN
  associations:
    - kind: Label
      name: label_1
    - kind: Label
      name: label_1
`,
					},
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, tt.kind, tt.resErr)
			}
		})
	})

	t.Run("pkg with tasks", func(t *testing.T) {
		t.Run("happy path", func(t *testing.T) {
			testfileRunner(t, "testdata/tasks", func(t *testing.T, pkg *Pkg) {
				sum := pkg.Summary()
				tasks := sum.Tasks
				require.Len(t, tasks, 2)

				baseEqual := func(t *testing.T, i int, status influxdb.Status, actual SummaryTask) {
					t.Helper()

					assert.Equal(t, "task_"+strconv.Itoa(i), actual.Name)
					assert.Equal(t, "desc_"+strconv.Itoa(i), actual.Description)
					assert.Equal(t, status, actual.Status)

					expectedQuery := "from(bucket: \"rucket_1\")\n  |> range(start: -5d, stop: -1h)\n  |> filter(fn: (r) => r._measurement == \"cpu\")\n  |> filter(fn: (r) => r._field == \"usage_idle\")\n  |> aggregateWindow(every: 1m, fn: mean)\n  |> yield(name: \"mean\")"
					assert.Equal(t, expectedQuery, actual.Query)

					require.Len(t, actual.LabelAssociations, 1)
					assert.Equal(t, "label_1", actual.LabelAssociations[0].Name)
				}

				require.Len(t, sum.Labels, 1)

				task1 := tasks[0]
				baseEqual(t, 0, influxdb.Inactive, task1)
				assert.Equal(t, (10 * time.Minute).String(), task1.Every)
				assert.Equal(t, (15 * time.Second).String(), task1.Offset)

				task2 := tasks[1]
				baseEqual(t, 1, influxdb.Active, task2)
				assert.Equal(t, "15 * * * *", task2.Cron)
			})
		})

		t.Run("handles bad config", func(t *testing.T) {
			tests := []struct {
				kind   Kind
				resErr testPkgResourceError
			}{
				{
					kind: KindTask,
					resErr: testPkgResourceError{
						name:           "missing name",
						validationErrs: 1,
						valFields:      []string{fieldName},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Task
metadata:
spec:
  description: desc_1
  cron: 15 * * * *
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
`,
					},
				},
				{
					kind: KindTask,
					resErr: testPkgResourceError{
						name:           "invalid status",
						validationErrs: 1,
						valFields:      []string{fieldStatus},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Task
metadata:
  name: task_0
spec:
  cron: 15 * * * *
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  status: RANDO WRONGO
`,
					},
				},
				{
					kind: KindTask,
					resErr: testPkgResourceError{
						name:           "missing query",
						validationErrs: 1,
						valFields:      []string{fieldQuery},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Task
metadata:
  name: task_0
spec:
  description: desc_0
  every: 10m
  offset: 15s
`,
					},
				},
				{
					kind: KindTask,
					resErr: testPkgResourceError{
						name:           "missing every and cron fields",
						validationErrs: 1,
						valFields:      []string{fieldEvery, fieldTaskCron},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Task
metadata:
  name: task_0
spec:
  description: desc_0
  offset: 15s
`,
					},
				},
				{
					kind: KindTask,
					resErr: testPkgResourceError{
						name:           "invalid association",
						validationErrs: 1,
						valFields:      []string{fieldAssociations},
						pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Task
metadata:
  name: task_1
spec:
  cron: 15 * * * *
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  associations:
    - kind: Label
      name: label_1
`,
					},
				},
				{
					kind: KindTask,
					resErr: testPkgResourceError{
						name:           "duplicate association",
						validationErrs: 1,
						valFields:      []string{fieldAssociations},
						pkgStr: `---
apiVersion: influxdata.com/v2alpha1
kind: Label
metadata:
  name: label_1
---
apiVersion: influxdata.com/v2alpha1
kind: Task
metadata:
  name: task_0
spec:
  every: 10m
  offset: 15s
  query:  >
    from(bucket: "rucket_1") |> yield(name: "mean")
  status: inactive
  associations:
    - kind: Label
      name: label_1
    - kind: Label
      name: label_1
`,
					},
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, tt.kind, tt.resErr)
			}
		})
	})

	t.Run("pkg with telegraf and label associations", func(t *testing.T) {
		t.Run("with valid fields", func(t *testing.T) {
			testfileRunner(t, "testdata/telegraf", func(t *testing.T, pkg *Pkg) {
				sum := pkg.Summary()
				require.Len(t, sum.TelegrafConfigs, 1)

				actual := sum.TelegrafConfigs[0]
				assert.Equal(t, "first_tele_config", actual.TelegrafConfig.Name)
				assert.Equal(t, "desc", actual.TelegrafConfig.Description)

				require.Len(t, actual.LabelAssociations, 1)
				assert.Equal(t, "label_1", actual.LabelAssociations[0].Name)

				require.Len(t, sum.LabelMappings, 1)
				expectedMapping := SummaryLabelMapping{
					ResourceName: "first_tele_config",
					LabelName:    "label_1",
					ResourceType: influxdb.TelegrafsResourceType,
				}
				assert.Equal(t, expectedMapping, sum.LabelMappings[0])
			})
		})

		t.Run("handles bad config", func(t *testing.T) {
			tests := []testPkgResourceError{
				{
					name:           "config missing",
					validationErrs: 1,
					valFields:      []string{"config"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Telegraf
metadata:
  name: first_tele_config
spec:
`,
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, KindTelegraf, tt)
			}
		})
	})

	t.Run("pkg with a variable", func(t *testing.T) {
		t.Run("with valid fields should produce summary", func(t *testing.T) {
			testfileRunner(t, "testdata/variables", func(t *testing.T, pkg *Pkg) {
				sum := pkg.Summary()

				require.Len(t, sum.Variables, 4)

				varEquals := func(t *testing.T, name, vType string, vals interface{}, v SummaryVariable) {
					t.Helper()

					assert.Equal(t, name, v.Name)
					assert.Equal(t, name+" desc", v.Description)
					require.NotNil(t, v.Arguments)
					assert.Equal(t, vType, v.Arguments.Type)
					assert.Equal(t, vals, v.Arguments.Values)
				}

				// validates we support all known variable types
				varEquals(t,
					"var_const_3",
					"constant",
					influxdb.VariableConstantValues([]string{"first val"}),
					sum.Variables[0],
				)

				varEquals(t,
					"var_map_4",
					"map",
					influxdb.VariableMapValues{"k1": "v1"},
					sum.Variables[1],
				)

				varEquals(t,
					"var_query_1",
					"query",
					influxdb.VariableQueryValues{
						Query:    `buckets()  |> filter(fn: (r) => r.name !~ /^_/)  |> rename(columns: {name: "_value"})  |> keep(columns: ["_value"])`,
						Language: "flux",
					},
					sum.Variables[2],
				)

				varEquals(t,
					"var_query_2",
					"query",
					influxdb.VariableQueryValues{
						Query:    "an influxql query of sorts",
						Language: "influxql",
					},
					sum.Variables[3],
				)
			})
		})

		t.Run("handles bad config", func(t *testing.T) {
			tests := []testPkgResourceError{
				{
					name:           "name missing",
					validationErrs: 1,
					valFields:      []string{"name"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Variable
metadata:
spec:
  description: var_map_4 desc
  type: map
  values:
    k1: v1
`,
				},
				{
					name:           "map var missing values",
					validationErrs: 1,
					valFields:      []string{"values"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Variable
metadata:
  name:  var_map_4
spec:
  description: var_map_4 desc
  type: map
`,
				},
				{
					name:           "const var missing values",
					validationErrs: 1,
					valFields:      []string{"values"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Variable
metadata:
  name:  var_const_3
spec:
  description: var_const_3 desc
  type: constant
`,
				},
				{
					name:           "query var missing query",
					validationErrs: 1,
					valFields:      []string{"query"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Variable
metadata:
  name:  var_query_2
spec:
  description: var_query_2 desc
  type: query
  language: influxql
`,
				},
				{
					name:           "query var missing query language",
					validationErrs: 1,
					valFields:      []string{"language"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Variable
metadata:
  name:  var_query_2
spec:
  description: var_query_2 desc
  type: query
  query: an influxql query of sorts
`,
				},
				{
					name:           "query var provides incorrect query language",
					validationErrs: 1,
					valFields:      []string{"language"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Variable
metadata:
  name:  var_query_2
spec:
  description: var_query_2 desc
  type: query
  query: an influxql query of sorts
  language: wrong Language
`,
				},
				{
					name:           "duplicate var names",
					validationErrs: 1,
					valFields:      []string{"name"},
					pkgStr: `apiVersion: influxdata.com/v2alpha1
kind: Variable
metadata:
  name:  var_query_2
spec:
  description: var_query_2 desc
  type: query
  query: an influxql query of sorts
  language: influxql
---
apiVersion: influxdata.com/v2alpha1
kind: Variable
metadata:
  name:  var_query_2
spec:
  description: var_query_2 desc
  type: query
  query: an influxql query of sorts
  language: influxql
`,
				},
			}

			for _, tt := range tests {
				testPkgErrors(t, KindVariable, tt)
			}
		})
	})

	t.Run("pkg with variable and labels associated", func(t *testing.T) {
		testfileRunner(t, "testdata/variable_associates_label.yml", func(t *testing.T, pkg *Pkg) {
			sum := pkg.Summary()
			require.Len(t, sum.Labels, 1)

			vars := sum.Variables
			require.Len(t, vars, 1)

			expectedLabelMappings := []struct {
				varName string
				labels  []string
			}{
				{
					varName: "var_1",
					labels:  []string{"label_1"},
				},
			}
			for i, expected := range expectedLabelMappings {
				v := vars[i]
				require.Len(t, v.LabelAssociations, len(expected.labels))

				for j, label := range expected.labels {
					assert.Equal(t, label, v.LabelAssociations[j].Name)
				}
			}

			expectedMappings := []SummaryLabelMapping{
				{
					ResourceName: "var_1",
					LabelName:    "label_1",
				},
			}

			require.Len(t, sum.LabelMappings, len(expectedMappings))
			for i, expected := range expectedMappings {
				expected.ResourceType = influxdb.VariablesResourceType
				assert.Equal(t, expected, sum.LabelMappings[i])
			}
		})
	})

	t.Run("referencing secrets", func(t *testing.T) {
		testfileRunner(t, "testdata/notification_endpoint_secrets.yml", func(t *testing.T, pkg *Pkg) {
			sum := pkg.Summary()

			endpoints := sum.NotificationEndpoints
			require.Len(t, endpoints, 1)

			expected := &endpoint.PagerDuty{
				Base: endpoint.Base{
					Name:   "pager_duty_notification_endpoint",
					Status: influxdb.TaskStatusActive,
				},
				ClientURL:  "http://localhost:8080/orgs/7167eb6719fa34e5/alert-history",
				RoutingKey: influxdb.SecretField{Key: "-routing-key", Value: strPtr("not emtpy")},
			}
			actual, ok := endpoints[0].NotificationEndpoint.(*endpoint.PagerDuty)
			require.True(t, ok)
			assert.Equal(t, expected.Base.Name, actual.Name)
			require.Nil(t, actual.RoutingKey.Value)
			assert.Equal(t, "routing-key", actual.RoutingKey.Key)

			_, ok = pkg.mSecrets["routing-key"]
			require.True(t, ok)
		})
	})

	t.Run("jsonnet support", func(t *testing.T) {
		pkg := validParsedPkg(t, "testdata/bucket_associates_labels.jsonnet", EncodingJsonnet, baseAsserts{
			version:     "0.1.0",
			kind:        KindPackage,
			description: "pack description",
			metaName:    "pkg_name",
			metaVersion: "1",
		})

		sum := pkg.Summary()

		labels := []SummaryLabel{
			{
				Name: "label_1",
				Properties: struct {
					Color       string `json:"color"`
					Description string `json:"description"`
				}{Color: "#eee888", Description: "desc_1"},
			},
		}
		assert.Equal(t, labels, sum.Labels)

		bkts := []SummaryBucket{
			{
				Name:              "rucket_1",
				Description:       "desc_1",
				RetentionPeriod:   10000 * time.Second,
				LabelAssociations: labels,
			},
			{
				Name:              "rucket_2",
				Description:       "desc_2",
				RetentionPeriod:   20000 * time.Second,
				LabelAssociations: labels,
			},
			{
				Name:              "rucket_3",
				Description:       "desc_3",
				RetentionPeriod:   30000 * time.Second,
				LabelAssociations: labels,
			},
		}
		assert.Equal(t, bkts, sum.Buckets)
	})
}

func Test_IsParseError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "base case",
			err:      &parseErr{},
			expected: true,
		},
		{
			name: "wrapped by influxdb error",
			err: &influxdb.Error{
				Err: &parseErr{},
			},
			expected: true,
		},
		{
			name: "deeply nested in influxdb error",
			err: &influxdb.Error{
				Err: &influxdb.Error{
					Err: &influxdb.Error{
						Err: &influxdb.Error{
							Err: &parseErr{},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "influxdb error without nested parse err",
			err: &influxdb.Error{
				Err: errors.New("nope"),
			},
			expected: false,
		},
		{
			name:     "plain error",
			err:      errors.New("nope"),
			expected: false,
		},
	}

	for _, tt := range tests {
		fn := func(t *testing.T) {
			isParseErr := IsParseErr(tt.err)
			assert.Equal(t, tt.expected, isParseErr)
		}
		t.Run(tt.name, fn)
	}
}

func Test_PkgValidationErr(t *testing.T) {
	iPtr := func(i int) *int {
		return &i
	}

	compIntSlcs := func(t *testing.T, expected []int, actuals []*int) {
		t.Helper()

		if len(expected) >= len(actuals) {
			require.FailNow(t, "expected array is larger than actuals")
		}

		for i, actual := range actuals {
			if i == len(expected) {
				assert.Nil(t, actual)
				continue
			}
			assert.Equal(t, expected[i], *actual)
		}
	}

	pErr := &parseErr{
		Resources: []resourceErr{
			{
				Kind: KindDashboard.String(),
				Idx:  intPtr(0),
				ValidationErrs: []validationErr{
					{
						Field: "charts",
						Index: iPtr(1),
						Nested: []validationErr{
							{
								Field: "colors",
								Index: iPtr(0),
								Nested: []validationErr{
									{
										Field: "hex",
										Msg:   "hex value required",
									},
								},
							},
							{
								Field: "kind",
								Msg:   "chart kind must be provided",
							},
						},
					},
				},
			},
		},
	}

	errs := pErr.ValidationErrs()
	require.Len(t, errs, 2)
	assert.Equal(t, KindDashboard.String(), errs[0].Kind)
	assert.Equal(t, []string{"kinds", "charts", "colors", "hex"}, errs[0].Fields)
	compIntSlcs(t, []int{0, 1, 0}, errs[0].Indexes)
	assert.Equal(t, "hex value required", errs[0].Reason)

	assert.Equal(t, KindDashboard.String(), errs[1].Kind)
	assert.Equal(t, []string{"kinds", "charts", "kind"}, errs[1].Fields)
	compIntSlcs(t, []int{0, 1}, errs[1].Indexes)
	assert.Equal(t, "chart kind must be provided", errs[1].Reason)
}

type testPkgResourceError struct {
	name           string
	encoding       Encoding
	pkgStr         string
	resourceErrs   int
	validationErrs int
	valFields      []string
	assErrs        int
	assIdxs        []int
}

// defaults to yaml encoding if encoding not provided
// defaults num resources to 1 if resource errs not provided.
func testPkgErrors(t *testing.T, k Kind, tt testPkgResourceError) {
	t.Helper()
	encoding := EncodingYAML
	if tt.encoding != EncodingUnknown {
		encoding = tt.encoding
	}

	resErrs := 1
	if tt.resourceErrs > 0 {
		resErrs = tt.resourceErrs
	}

	fn := func(t *testing.T) {
		t.Helper()

		_, err := Parse(encoding, FromString(tt.pkgStr))
		require.Error(t, err)

		require.True(t, IsParseErr(err), err)

		pErr := err.(*parseErr)
		require.Len(t, pErr.Resources, resErrs)

		resErr := pErr.Resources[0]
		assert.Equal(t, k.String(), resErr.Kind)

		for i, vFail := range resErr.ValidationErrs {
			if len(tt.valFields) == i {
				break
			}
			expectedField := tt.valFields[i]
			findErr(t, expectedField, vFail)
		}

		if tt.assErrs == 0 {
			return
		}

		assFails := pErr.Resources[0].AssociationErrs
		for i, assFail := range assFails {
			if len(tt.valFields) == i {
				break
			}
			expectedField := tt.valFields[i]
			findErr(t, expectedField, assFail)
		}
	}
	t.Run(tt.name, fn)
}

func findErr(t *testing.T, expectedField string, vErr validationErr) validationErr {
	t.Helper()

	fields := strings.Split(expectedField, ".")
	if len(fields) == 1 {
		require.Equal(t, expectedField, vErr.Field)
		return vErr
	}

	currentFieldName, idx := nextField(t, fields[0])
	if idx > -1 {
		require.NotNil(t, vErr.Index)
		require.Equal(t, idx, *vErr.Index)
	}
	require.Equal(t, currentFieldName, vErr.Field)

	next := strings.Join(fields[1:], ".")
	nestedField, _ := nextField(t, next)
	for _, n := range vErr.Nested {
		if n.Field == nestedField {
			return findErr(t, next, n)
		}
	}
	assert.Fail(t, "did not find field: "+expectedField)

	return vErr
}

func nextField(t *testing.T, field string) (string, int) {
	t.Helper()

	fields := strings.Split(field, ".")
	if len(fields) == 1 && !strings.HasSuffix(fields[0], "]") {
		return field, -1
	}
	parts := strings.Split(fields[0], "[")
	if len(parts) == 1 {
		return "", 0
	}
	fieldName := parts[0]

	if strIdx := strings.Index(parts[1], "]"); strIdx > -1 {
		idx, err := strconv.Atoi(parts[1][:strIdx])
		require.NoError(t, err)
		return fieldName, idx
	}
	return "", -1
}

type baseAsserts struct {
	version     string
	kind        Kind
	description string
	metaName    string
	metaVersion string
}

func validParsedPkg(t *testing.T, path string, encoding Encoding, expected baseAsserts) *Pkg {
	t.Helper()

	pkg, err := Parse(encoding, FromFile(path))
	require.NoError(t, err)

	for _, k := range pkg.Objects {
		require.Equal(t, APIVersion, k.APIVersion)
	}

	require.True(t, pkg.isParsed)
	return pkg
}

func testfileRunner(t *testing.T, path string, testFn func(t *testing.T, pkg *Pkg)) {
	t.Helper()

	tests := []struct {
		name      string
		extension string
		encoding  Encoding
	}{
		{
			name:      "yaml",
			extension: ".yml",
			encoding:  EncodingYAML,
		},
		{
			name:      "json",
			extension: ".json",
			encoding:  EncodingJSON,
		},
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".yml":
		tests = tests[:1]
	case ".json":
		tests = tests[1:]
	}

	path = strings.TrimSuffix(path, ext)

	for _, tt := range tests {
		fn := func(t *testing.T) {
			t.Helper()

			pkg := validParsedPkg(t, path+tt.extension, tt.encoding, baseAsserts{
				version:     "0.1.0",
				kind:        KindPackage,
				description: "pack description",
				metaName:    "pkg_name",
				metaVersion: "1",
			})
			if testFn != nil {
				testFn(t, pkg)
			}
		}
		t.Run(tt.name, fn)
	}
}

type labelMapping struct {
	labelName string
	resName   string
	resType   influxdb.ResourceType
}

func containsLabelMappings(t *testing.T, labelMappings []SummaryLabelMapping, matches ...labelMapping) {
	t.Helper()

	for _, expected := range matches {
		expectedMapping := SummaryLabelMapping{
			ResourceName: expected.resName,
			LabelName:    expected.labelName,
			ResourceType: expected.resType,
		}
		assert.Contains(t, labelMappings, expectedMapping)
	}
}

func strPtr(s string) *string {
	return &s
}

func mustDuration(t *testing.T, d time.Duration) *notification.Duration {
	t.Helper()
	dur, err := notification.FromTimeDuration(d)
	require.NoError(t, err)
	return &dur
}
