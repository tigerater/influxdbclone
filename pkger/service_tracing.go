package pkger

import (
	"context"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/kit/tracing"
)

type traceMW struct {
	next SVC
}

// MWTracing adds tracing functionality for the service.
func MWTracing() SVCMiddleware {
	return func(svc SVC) SVC {
		return &traceMW{next: svc}
	}
}

var _ SVC = (*traceMW)(nil)

func (s *traceMW) CreatePkg(ctx context.Context, setters ...CreatePkgSetFn) (pkg *Pkg, err error) {
	span, ctx := tracing.StartSpanFromContextWithOperationName(ctx, "CreatePkg")
	defer span.Finish()
	return s.next.CreatePkg(ctx, setters...)
}

func (s *traceMW) DryRun(ctx context.Context, orgID, userID influxdb.ID, pkg *Pkg) (sum Summary, diff Diff, err error) {
	span, ctx := tracing.StartSpanFromContextWithOperationName(ctx, "DryRun")
	span.LogKV("orgID", orgID.String(), "userID", userID.String())
	defer span.Finish()
	return s.next.DryRun(ctx, orgID, userID, pkg)
}

func (s *traceMW) Apply(ctx context.Context, orgID, userID influxdb.ID, pkg *Pkg, opts ...ApplyOptFn) (sum Summary, err error) {
	span, ctx := tracing.StartSpanFromContextWithOperationName(ctx, "Apply")
	span.LogKV("orgID", orgID.String(), "userID", userID.String())
	defer span.Finish()
	return s.next.Apply(ctx, orgID, userID, pkg, opts...)
}
