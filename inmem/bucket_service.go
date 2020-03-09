package inmem

import (
	"context"
	"fmt"
	"sort"
	"time"

	platform "github.com/influxdata/influxdb"
)

var (
	errBucketNotFound = "bucket not found"
)

func (s *Service) loadBucket(ctx context.Context, id platform.ID) (*platform.Bucket, *platform.Error) {
	i, ok := s.bucketKV.Load(id.String())
	if !ok {
		return nil, &platform.Error{
			Code: platform.ENotFound,
			Msg:  errBucketNotFound,
		}
	}

	b, ok := i.(platform.Bucket)
	if !ok {
		return nil, &platform.Error{
			Code: platform.EInternal,
			Msg:  fmt.Sprintf("type %T is not a bucket", i),
		}
	}

	return &b, nil
}

// FindBucketByID returns a single bucket by ID.
func (s *Service) FindBucketByID(ctx context.Context, id platform.ID) (*platform.Bucket, error) {
	var b *platform.Bucket
	var pe *platform.Error
	var err error

	if b, pe = s.loadBucket(ctx, id); pe != nil {
		pe.Op = OpPrefix + platform.OpFindBucketByID
		err = pe
	}
	return b, err
}

func (s *Service) forEachBucket(ctx context.Context, descending bool, fn func(b *platform.Bucket) bool) error {
	var err error
	bs := make([]*platform.Bucket, 0)
	s.bucketKV.Range(func(k, v interface{}) bool {
		b, ok := v.(platform.Bucket)
		if !ok {
			err = fmt.Errorf("type %T is not a bucket", v)
			return false
		}

		bs = append(bs, &b)
		return true
	})

	// sort by id
	sort.Slice(bs, func(i, j int) bool {
		if descending {
			return bs[i].ID.String() > bs[j].ID.String()
		}
		return bs[i].ID.String() < bs[j].ID.String()
	})

	for _, b := range bs {
		if !fn(b) {
			return nil
		}
	}

	return err
}

func (s *Service) filterBuckets(ctx context.Context, fn func(b *platform.Bucket) bool, opts ...platform.FindOptions) ([]*platform.Bucket, error) {
	var offset, limit, count int
	var descending bool
	if len(opts) > 0 {
		offset = opts[0].Offset
		limit = opts[0].Limit
		descending = opts[0].Descending
	}

	buckets := []*platform.Bucket{}
	err := s.forEachBucket(ctx, descending, func(b *platform.Bucket) bool {
		if fn(b) {
			if count >= offset {
				buckets = append(buckets, b)
			}
			count += 1
		}

		if limit > 0 && len(buckets) >= limit {
			return false
		}

		return true
	})

	if err != nil {
		return nil, err
	}

	return buckets, nil
}

// FindBucket returns the first bucket that matches filter.
func (s *Service) FindBucket(ctx context.Context, filter platform.BucketFilter) (*platform.Bucket, error) {
	op := OpPrefix + platform.OpFindBucket
	var err error
	var b *platform.Bucket

	if filter.ID == nil && filter.Name == nil && filter.OrganizationID == nil {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Op:   op,
			Msg:  "no filter parameters provided",
		}
	}

	// filter by bucket id
	if filter.ID != nil {
		b, err = s.FindBucketByID(ctx, *filter.ID)
		if err != nil {
			return nil, &platform.Error{
				Op:  op,
				Err: err,
			}
		}
		return b, nil
	}

	bs, n, err := s.FindBuckets(ctx, filter)
	if err != nil {
		return nil, err
	}

	if n < 1 && filter.Name != nil {
		return nil, &platform.Error{
			Code: platform.ENotFound,
			Op:   op,
			Msg:  fmt.Sprintf("bucket %q not found", *filter.Name),
		}
	} else if n < 1 {
		return nil, &platform.Error{
			Code: platform.ENotFound,
			Op:   op,
			Msg:  "bucket not found",
		}
	}

	return bs[0], nil
}

func (s *Service) findBuckets(ctx context.Context, filter platform.BucketFilter, opt ...platform.FindOptions) ([]*platform.Bucket, *platform.Error) {
	// filter by bucket id
	if filter.ID != nil {
		b, err := s.FindBucketByID(ctx, *filter.ID)
		if err != nil {
			return nil, &platform.Error{
				Err: err,
			}
		}

		return []*platform.Bucket{b}, nil
	}

	if filter.Org != nil {
		o, err := s.findOrganizationByName(ctx, *filter.Org)
		if err != nil {
			return nil, &platform.Error{
				Err: err,
			}
		}
		filter.OrganizationID = &o.ID
	}

	filterFunc := func(b *platform.Bucket) bool { return true }

	if filter.Name != nil && filter.OrganizationID != nil {
		filterFunc = func(b *platform.Bucket) bool {
			return b.Name == *filter.Name && b.OrgID == *filter.OrganizationID
		}
	} else if filter.Name != nil {
		// filter by bucket name
		filterFunc = func(b *platform.Bucket) bool {
			return b.Name == *filter.Name
		}
	} else if filter.OrganizationID != nil {
		// filter by organization id
		filterFunc = func(b *platform.Bucket) bool {
			return b.OrgID == *filter.OrganizationID
		}
	}

	bs, err := s.filterBuckets(ctx, filterFunc, opt...)
	if err != nil {
		return nil, &platform.Error{
			Err: err,
		}
	}
	return bs, nil
}

// FindBuckets returns a list of buckets that match filter and the total count of matching buckets.
// Additional options provide pagination & sorting.
func (s *Service) FindBuckets(ctx context.Context, filter platform.BucketFilter, opt ...platform.FindOptions) ([]*platform.Bucket, int, error) {
	if filter.Name != nil {
		internal, err := s.findSystemBucket(*filter.Name)
		// if found in our internals list, return mock
		if err == nil {
			return []*platform.Bucket{internal}, 0, nil
		}
	}

	var err error
	bs, pe := s.findBuckets(ctx, filter, opt...)
	if pe != nil {
		pe.Op = OpPrefix + platform.OpFindBuckets
		err = pe
		return nil, 0, err
	}
	return bs, len(bs), nil
}

func (s *Service) findSystemBucket(n string) (*platform.Bucket, error) {
	switch n {
	case "_tasks":
		return &platform.Bucket{
			ID:              platform.TasksSystemBucketID,
			Type:            platform.BucketTypeSystem,
			Name:            "_tasks",
			RetentionPeriod: time.Hour * 24 * 3,
			Description:     "System bucket for task logs",
		}, nil
	case "_monitoring":
		return &platform.Bucket{
			ID:              platform.MonitoringSystemBucketID,
			Type:            platform.BucketTypeSystem,
			Name:            "_monitoring",
			RetentionPeriod: time.Hour * 24 * 7,
			Description:     "System bucket for monitoring logs",
		}, nil
	default:
		return nil, &platform.Error{
			Code: platform.ENotFound,
			Msg:  fmt.Sprintf("system bucket %q not found", n),
		}
	}
}

// CreateBucket creates a new bucket and sets b.ID with the new identifier.
func (s *Service) CreateBucket(ctx context.Context, b *platform.Bucket) error {
	if b.OrgID.Valid() {
		_, pe := s.FindOrganizationByID(ctx, b.OrgID)
		if pe != nil {
			return &platform.Error{
				Err: pe,
				Op:  OpPrefix + platform.OpCreateBucket,
			}
		}
	}
	filter := platform.BucketFilter{
		Name:           &b.Name,
		OrganizationID: &b.OrgID,
	}
	if _, err := s.FindBucket(ctx, filter); err == nil {
		return &platform.Error{
			Code: platform.EConflict,
			Op:   OpPrefix + platform.OpCreateBucket,
			Msg:  fmt.Sprintf("bucket with name %s already exists", b.Name),
		}
	}
	b.ID = s.IDGenerator.ID()
	b.CreatedAt = s.Now()
	b.UpdatedAt = s.Now()
	return s.PutBucket(ctx, b)
}

// PutBucket sets bucket with the current ID.
func (s *Service) PutBucket(ctx context.Context, b *platform.Bucket) error {
	s.bucketKV.Store(b.ID.String(), *b)
	return nil
}

// UpdateBucket updates a single bucket with changeset.
// Returns the new bucket state after update.
func (s *Service) UpdateBucket(ctx context.Context, id platform.ID, upd platform.BucketUpdate) (*platform.Bucket, error) {
	b, err := s.FindBucketByID(ctx, id)
	if err != nil {
		return nil, &platform.Error{
			Op:  OpPrefix + platform.OpUpdateBucket,
			Err: err,
		}
	}

	if upd.Name != nil {
		b.Name = *upd.Name
	}

	if upd.RetentionPeriod != nil {
		b.RetentionPeriod = *upd.RetentionPeriod
	}

	if upd.Description != nil {
		b.Description = *upd.Description
	}

	b0, err := s.FindBucket(ctx, platform.BucketFilter{
		Name: upd.Name,
	})
	if err == nil && b0.ID != id {
		return nil, &platform.Error{
			Code: platform.EConflict,
			Msg:  "bucket name is not unique",
		}
	}

	b.UpdatedAt = s.Now()
	s.bucketKV.Store(b.ID.String(), b)

	return b, nil
}

// DeleteBucket removes a bucket by ID.
func (s *Service) DeleteBucket(ctx context.Context, id platform.ID) error {
	if _, err := s.FindBucketByID(ctx, id); err != nil {
		return &platform.Error{
			Op:  OpPrefix + platform.OpDeleteBucket,
			Err: err,
		}
	}
	s.bucketKV.Delete(id.String())

	// return s.deleteLabel(ctx, platform.LabelFilter{ResourceID: id})
	return nil
}

// DeleteOrganizationBuckets removes all the buckets for a given org
func (s *Service) DeleteOrganizationBuckets(ctx context.Context, id platform.ID) error {
	bucks, err := s.findBuckets(ctx, platform.BucketFilter{
		OrganizationID: &id,
	})
	if err != nil {
		return err
	}
	for _, buck := range bucks {
		s.bucketKV.Delete(buck.ID.String())
	}
	return nil
}
