package launcher_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/cmd/influxd/launcher"
	"github.com/influxdata/influxdb/mock"
	"github.com/influxdata/influxdb/pkger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLauncher_Pkger(t *testing.T) {
	l := launcher.RunTestLauncherOrFail(t, ctx)
	l.SetupOrFail(t)
	defer l.ShutdownOrFail(t, ctx)

	svc := l.PkgerService(t)

	t.Run("create a new package", func(t *testing.T) {
		newPkg, err := svc.CreatePkg(timedCtx(time.Second),
			pkger.CreateWithMetadata(pkger.Metadata{
				Description: "new desc",
				Name:        "new name",
				Version:     "v1.0.0",
			}),
		)
		require.NoError(t, err)

		assert.Equal(t, "new name", newPkg.Metadata.Name)
		assert.Equal(t, "new desc", newPkg.Metadata.Description)
		assert.Equal(t, "v1.0.0", newPkg.Metadata.Version)
	})

	t.Run("errors incurred during application of package rolls back to state before package", func(t *testing.T) {
		svc := pkger.NewService(
			pkger.WithBucketSVC(l.BucketService(t)),
			pkger.WithDashboardSVC(l.DashboardService(t)),
			pkger.WithLabelSVC(&fakeLabelSVC{
				LabelService: l.LabelService(t),
				killCount:    2, // hits error on 3rd attempt at creating a mapping
			}),
			pkger.WithNoticationEndpointSVC(l.NotificationEndpointService(t)),
			pkger.WithTelegrafSVC(l.TelegrafService(t)),
			pkger.WithVariableSVC(l.VariableService(t)),
		)

		_, err := svc.Apply(ctx, l.Org.ID, l.User.ID, newPkg(t))
		require.Error(t, err)

		bkts, _, err := l.BucketService(t).FindBuckets(ctx, influxdb.BucketFilter{OrganizationID: &l.Org.ID})
		require.NoError(t, err)
		for _, b := range bkts {
			if influxdb.BucketTypeSystem == b.Type {
				continue
			}
			// verify system buckets and org bucket are the buckets available
			assert.Equal(t, l.Bucket.Name, b.Name)
		}

		labels, err := l.LabelService(t).FindLabels(ctx, influxdb.LabelFilter{OrgID: &l.Org.ID})
		require.NoError(t, err)
		assert.Empty(t, labels)

		dashs, _, err := l.DashboardService(t).FindDashboards(ctx, influxdb.DashboardFilter{
			OrganizationID: &l.Org.ID,
		}, influxdb.DefaultDashboardFindOptions)
		require.NoError(t, err)
		assert.Empty(t, dashs)

		endpoints, _, err := l.NotificationEndpointService(t).FindNotificationEndpoints(ctx, influxdb.NotificationEndpointFilter{
			OrgID: &l.Org.ID,
		})
		require.NoError(t, err)
		assert.Empty(t, endpoints)

		teles, _, err := l.TelegrafService(t).FindTelegrafConfigs(ctx, influxdb.TelegrafConfigFilter{
			OrgID: &l.Org.ID,
		})
		require.NoError(t, err)
		assert.Empty(t, teles)

		vars, err := l.VariableService(t).FindVariables(ctx, influxdb.VariableFilter{OrganizationID: &l.Org.ID})
		require.NoError(t, err)
		assert.Empty(t, vars)
	})

	hasLabelAssociations := func(t *testing.T, associations []pkger.SummaryLabel, numAss int, expectedNames ...string) {
		t.Helper()

		require.Len(t, associations, numAss)

		hasAss := func(t *testing.T, expected string) {
			t.Helper()
			for _, ass := range associations {
				if ass.Name == expected {
					return
				}
			}
			require.FailNow(t, "did not find expected association: "+expected)
		}

		for _, expected := range expectedNames {
			hasAss(t, expected)
		}
	}

	hasMapping := func(t *testing.T, actuals []pkger.SummaryLabelMapping, expected pkger.SummaryLabelMapping) {
		t.Helper()

		for _, actual := range actuals {
			if actual == expected {
				return
			}
		}
		require.FailNowf(t, "did not find expected mapping", "expected: %v", expected)
	}

	t.Run("dry run a package with no existing resources", func(t *testing.T) {
		sum, diff, err := svc.DryRun(ctx, l.Org.ID, l.User.ID, newPkg(t))
		require.NoError(t, err)

		diffBkts := diff.Buckets
		require.Len(t, diffBkts, 1)
		assert.True(t, diffBkts[0].IsNew())

		diffLabels := diff.Labels
		require.Len(t, diffLabels, 1)
		assert.True(t, diffLabels[0].IsNew())

		diffVars := diff.Variables
		require.Len(t, diffVars, 1)
		assert.True(t, diffVars[0].IsNew())

		require.Len(t, diff.Dashboards, 1)
		require.Len(t, diff.NotificationEndpoints, 1)
		require.Len(t, diff.Telegrafs, 1)

		labels := sum.Labels
		require.Len(t, labels, 1)
		assert.Equal(t, "label_1", labels[0].Name)

		bkts := sum.Buckets
		require.Len(t, bkts, 1)
		assert.Equal(t, "rucket_1", bkts[0].Name)
		hasLabelAssociations(t, bkts[0].LabelAssociations, 1, "label_1")

		dashs := sum.Dashboards
		require.Len(t, dashs, 1)
		assert.Equal(t, "dash_1", dashs[0].Name)
		assert.Equal(t, "desc1", dashs[0].Description)
		hasLabelAssociations(t, dashs[0].LabelAssociations, 1, "label_1")

		endpoints := sum.NotificationEndpoints
		require.Len(t, endpoints, 1)
		assert.Equal(t, "http_none_auth_notification_endpoint", endpoints[0].NotificationEndpoint.GetName())
		assert.Equal(t, "http none auth desc", endpoints[0].NotificationEndpoint.GetDescription())
		hasLabelAssociations(t, endpoints[0].LabelAssociations, 1, "label_1")

		teles := sum.TelegrafConfigs
		require.Len(t, teles, 1)
		assert.Equal(t, "first_tele_config", teles[0].TelegrafConfig.Name)
		assert.Equal(t, "desc", teles[0].TelegrafConfig.Description)
		hasLabelAssociations(t, teles[0].LabelAssociations, 1, "label_1")

		vars := sum.Variables
		require.Len(t, vars, 1)
		assert.Equal(t, "var_query_1", vars[0].Name)
		hasLabelAssociations(t, vars[0].LabelAssociations, 1, "label_1")
		varArgs := vars[0].Arguments
		require.NotNil(t, varArgs)
		assert.Equal(t, "query", varArgs.Type)
		assert.Equal(t, influxdb.VariableQueryValues{
			Query:    "buckets()  |> filter(fn: (r) => r.name !~ /^_/)  |> rename(columns: {name: \"_value\"})  |> keep(columns: [\"_value\"])",
			Language: "flux",
		}, varArgs.Values)
	})

	t.Run("apply a package of all new resources", func(t *testing.T) {
		// this initial test is also setup for the sub tests
		sum1, err := svc.Apply(timedCtx(5*time.Second), l.Org.ID, l.User.ID, newPkg(t))
		require.NoError(t, err)

		labels := sum1.Labels
		require.Len(t, labels, 1)
		assert.NotZero(t, labels[0].ID)
		assert.Equal(t, "label_1", labels[0].Name)

		bkts := sum1.Buckets
		require.Len(t, bkts, 1)
		assert.NotZero(t, bkts[0].ID)
		assert.Equal(t, "rucket_1", bkts[0].Name)
		hasLabelAssociations(t, bkts[0].LabelAssociations, 1, "label_1")

		dashs := sum1.Dashboards
		require.Len(t, dashs, 1)
		assert.NotZero(t, dashs[0].ID)
		assert.Equal(t, "dash_1", dashs[0].Name)
		assert.Equal(t, "desc1", dashs[0].Description)
		hasLabelAssociations(t, dashs[0].LabelAssociations, 1, "label_1")
		require.Len(t, dashs[0].Charts, 1)
		assert.Equal(t, influxdb.ViewPropertyTypeSingleStat, dashs[0].Charts[0].Properties.GetType())

		endpoints := sum1.NotificationEndpoints
		require.Len(t, endpoints, 1)
		assert.NotZero(t, endpoints[0].NotificationEndpoint.GetID())
		assert.Equal(t, "http_none_auth_notification_endpoint", endpoints[0].NotificationEndpoint.GetName())
		assert.Equal(t, "http none auth desc", endpoints[0].NotificationEndpoint.GetDescription())
		assert.Equal(t, influxdb.TaskStatusInactive, string(endpoints[0].NotificationEndpoint.GetStatus()))
		hasLabelAssociations(t, endpoints[0].LabelAssociations, 1, "label_1")

		teles := sum1.TelegrafConfigs
		require.Len(t, teles, 1)
		assert.NotZero(t, teles[0].TelegrafConfig.ID)
		assert.Equal(t, l.Org.ID, teles[0].TelegrafConfig.OrgID)
		assert.Equal(t, "first_tele_config", teles[0].TelegrafConfig.Name)
		assert.Equal(t, "desc", teles[0].TelegrafConfig.Description)
		assert.Len(t, teles[0].TelegrafConfig.Plugins, 2)

		vars := sum1.Variables
		require.Len(t, vars, 1)
		assert.NotZero(t, vars[0].ID)
		assert.Equal(t, "var_query_1", vars[0].Name)
		hasLabelAssociations(t, vars[0].LabelAssociations, 1, "label_1")
		varArgs := vars[0].Arguments
		require.NotNil(t, varArgs)
		assert.Equal(t, "query", varArgs.Type)
		assert.Equal(t, influxdb.VariableQueryValues{
			Query:    "buckets()  |> filter(fn: (r) => r.name !~ /^_/)  |> rename(columns: {name: \"_value\"})  |> keep(columns: [\"_value\"])",
			Language: "flux",
		}, varArgs.Values)

		newSumMapping := func(id pkger.SafeID, name string, rt influxdb.ResourceType) pkger.SummaryLabelMapping {
			return pkger.SummaryLabelMapping{
				ResourceName: name,
				LabelName:    labels[0].Name,
				LabelID:      labels[0].ID,
				ResourceID:   pkger.SafeID(id),
				ResourceType: rt,
			}
		}

		mappings := sum1.LabelMappings
		require.Len(t, mappings, 5)
		hasMapping(t, mappings, newSumMapping(bkts[0].ID, bkts[0].Name, influxdb.BucketsResourceType))
		hasMapping(t, mappings, newSumMapping(dashs[0].ID, dashs[0].Name, influxdb.DashboardsResourceType))
		hasMapping(t, mappings, newSumMapping(vars[0].ID, vars[0].Name, influxdb.VariablesResourceType))
		hasMapping(t, mappings, newSumMapping(pkger.SafeID(teles[0].TelegrafConfig.ID), teles[0].TelegrafConfig.Name, influxdb.TelegrafsResourceType))

		t.Run("pkg with same bkt-var-label does nto create new resources for them", func(t *testing.T) {
			// validate the new package doesn't create new resources for bkts/labels/vars
			// since names collide.
			sum2, err := svc.Apply(timedCtx(5*time.Second), l.Org.ID, l.User.ID, newPkg(t))
			require.NoError(t, err)

			require.Equal(t, sum1.Buckets, sum2.Buckets)
			require.Equal(t, sum1.Labels, sum2.Labels)
			require.Equal(t, sum1.NotificationEndpoints, sum2.NotificationEndpoints)
			require.Equal(t, sum1.Variables, sum2.Variables)

			// dashboards should be new
			require.NotEqual(t, sum1.Dashboards, sum2.Dashboards)
		})

		t.Run("referenced secret values provided do not create new secrets", func(t *testing.T) {
			applyPkgStr := func(t *testing.T, pkgStr string) pkger.Summary {
				t.Helper()
				pkg, err := pkger.Parse(pkger.EncodingYAML, pkger.FromString(pkgStr))
				require.NoError(t, err)

				sum, err := svc.Apply(ctx, l.Org.ID, l.User.ID, pkg)
				require.NoError(t, err)
				return sum
			}

			const pkgWithSecretRaw = `apiVersion: 0.1.0
kind: Package
meta:
  pkgName:      pkg_name
  pkgVersion:   1
  description:  pack description
spec:
  resources:
    - kind: Notification_Endpoint_Pager_Duty
      name: pager_duty_notification_endpoint
      url:  http://localhost:8080/orgs/7167eb6719fa34e5/alert-history
      routingKey: secret-sauce
`
			secretSum := applyPkgStr(t, pkgWithSecretRaw)
			require.Len(t, secretSum.NotificationEndpoints, 1)

			id := secretSum.NotificationEndpoints[0].NotificationEndpoint.GetID()
			expected := influxdb.SecretField{
				Key: id.String() + "-routing-key",
			}
			secrets := secretSum.NotificationEndpoints[0].NotificationEndpoint.SecretFields()
			require.Len(t, secrets, 1)
			assert.Equal(t, expected, secrets[0])

			const pkgWithSecretRef = `apiVersion: 0.1.0
kind: Package
meta:
  pkgName:      pkg_name
  pkgVersion:   1
  description:  pack description
spec:
  resources:
    - kind: Notification_Endpoint_Pager_Duty
      name: pager_duty_notification_endpoint
      url:  http://localhost:8080/orgs/7167eb6719fa34e5/alert-history
      routingKey:
        secretRef:
          key: %s-routing-key
`
			secretSum = applyPkgStr(t, fmt.Sprintf(pkgWithSecretRef, id.String()))
			require.Len(t, secretSum.NotificationEndpoints, 1)

			expected = influxdb.SecretField{
				Key: id.String() + "-routing-key",
			}
			secrets = secretSum.NotificationEndpoints[0].NotificationEndpoint.SecretFields()
			require.Len(t, secrets, 1)
			assert.Equal(t, expected, secrets[0])
		})

		t.Run("exporting resources with existing ids should return a valid pkg", func(t *testing.T) {
			resToClone := []pkger.ResourceToClone{
				{
					Kind: pkger.KindBucket,
					ID:   influxdb.ID(bkts[0].ID),
				},
				{
					Kind: pkger.KindDashboard,
					ID:   influxdb.ID(dashs[0].ID),
				},
				{
					Kind: pkger.KindLabel,
					ID:   influxdb.ID(labels[0].ID),
				},
				{
					Kind: pkger.KindNotificationEndpoint,
					ID:   endpoints[0].NotificationEndpoint.GetID(),
				},
				{
					Kind: pkger.KindTelegraf,
					ID:   teles[0].TelegrafConfig.ID,
				},
			}

			resWithNewName := []pkger.ResourceToClone{
				{
					Kind: pkger.KindVariable,
					Name: "new name",
					ID:   influxdb.ID(vars[0].ID),
				},
			}

			newPkg, err := svc.CreatePkg(timedCtx(2*time.Second),
				pkger.CreateWithMetadata(pkger.Metadata{
					Description: "newest desc",
					Name:        "newest name",
					Version:     "v1.0.1",
				}),
				pkger.CreateWithExistingResources(append(resToClone, resWithNewName...)...),
			)
			require.NoError(t, err)

			assert.Equal(t, "newest desc", newPkg.Metadata.Description)
			assert.Equal(t, "newest name", newPkg.Metadata.Name)
			assert.Equal(t, "v1.0.1", newPkg.Metadata.Version)

			newSum := newPkg.Summary()

			labels := newSum.Labels
			require.Len(t, labels, 1)
			assert.Zero(t, labels[0].ID)
			assert.Equal(t, "label_1", labels[0].Name)

			bkts := newSum.Buckets
			require.Len(t, bkts, 1)
			assert.Zero(t, bkts[0].ID)
			assert.Equal(t, "rucket_1", bkts[0].Name)
			hasLabelAssociations(t, bkts[0].LabelAssociations, 1, "label_1")

			dashs := newSum.Dashboards
			require.Len(t, dashs, 1)
			assert.Zero(t, dashs[0].ID)
			assert.Equal(t, "dash_1", dashs[0].Name)
			assert.Equal(t, "desc1", dashs[0].Description)
			hasLabelAssociations(t, dashs[0].LabelAssociations, 1, "label_1")
			require.Len(t, dashs[0].Charts, 1)
			assert.Equal(t, influxdb.ViewPropertyTypeSingleStat, dashs[0].Charts[0].Properties.GetType())

			newEndpoints := newSum.NotificationEndpoints
			require.Len(t, newEndpoints, 1)
			assert.Equal(t, endpoints[0].NotificationEndpoint.GetName(), newEndpoints[0].NotificationEndpoint.GetName())
			assert.Equal(t, endpoints[0].NotificationEndpoint.GetDescription(), newEndpoints[0].NotificationEndpoint.GetDescription())
			hasLabelAssociations(t, newEndpoints[0].LabelAssociations, 1, "label_1")

			require.Len(t, newSum.TelegrafConfigs, 1)
			assert.Equal(t, teles[0].TelegrafConfig.Name, newSum.TelegrafConfigs[0].TelegrafConfig.Name)
			assert.Equal(t, teles[0].TelegrafConfig.Description, newSum.TelegrafConfigs[0].TelegrafConfig.Description)
			hasLabelAssociations(t, newSum.TelegrafConfigs[0].LabelAssociations, 1, "label_1")

			vars := newSum.Variables
			require.Len(t, vars, 1)
			assert.Zero(t, vars[0].ID)
			assert.Equal(t, "new name", vars[0].Name) // new name
			hasLabelAssociations(t, vars[0].LabelAssociations, 1, "label_1")
			varArgs := vars[0].Arguments
			require.NotNil(t, varArgs)
			assert.Equal(t, "query", varArgs.Type)
			assert.Equal(t, influxdb.VariableQueryValues{
				Query:    "buckets()  |> filter(fn: (r) => r.name !~ /^_/)  |> rename(columns: {name: \"_value\"})  |> keep(columns: [\"_value\"])",
				Language: "flux",
			}, varArgs.Values)
		})

		t.Run("error incurs during package application when resources already exist rollsback to prev state", func(t *testing.T) {
			updatePkg, err := pkger.Parse(pkger.EncodingYAML, pkger.FromString(updatePkgYMLStr))
			require.NoError(t, err)

			svc := pkger.NewService(
				pkger.WithBucketSVC(&fakeBucketSVC{
					BucketService: l.BucketService(t),
					killCount:     0, // kill on first update for bucket
				}),
				pkger.WithDashboardSVC(l.DashboardService(t)),
				pkger.WithLabelSVC(l.LabelService(t)),
				pkger.WithNoticationEndpointSVC(l.NotificationEndpointService(t)),
				pkger.WithTelegrafSVC(l.TelegrafService(t)),
				pkger.WithVariableSVC(l.VariableService(t)),
			)

			_, err = svc.Apply(ctx, l.Org.ID, 0, updatePkg)
			require.Error(t, err)

			bkt, err := l.BucketService(t).FindBucketByID(ctx, influxdb.ID(bkts[0].ID))
			require.NoError(t, err)
			// make sure the desc change is not applied and is rolled back to prev desc
			assert.Equal(t, bkts[0].Description, bkt.Description)

			label, err := l.LabelService(t).FindLabelByID(ctx, influxdb.ID(labels[0].ID))
			require.NoError(t, err)
			assert.Equal(t, labels[0].Properties.Description, label.Properties["description"])

			endpoint, err := l.NotificationEndpointService(t).FindNotificationEndpointByID(ctx, endpoints[0].NotificationEndpoint.GetID())
			require.NoError(t, err)
			assert.Equal(t, endpoints[0].NotificationEndpoint.GetDescription(), endpoint.GetDescription())

			v, err := l.VariableService(t).FindVariableByID(ctx, influxdb.ID(vars[0].ID))
			require.NoError(t, err)
			assert.Equal(t, vars[0].Description, v.Description)
		})
	})
}

func timedCtx(d time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(ctx, d)
	var _ = cancel
	return ctx
}

func newPkg(t *testing.T) *pkger.Pkg {
	t.Helper()

	pkg, err := pkger.Parse(pkger.EncodingYAML, pkger.FromString(pkgYMLStr))
	require.NoError(t, err)
	return pkg
}

const pkgYMLStr = `apiVersion: 0.1.0
kind: Package
meta:
  pkgName:      pkg_name
  pkgVersion:   1
  description:  pack description
spec:
  resources:
    - kind: Label
      name: label_1
    - kind: Bucket
      name: rucket_1
      associations:
        - kind: Label
          name: label_1
    - kind: Dashboard
      name: dash_1
      description: desc1
      associations:
        - kind: Label
          name: label_1
      charts:
        - kind:   Single_Stat
          name:   single stat
          suffix: days
          width:  6
          height: 3
          shade: true
          queries:
            - query: >
                from(bucket: v.bucket) |> range(start: v.timeRangeStart) |> filter(fn: (r) => r._measurement == "system") |> filter(fn: (r) => r._field == "uptime") |> last() |> map(fn: (r) => ({r with _value: r._value / 86400})) |> yield(name: "last")
          colors:
            - name: laser
              type: text
              hex: "#8F8AF4"
    - kind: Variable
      name: var_query_1
      description: var_query_1 desc
      type: query
      language: flux
      query: |
        buckets()  |> filter(fn: (r) => r.name !~ /^_/)  |> rename(columns: {name: "_value"})  |> keep(columns: ["_value"])
      associations:
        - kind: Label
          name: label_1
    - kind: Telegraf
      name: first_tele_config
      description: desc
      associations:
        - kind: Label
          name: label_1
      config: |
        [agent]
          interval = "10s"
          metric_batch_size = 1000
          metric_buffer_limit = 10000
          collection_jitter = "0s"
          flush_interval = "10s"
        [[outputs.influxdb_v2]]
          urls = ["http://localhost:9999"]
          token = "$INFLUX_TOKEN"
          organization = "rg"
          bucket = "rucket_3"
        [[inputs.cpu]]
          percpu = true
    - kind: Notification_Endpoint_HTTP
      name: http_none_auth_notification_endpoint
      type: none
      description: http none auth desc
      method: GET
      url:  https://www.example.com/endpoint/noneauth
      status: inactive
      associations:
        - kind: Label
          name: label_1
`

const updatePkgYMLStr = `apiVersion: 0.1.0
kind: Package
meta:
  pkgName:      pkg_name
  pkgVersion:   1
  description:  pack description
spec:
  resources:
    - kind: Label
      name: label_1
      description: new desc
    - kind: Bucket
      name: rucket_1
      description: new desc
      associations:
        - kind: Label
          name: label_1
    - kind: Variable
      name: var_query_1
      description: new desc
      type: query
      language: flux
      query: |
        buckets()  |> filter(fn: (r) => r.name !~ /^_/)  |> rename(columns: {name: "_value"})  |> keep(columns: ["_value"])
      associations:
        - kind: Label
          name: label_1
    - kind: Notification_Endpoint_HTTP
      name: http_none_auth_notification_endpoint
      type: none
      description: new desc
      method: GET
      url:  https://www.example.com/endpoint/noneauth
      status: active
`

type fakeBucketSVC struct {
	influxdb.BucketService
	updateCallCount mock.SafeCount
	killCount       int
}

func (f *fakeBucketSVC) UpdateBucket(ctx context.Context, id influxdb.ID, upd influxdb.BucketUpdate) (*influxdb.Bucket, error) {
	if f.updateCallCount.Count() == f.killCount {
		return nil, errors.New("reached kill count")
	}
	defer f.updateCallCount.IncrFn()()
	return f.BucketService.UpdateBucket(ctx, id, upd)
}

type fakeLabelSVC struct {
	influxdb.LabelService
	callCount mock.SafeCount
	killCount int
}

func (f *fakeLabelSVC) CreateLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	defer f.callCount.IncrFn()()
	if f.callCount.Count() == f.killCount {
		return errors.New("reached kill count")
	}
	return f.LabelService.CreateLabelMapping(ctx, m)
}
