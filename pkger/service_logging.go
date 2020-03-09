package pkger

import (
	"context"
	"time"

	"github.com/influxdata/influxdb"
	"go.uber.org/zap"
)

type loggingMW struct {
	logger *zap.Logger
	next   SVC
}

// MWLogging adds logging functionality for the service.
func MWLogging(log *zap.Logger) SVCMiddleware {
	return func(svc SVC) SVC {
		return &loggingMW{
			logger: log,
			next:   svc,
		}
	}
}

var _ SVC = (*loggingMW)(nil)

func (s *loggingMW) CreatePkg(ctx context.Context, setters ...CreatePkgSetFn) (pkg *Pkg, err error) {
	defer func(start time.Time) {
		dur := zap.Duration("took", time.Since(start))
		if err != nil {
			s.logger.Error("failed to create pkg", zap.Error(err), dur)
			return
		}
		s.logger.Info("pkg create", append(s.summaryLogFields(pkg.Summary()), dur)...)
	}(time.Now())
	return s.next.CreatePkg(ctx, setters...)
}

func (s *loggingMW) DryRun(ctx context.Context, orgID, userID influxdb.ID, pkg *Pkg) (sum Summary, diff Diff, err error) {
	defer func(start time.Time) {
		dur := zap.Duration("took", time.Since(start))
		if err != nil {
			s.logger.Error("failed to dry run pkg",
				zap.String("orgID", orgID.String()),
				zap.String("userID", userID.String()),
				zap.Error(err),
				dur,
			)
			return
		}
		s.logger.Info("pkg dry run successful", append(s.summaryLogFields(sum), dur)...)
	}(time.Now())
	return s.next.DryRun(ctx, orgID, userID, pkg)
}

func (s *loggingMW) Apply(ctx context.Context, orgID, userID influxdb.ID, pkg *Pkg, opts ...ApplyOptFn) (sum Summary, err error) {
	defer func(start time.Time) {
		dur := zap.Duration("took", time.Since(start))
		if err != nil {
			s.logger.Error("failed to apply pkg",
				zap.String("orgID", orgID.String()),
				zap.String("userID", userID.String()),
				zap.Error(err),
				dur,
			)
		}
		s.logger.Info("pkg apply successful", append(s.summaryLogFields(sum), dur)...)
	}(time.Now())
	return s.next.Apply(ctx, orgID, userID, pkg, opts...)
}

func (s *loggingMW) summaryLogFields(sum Summary) []zap.Field {
	potentialFields := []struct {
		key string
		val int
	}{
		{key: "buckets", val: len(sum.Buckets)},
		{key: "checks", val: len(sum.Checks)},
		{key: "dashboards", val: len(sum.Dashboards)},
		{key: "endpoints", val: len(sum.NotificationEndpoints)},
		{key: "labels", val: len(sum.Labels)},
		{key: "label_mappings", val: len(sum.LabelMappings)},
		{key: "rules", val: len(sum.NotificationRules)},
		{key: "secrets", val: len(sum.MissingSecrets)},
		{key: "tasks", val: len(sum.Tasks)},
		{key: "telegrafs", val: len(sum.TelegrafConfigs)},
		{key: "variables", val: len(sum.Variables)},
	}

	var fields []zap.Field
	for _, f := range potentialFields {
		if f.val > 0 {
			fields = append(fields, zap.Int("num_"+f.key, f.val))
		}
	}

	return fields
}
