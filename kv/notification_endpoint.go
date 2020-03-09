package kv

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/influxdata/influxdb/kit/tracing"
	"github.com/influxdata/influxdb/notification/endpoint"

	"github.com/influxdata/influxdb"
)

var (
	// ErrNotificationEndpointNotFound is used when the notification endpoint is not found.
	ErrNotificationEndpointNotFound = &influxdb.Error{
		Msg:  "notification endpoint not found",
		Code: influxdb.ENotFound,
	}
)

var _ influxdb.NotificationEndpointService = (*Service)(nil)

func newEndpointStore(kv Store) *uniqByNameStore {
	return &uniqByNameStore{
		kv:        kv,
		resource:  "notification endpoint",
		bktName:   []byte("notificationEndpointv1"),
		indexName: []byte("notificationEndpointIndexv1"),
		decodeBucketEntFn: func(key, val []byte) ([]byte, interface{}, error) {
			edp, err := endpoint.UnmarshalJSON(val)
			return key, edp, err
		},
		decodeOrgNameFn: func(body []byte) (influxdb.ID, string, error) {
			edp, err := endpoint.UnmarshalJSON(body)
			if err != nil {
				return 0, "", err
			}
			return edp.GetOrgID(), edp.GetName(), nil
		},
	}
}

// CreateNotificationEndpoint creates a new notification endpoint and sets b.ID with the new identifier.
func (s *Service) CreateNotificationEndpoint(ctx context.Context, edp influxdb.NotificationEndpoint, userID influxdb.ID) error {
	return s.kv.Update(ctx, func(tx Tx) error {
		return s.createNotificationEndpoint(ctx, tx, edp, userID)
	})
}

func (s *Service) createNotificationEndpoint(ctx context.Context, tx Tx, edp influxdb.NotificationEndpoint, userID influxdb.ID) error {
	// TODO(jsteenb2): why is org id check not necesssary if orgID isn't valid... feels odd
	if edp.GetOrgID().Valid() {
		span, ctx := tracing.StartSpanFromContext(ctx)
		// TODO(jsteenb2): this defer doesn't get called until the end of entire function,
		//  need to rip this out as is
		defer span.Finish()

		if _, err := s.findOrganizationByID(ctx, tx, edp.GetOrgID()); err != nil {
			return err
		}
	}
	// notification endpoint name unique
	if _, err := s.findNotificationEndpointByName(ctx, tx, edp.GetOrgID(), edp.GetName()); err == nil {
		return &influxdb.Error{
			Code: influxdb.EConflict,
			Msg:  fmt.Sprintf("notification endpoint with name %s already exists", edp.GetName()),
		}
	}
	id := s.IDGenerator.ID()
	edp.SetID(id)
	now := s.TimeGenerator.Now()
	edp.SetCreatedAt(now)
	edp.SetUpdatedAt(now)
	edp.BackfillSecretKeys()

	if err := edp.Valid(); err != nil {
		return err
	}

	b, err := json.Marshal(edp)
	if err != nil {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "unable to marshal notification endpoint",
			Err:  err,
		}
	}

	ent := Entity{
		ID:    edp.GetID(),
		Name:  edp.GetName(),
		OrgID: edp.GetOrgID(),
		Body:  b,
	}
	if err := s.endpointStore.Put(ctx, tx, ent); err != nil {
		return err
	}

	urm := &influxdb.UserResourceMapping{
		ResourceID:   edp.GetID(),
		UserID:       userID,
		UserType:     influxdb.Owner,
		ResourceType: influxdb.NotificationEndpointResourceType,
	}
	return s.createUserResourceMapping(ctx, tx, urm)
}

func (s *Service) findNotificationEndpointByName(ctx context.Context, tx Tx, orgID influxdb.ID, name string) (influxdb.NotificationEndpoint, error) {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	body, err := s.endpointStore.FindByName(ctx, tx, orgID, name)
	if err != nil {
		return nil, err
	}

	return endpoint.UnmarshalJSON(body)
}

// UpdateNotificationEndpoint updates a single notification endpoint.
// Returns the new notification endpoint after update.
func (s *Service) UpdateNotificationEndpoint(ctx context.Context, id influxdb.ID, edp influxdb.NotificationEndpoint, userID influxdb.ID) (influxdb.NotificationEndpoint, error) {
	var err error
	err = s.kv.Update(ctx, func(tx Tx) error {
		edp, err = s.updateNotificationEndpoint(ctx, tx, id, edp, userID)
		return err
	})
	return edp, err
}

func (s *Service) updateNotificationEndpoint(ctx context.Context, tx Tx, id influxdb.ID, edp influxdb.NotificationEndpoint, userID influxdb.ID) (influxdb.NotificationEndpoint, error) {
	current, err := s.findNotificationEndpointByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	if edpName, curName := edp.GetName(), current.GetName(); edpName != curName {
		edp0, err := s.findNotificationEndpointByName(ctx, tx, current.GetOrgID(), edpName)
		// TODO: when can id every be zero value from store?... feels off
		if err == nil && edp0.GetID() != id {
			return nil, &influxdb.Error{
				Code: influxdb.EConflict,
				Msg:  "notification endpoint name is not unique",
			}
		}

		err = s.endpointStore.deleteInIndex(ctx, tx, current.GetOrgID(), curName)
		if err != nil {
			return nil, err
		}
	}

	// ID and OrganizationID can not be updated
	edp.SetID(current.GetID())
	edp.SetOrgID(current.GetOrgID())
	edp.SetCreatedAt(current.GetCRUDLog().CreatedAt)
	edp.SetUpdatedAt(s.TimeGenerator.Now())

	if err := edp.Valid(); err != nil {
		return nil, err
	}

	b, err := json.Marshal(edp)
	if err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "unable to marshal notification endpoint",
			Err:  err,
		}
	}

	ent := Entity{
		ID:    edp.GetID(),
		Name:  edp.GetName(),
		OrgID: edp.GetOrgID(),
		Body:  b,
	}
	if err := s.endpointStore.Put(ctx, tx, ent); err != nil {
		return nil, err
	}

	return edp, nil
}

// PatchNotificationEndpoint updates a single  notification endpoint with changeset.
// Returns the new notification endpoint state after update.
func (s *Service) PatchNotificationEndpoint(ctx context.Context, id influxdb.ID, upd influxdb.NotificationEndpointUpdate) (influxdb.NotificationEndpoint, error) {
	var edp influxdb.NotificationEndpoint
	if err := s.kv.Update(ctx, func(tx Tx) (err error) {
		edp, err = s.patchNotificationEndpoint(ctx, tx, id, upd)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return edp, nil
}

func (s *Service) patchNotificationEndpoint(ctx context.Context, tx Tx, id influxdb.ID, upd influxdb.NotificationEndpointUpdate) (influxdb.NotificationEndpoint, error) {
	edp, err := s.findNotificationEndpointByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if upd.Name != nil {
		edp0, err := s.findNotificationEndpointByName(ctx, tx, edp.GetOrgID(), *upd.Name)
		if err == nil && edp0.GetID() != id {
			return nil, &influxdb.Error{
				Code: influxdb.EConflict,
				Msg:  "notification endpoint name is not unique",
			}
		}

		err = s.endpointStore.deleteInIndex(ctx, tx, edp.GetOrgID(), edp.GetName())
		if err != nil {
			return nil, err
		}
	}

	if upd.Name != nil {
		edp.SetName(*upd.Name)
	}
	if upd.Description != nil {
		edp.SetDescription(*upd.Description)
	}
	if upd.Status != nil {
		edp.SetStatus(*upd.Status)
	}
	edp.SetUpdatedAt(s.TimeGenerator.Now())

	if err := edp.Valid(); err != nil {
		return nil, err
	}

	b, err := json.Marshal(edp)
	if err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "unable to marshal notification endpoint",
			Err:  err,
		}
	}
	// TODO(jsteenb2): every above here moves into service layer

	ent := Entity{
		ID:    edp.GetID(),
		Name:  edp.GetName(),
		OrgID: edp.GetOrgID(),
		Body:  b,
	}
	if err := s.endpointStore.Put(ctx, tx, ent); err != nil {
		return nil, err
	}

	return edp, nil
}

// PutNotificationEndpoint put a notification endpoint to storage.
func (s *Service) PutNotificationEndpoint(ctx context.Context, edp influxdb.NotificationEndpoint) error {
	// TODO(jsteenb2): all the stuffs before the update should be moved up into the
	//  service layer as well as all the id/time setting items
	if err := edp.Valid(); err != nil {
		return err
	}

	b, err := json.Marshal(edp)
	if err != nil {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "unable to marshal notification endpoint",
			Err:  err,
		}
	}

	return s.kv.Update(ctx, func(tx Tx) (err error) {
		ent := Entity{
			ID:    edp.GetID(),
			Name:  edp.GetName(),
			OrgID: edp.GetOrgID(),
			Body:  b,
		}
		return s.endpointStore.Put(ctx, tx, ent)
	})
}

// FindNotificationEndpointByID returns a single notification endpoint by ID.
func (s *Service) FindNotificationEndpointByID(ctx context.Context, id influxdb.ID) (influxdb.NotificationEndpoint, error) {
	var (
		edp influxdb.NotificationEndpoint
		err error
	)

	err = s.kv.View(ctx, func(tx Tx) error {
		edp, err = s.findNotificationEndpointByID(ctx, tx, id)
		return err
	})

	return edp, err
}

func (s *Service) findNotificationEndpointByID(ctx context.Context, tx Tx, id influxdb.ID) (influxdb.NotificationEndpoint, error) {
	body, err := s.endpointStore.FindByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	return endpoint.UnmarshalJSON(body)
}

// FindNotificationEndpoints returns a list of notification endpoints that match isNext and the total count of matching notification endpoints.
// Additional options provide pagination & sorting.
func (s *Service) FindNotificationEndpoints(ctx context.Context, filter influxdb.NotificationEndpointFilter, opt ...influxdb.FindOptions) (edps []influxdb.NotificationEndpoint, n int, err error) {
	err = s.kv.View(ctx, func(tx Tx) error {
		edps, n, err = s.findNotificationEndpoints(ctx, tx, filter, opt...)
		return err
	})
	return edps, n, err
}

func (s *Service) findNotificationEndpoints(ctx context.Context, tx Tx, filter influxdb.NotificationEndpointFilter, opt ...influxdb.FindOptions) ([]influxdb.NotificationEndpoint, int, error) {
	m, err := s.findUserResourceMappings(ctx, tx, filter.UserResourceMappingFilter)
	if err != nil {
		return nil, 0, err
	}

	if len(m) == 0 {
		return []influxdb.NotificationEndpoint{}, 0, nil
	}

	idMap := make(map[influxdb.ID]bool)
	for _, item := range m {
		idMap[item.ResourceID] = true
	}

	if filter.Org != nil {
		o, err := s.findOrganizationByName(ctx, tx, *filter.Org)
		if err != nil {
			return nil, 0, &influxdb.Error{
				Err: err,
			}
		}
		filter.OrgID = &o.ID
	}

	var o influxdb.FindOptions
	if len(opt) > 0 {
		o = opt[0]
	}

	edps := make([]influxdb.NotificationEndpoint, 0)
	err = s.endpointStore.Find(ctx, tx, o, filterEndpointsFn(idMap, filter), func(k []byte, v interface{}) {
		edps = append(edps, v.(influxdb.NotificationEndpoint))
	})
	if err != nil {
		return nil, 0, err
	}

	return edps, len(edps), err
}

func filterEndpointsFn(idMap map[influxdb.ID]bool, filter influxdb.NotificationEndpointFilter) func([]byte, interface{}) bool {
	return func(key []byte, val interface{}) bool {
		edp := val.(influxdb.NotificationEndpoint)
		if filter.ID != nil && edp.GetID() != *filter.ID {
			return false
		}

		if filter.OrgID != nil && edp.GetOrgID() != *filter.OrgID {
			return false
		}

		if idMap == nil {
			return true
		}
		return idMap[edp.GetID()]
	}
}

// DeleteNotificationEndpoint removes a notification endpoint by ID.
func (s *Service) DeleteNotificationEndpoint(ctx context.Context, id influxdb.ID) (flds []influxdb.SecretField, orgID influxdb.ID, err error) {
	err = s.kv.Update(ctx, func(tx Tx) error {
		flds, orgID, err = s.deleteNotificationEndpoint(ctx, tx, id)
		return err
	})
	return flds, orgID, err
}

func (s *Service) deleteNotificationEndpoint(ctx context.Context, tx Tx, id influxdb.ID) (flds []influxdb.SecretField, orgID influxdb.ID, err error) {
	edp, err := s.findNotificationEndpointByID(ctx, tx, id)
	if err != nil {
		return nil, 0, err
	}

	if err := s.endpointStore.Delete(ctx, tx, id); err != nil {
		return nil, 0, err
	}

	return edp.SecretFields(), edp.GetOrgID(), s.deleteUserResourceMappings(ctx, tx, influxdb.UserResourceMappingFilter{
		ResourceID:   id,
		ResourceType: influxdb.NotificationEndpointResourceType,
	})
}
