package kv

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/influxdata/influxdb/notification/rule"

	"github.com/influxdata/influxdb"
)

var (
	notificationRuleBucket = []byte("notificationRulev1")

	// ErrNotificationRuleNotFound is used when the notification rule is not found.
	ErrNotificationRuleNotFound = &influxdb.Error{
		Msg:  "notification rule not found",
		Code: influxdb.ENotFound,
	}

	// ErrInvalidNotificationRuleID is used when the service was provided
	// an invalid ID format.
	ErrInvalidNotificationRuleID = &influxdb.Error{
		Code: influxdb.EInvalid,
		Msg:  "provided notification rule ID has invalid format",
	}
)

var _ influxdb.NotificationRuleStore = (*Service)(nil)

func (s *Service) initializeNotificationRule(ctx context.Context, tx Tx) error {
	if _, err := s.notificationRuleBucket(tx); err != nil {
		return err
	}
	return nil
}

// UnavailableNotificationRuleStoreError is used if we aren't able to interact with the
// store, it means the store is not available at the moment (e.g. network).
func UnavailableNotificationRuleStoreError(err error) *influxdb.Error {
	return &influxdb.Error{
		Code: influxdb.EInternal,
		Msg:  fmt.Sprintf("Unable to connect to notification rule store service. Please try again; Err: %v", err),
		Op:   "kv/notificationRule",
	}
}

// InternalNotificationRuleStoreError is used when the error comes from an
// internal system.
func InternalNotificationRuleStoreError(err error) *influxdb.Error {
	return &influxdb.Error{
		Code: influxdb.EInternal,
		Msg:  fmt.Sprintf("Unknown internal notificationRule data error; Err: %v", err),
		Op:   "kv/notificationRule",
	}
}

func (s *Service) notificationRuleBucket(tx Tx) (Bucket, error) {
	b, err := tx.Bucket(notificationRuleBucket)
	if err != nil {
		return nil, UnavailableNotificationRuleStoreError(err)
	}
	return b, nil
}

// CreateNotificationRule creates a new notification rule and sets b.ID with the new identifier.
func (s *Service) CreateNotificationRule(ctx context.Context, nr influxdb.NotificationRule, userID influxdb.ID) error {
	return s.kv.Update(ctx, func(tx Tx) error {
		return s.createNotificationRule(ctx, tx, nr, userID)
	})
}

func (s *Service) createNotificationRule(ctx context.Context, tx Tx, nr influxdb.NotificationRule, userID influxdb.ID) error {
	id := s.IDGenerator.ID()
	nr.SetID(id)
	now := s.TimeGenerator.Now()
	nr.SetCreatedAt(now)
	nr.SetUpdatedAt(now)
	if err := s.putNotificationRule(ctx, tx, nr); err != nil {
		return err
	}

	urm := &influxdb.UserResourceMapping{
		ResourceID:   id,
		UserID:       userID,
		UserType:     influxdb.Owner,
		ResourceType: influxdb.NotificationRuleResourceType,
	}
	return s.createUserResourceMapping(ctx, tx, urm)
}

// UpdateNotificationRule updates a single notification rule.
// Returns the new notification rule after update.
func (s *Service) UpdateNotificationRule(ctx context.Context, id influxdb.ID, nr influxdb.NotificationRule, userID influxdb.ID) (influxdb.NotificationRule, error) {
	var err error
	err = s.kv.Update(ctx, func(tx Tx) error {
		nr, err = s.updateNotificationRule(ctx, tx, id, nr, userID)
		return err
	})
	return nr, err
}

func (s *Service) updateNotificationRule(ctx context.Context, tx Tx, id influxdb.ID, nr influxdb.NotificationRule, userID influxdb.ID) (influxdb.NotificationRule, error) {
	current, err := s.findNotificationRuleByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	// ID and OrganizationID can not be updated
	nr.SetID(current.GetID())
	nr.SetOrgID(current.GetOrgID())
	nr.SetCreatedAt(current.GetCRUDLog().CreatedAt)
	nr.SetUpdatedAt(s.TimeGenerator.Now())
	err = s.putNotificationRule(ctx, tx, nr)
	return nr, err
}

// PatchNotificationRule updates a single  notification rule with changeset.
// Returns the new notification rule state after update.
func (s *Service) PatchNotificationRule(ctx context.Context, id influxdb.ID, upd influxdb.NotificationRuleUpdate) (influxdb.NotificationRule, error) {
	var nr influxdb.NotificationRule
	if err := s.kv.Update(ctx, func(tx Tx) (err error) {
		nr, err = s.patchNotificationRule(ctx, tx, id, upd)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return nr, nil
}

func (s *Service) patchNotificationRule(ctx context.Context, tx Tx, id influxdb.ID, upd influxdb.NotificationRuleUpdate) (influxdb.NotificationRule, error) {
	nr, err := s.findNotificationRuleByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	if upd.Name != nil {
		nr.SetName(*upd.Name)
	}
	if upd.Description != nil {
		nr.SetDescription(*upd.Description)
	}
	if upd.Status != nil {
		nr.SetStatus(*upd.Status)
	}
	nr.SetUpdatedAt(s.TimeGenerator.Now())
	err = s.putNotificationRule(ctx, tx, nr)
	if err != nil {
		return nil, err
	}

	return nr, nil
}

// PutNotificationRule put a notification rule to storage.
func (s *Service) PutNotificationRule(ctx context.Context, nr influxdb.NotificationRule) error {
	return s.kv.Update(ctx, func(tx Tx) (err error) {
		return s.putNotificationRule(ctx, tx, nr)
	})
}

func (s *Service) putNotificationRule(ctx context.Context, tx Tx, nr influxdb.NotificationRule) error {
	if err := nr.Valid(); err != nil {
		return err
	}
	encodedID, _ := nr.GetID().Encode()

	v, err := json.Marshal(nr)
	if err != nil {
		return err
	}

	bucket, err := s.notificationRuleBucket(tx)
	if err != nil {
		return err
	}

	if err := bucket.Put(encodedID, v); err != nil {
		return UnavailableNotificationRuleStoreError(err)
	}
	return nil
}

// FindNotificationRuleByID returns a single notification rule by ID.
func (s *Service) FindNotificationRuleByID(ctx context.Context, id influxdb.ID) (influxdb.NotificationRule, error) {
	var (
		nr  influxdb.NotificationRule
		err error
	)

	err = s.kv.View(ctx, func(tx Tx) error {
		nr, err = s.findNotificationRuleByID(ctx, tx, id)
		return err
	})

	return nr, err
}

func (s *Service) findNotificationRuleByID(ctx context.Context, tx Tx, id influxdb.ID) (influxdb.NotificationRule, error) {
	encID, err := id.Encode()
	if err != nil {
		return nil, ErrInvalidNotificationRuleID
	}

	bucket, err := s.notificationRuleBucket(tx)
	if err != nil {
		return nil, err
	}

	v, err := bucket.Get(encID)
	if IsNotFound(err) {
		return nil, ErrNotificationRuleNotFound
	}
	if err != nil {
		return nil, InternalNotificationRuleStoreError(err)
	}

	return rule.UnmarshalJSON(v)
}

// FindNotificationRules returns a list of notification rules that match filter and the total count of matching notification rules.
// Additional options provide pagination & sorting.
func (s *Service) FindNotificationRules(ctx context.Context, filter influxdb.NotificationRuleFilter, opt ...influxdb.FindOptions) (nrs []influxdb.NotificationRule, n int, err error) {
	err = s.kv.View(ctx, func(tx Tx) error {
		nrs, n, err = s.findNotificationRules(ctx, tx, filter)
		return err
	})
	return nrs, n, err
}

func (s *Service) findNotificationRules(ctx context.Context, tx Tx, filter influxdb.NotificationRuleFilter, opt ...influxdb.FindOptions) ([]influxdb.NotificationRule, int, error) {
	nrs := make([]influxdb.NotificationRule, 0)

	m, err := s.findUserResourceMappings(ctx, tx, filter.UserResourceMappingFilter)
	if err != nil {
		return nil, 0, err
	}

	if len(m) == 0 {
		return nrs, 0, nil
	}

	idMap := make(map[influxdb.ID]bool)
	for _, item := range m {
		idMap[item.ResourceID] = false
	}

	if filter.OrgID != nil || filter.Organization != nil {
		o, err := s.FindOrganization(ctx, influxdb.OrganizationFilter{
			ID:   filter.OrgID,
			Name: filter.Organization,
		})

		if err != nil {
			return nrs, 0, err
		}
		filter.OrgID = &o.ID
	}

	var offset, limit, count int
	var descending bool
	if len(opt) > 0 {
		offset = opt[0].Offset
		limit = opt[0].Limit
		descending = opt[0].Descending
	}
	filterFn := filterNotificationRulesFn(idMap, filter)
	err = s.forEachNotificationRule(ctx, tx, descending, func(nr influxdb.NotificationRule) bool {
		if filterFn(nr) {
			if count >= offset {
				nrs = append(nrs, nr)
			}
			count++
		}

		if limit > 0 && len(nrs) >= limit {
			return false
		}

		return true
	})

	return nrs, len(nrs), err
}

// forEachNotificationRule will iterate through all notification rules while fn returns true.
func (s *Service) forEachNotificationRule(ctx context.Context, tx Tx, descending bool, fn func(influxdb.NotificationRule) bool) error {

	bkt, err := s.notificationRuleBucket(tx)
	if err != nil {
		return err
	}

	cur, err := bkt.Cursor()
	if err != nil {
		return err
	}

	var k, v []byte
	if descending {
		k, v = cur.Last()
	} else {
		k, v = cur.First()
	}

	for k != nil {
		nr, err := rule.UnmarshalJSON(v)
		if err != nil {
			return err
		}
		if !fn(nr) {
			break
		}

		if descending {
			k, v = cur.Prev()
		} else {
			k, v = cur.Next()
		}
	}

	return nil
}

func filterNotificationRulesFn(
	idMap map[influxdb.ID]bool,
	filter influxdb.NotificationRuleFilter) func(nr influxdb.NotificationRule) bool {
	if filter.OrgID != nil {
		return func(nr influxdb.NotificationRule) bool {
			_, ok := idMap[nr.GetID()]
			return nr.GetOrgID() == *filter.OrgID && ok
		}
	}

	return func(nr influxdb.NotificationRule) bool {
		_, ok := idMap[nr.GetID()]
		return ok
	}
}

// DeleteNotificationRule removes a notification rule by ID.
func (s *Service) DeleteNotificationRule(ctx context.Context, id influxdb.ID) error {
	return s.kv.Update(ctx, func(tx Tx) error {
		return s.deleteNotificationRule(ctx, tx, id)
	})
}

func (s *Service) deleteNotificationRule(ctx context.Context, tx Tx, id influxdb.ID) error {
	encodedID, err := id.Encode()
	if err != nil {
		return ErrInvalidNotificationRuleID
	}

	bucket, err := s.notificationRuleBucket(tx)
	if err != nil {
		return err
	}

	_, err = bucket.Get(encodedID)
	if IsNotFound(err) {
		return ErrNotificationRuleNotFound
	}
	if err != nil {
		return InternalNotificationRuleStoreError(err)
	}

	if err := bucket.Delete(encodedID); err != nil {
		return InternalNotificationRuleStoreError(err)
	}

	return s.deleteUserResourceMappings(ctx, tx, influxdb.UserResourceMappingFilter{
		ResourceID:   id,
		ResourceType: influxdb.NotificationRuleResourceType,
	})
}
