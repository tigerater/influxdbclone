package kv

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/kit/tracing"
)

var (
	labelBucket        = []byte("labelsv1")
	labelMappingBucket = []byte("labelmappingsv1")
)

func (s *Service) initializeLabels(ctx context.Context, tx Tx) error {
	if _, err := tx.Bucket(labelBucket); err != nil {
		return err
	}

	if _, err := tx.Bucket(labelMappingBucket); err != nil {
		return err
	}

	return nil
}

// FindLabelByID finds a label by its ID
func (s *Service) FindLabelByID(ctx context.Context, id influxdb.ID) (*influxdb.Label, error) {
	var l *influxdb.Label

	err := s.kv.View(ctx, func(tx Tx) error {
		label, pe := s.findLabelByID(ctx, tx, id)
		if pe != nil {
			return pe
		}
		l = label
		return nil
	})

	if err != nil {
		return nil, &influxdb.Error{
			Err: err,
		}
	}

	return l, nil
}

func (s *Service) findLabelByID(ctx context.Context, tx Tx, id influxdb.ID) (*influxdb.Label, error) {
	encodedID, err := id.Encode()
	if err != nil {
		return nil, &influxdb.Error{
			Err: err,
		}
	}

	b, err := tx.Bucket(labelBucket)
	if err != nil {
		return nil, err
	}

	v, err := b.Get(encodedID)
	if IsNotFound(err) {
		return nil, &influxdb.Error{
			Code: influxdb.ENotFound,
			Msg:  influxdb.ErrLabelNotFound,
		}
	}

	if err != nil {
		return nil, err
	}

	var l influxdb.Label
	if err := json.Unmarshal(v, &l); err != nil {
		return nil, &influxdb.Error{
			Err: err,
		}
	}

	return &l, nil
}

func filterLabelsFn(filter influxdb.LabelFilter) func(l *influxdb.Label) bool {
	return func(label *influxdb.Label) bool {
		return (filter.Name == "" || (filter.Name == label.Name)) &&
			((filter.OrgID == nil) || (filter.OrgID != nil && *filter.OrgID == label.OrgID))
	}
}

// FindLabels returns a list of labels that match a filter.
func (s *Service) FindLabels(ctx context.Context, filter influxdb.LabelFilter, opt ...influxdb.FindOptions) ([]*influxdb.Label, error) {
	ls := []*influxdb.Label{}
	err := s.kv.View(ctx, func(tx Tx) error {
		labels, err := s.findLabels(ctx, tx, filter)
		if err != nil {
			return err
		}
		ls = labels
		return nil
	})

	if err != nil {
		return nil, err
	}

	return ls, nil
}

func (s *Service) findLabels(ctx context.Context, tx Tx, filter influxdb.LabelFilter) ([]*influxdb.Label, error) {
	ls := []*influxdb.Label{}
	filterFn := filterLabelsFn(filter)
	err := s.forEachLabel(ctx, tx, func(l *influxdb.Label) bool {
		if filterFn(l) {
			ls = append(ls, l)
		}
		return true
	})

	if err != nil {
		return nil, err
	}

	return ls, nil
}

func decodeLabelMappingKey(key []byte) (resourceID influxdb.ID, labelID influxdb.ID, err error) {
	if len(key) != 2*influxdb.IDLength {
		return 0, 0, &influxdb.Error{Code: influxdb.EInvalid, Msg: "malformed label mapping key (please report this error)"}
	}

	if err := (&resourceID).Decode(key[:influxdb.IDLength]); err != nil {
		return 0, 0, &influxdb.Error{Code: influxdb.EInvalid, Msg: "bad resource id", Err: influxdb.ErrInvalidID}
	}

	if err := (&labelID).Decode(key[influxdb.IDLength:]); err != nil {
		return 0, 0, &influxdb.Error{Code: influxdb.EInvalid, Msg: "bad label id", Err: influxdb.ErrInvalidID}
	}

	return resourceID, labelID, nil
}

func (s *Service) findResourceLabels(ctx context.Context, tx Tx, filter influxdb.LabelMappingFilter, ls *[]*influxdb.Label) error {
	if !filter.ResourceID.Valid() {
		return &influxdb.Error{Code: influxdb.EInvalid, Msg: "filter requires a valid resource id", Err: influxdb.ErrInvalidID}
	}
	idx, err := tx.Bucket(labelMappingBucket)
	if err != nil {
		return err
	}

	cur, err := idx.Cursor()
	if err != nil {
		return err
	}

	prefix, err := filter.ResourceID.Encode()
	if err != nil {
		return err
	}

	for k, _ := cur.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = cur.Next() {
		_, id, err := decodeLabelMappingKey(k)
		if err != nil {
			return err
		}

		l, err := s.findLabelByID(ctx, tx, id)
		if l == nil && err != nil {
			// TODO(jm): return error instead of continuing once orphaned mappings are fixed
			// (see https://github.com/influxdata/influxdb/issues/11278)
			continue
		}

		*ls = append(*ls, l)
	}
	return nil
}

func (s *Service) FindResourceLabels(ctx context.Context, filter influxdb.LabelMappingFilter) ([]*influxdb.Label, error) {
	ls := []*influxdb.Label{}
	if err := s.kv.View(ctx, func(tx Tx) error {
		return s.findResourceLabels(ctx, tx, filter, &ls)
	}); err != nil {
		return nil, err
	}

	return ls, nil
}

// CreateLabelMapping creates a new mapping between a resource and a label.
func (s *Service) CreateLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	return s.kv.Update(ctx, func(tx Tx) error {
		return s.createLabelMapping(ctx, tx, m)
	})
}

// createLabelMapping creates a new mapping between a resource and a label.
func (s *Service) createLabelMapping(ctx context.Context, tx Tx, m *influxdb.LabelMapping) error {
	if _, err := s.findLabelByID(ctx, tx, m.LabelID); err != nil {
		return err
	}

	if err := s.putLabelMapping(ctx, tx, m); err != nil {
		return err
	}

	return nil
}

// DeleteLabelMapping deletes a label mapping.
func (s *Service) DeleteLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	err := s.kv.Update(ctx, func(tx Tx) error {
		return s.deleteLabelMapping(ctx, tx, m)
	})
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}
	return nil
}

func (s *Service) deleteLabelMapping(ctx context.Context, tx Tx, m *influxdb.LabelMapping) error {
	key, err := labelMappingKey(m)
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	idx, err := tx.Bucket(labelMappingBucket)
	if err != nil {
		return err
	}

	if err := idx.Delete(key); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	return nil
}

// CreateLabel creates a new label.
func (s *Service) CreateLabel(ctx context.Context, l *influxdb.Label) error {
	err := s.kv.Update(ctx, func(tx Tx) error {
		l.ID = s.IDGenerator.ID()

		if err := s.putLabel(ctx, tx, l); err != nil {
			return err
		}

		if err := s.createLabelUserResourceMappings(ctx, tx, l); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}
	return nil
}

// PutLabel creates a label from the provided struct, without generating a new ID.
func (s *Service) PutLabel(ctx context.Context, l *influxdb.Label) error {
	return s.kv.Update(ctx, func(tx Tx) error {
		var err error
		pe := s.putLabel(ctx, tx, l)
		if pe != nil {
			err = pe
		}
		return err
	})
}

func (s *Service) createLabelUserResourceMappings(ctx context.Context, tx Tx, l *influxdb.Label) error {
	span, ctx := tracing.StartSpanFromContext(ctx)
	defer span.Finish()

	ms, err := s.findUserResourceMappings(ctx, tx, influxdb.UserResourceMappingFilter{
		ResourceType: influxdb.OrgsResourceType,
		ResourceID:   l.OrgID,
	})
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	for _, m := range ms {
		if err := s.createUserResourceMapping(ctx, tx, &influxdb.UserResourceMapping{
			ResourceType: influxdb.LabelsResourceType,
			ResourceID:   l.ID,
			UserID:       m.UserID,
			UserType:     m.UserType,
		}); err != nil {
			return &influxdb.Error{
				Err: err,
			}
		}
	}

	return nil
}

func labelMappingKey(m *influxdb.LabelMapping) ([]byte, error) {
	lid, err := m.LabelID.Encode()
	if err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	rid, err := m.ResourceID.Encode()
	if err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	key := make([]byte, influxdb.IDLength+influxdb.IDLength) // len(rid) + len(lid)
	copy(key, rid)
	copy(key[len(rid):], lid)

	return key, nil
}

func (s *Service) forEachLabel(ctx context.Context, tx Tx, fn func(*influxdb.Label) bool) error {
	b, err := tx.Bucket(labelBucket)
	if err != nil {
		return err
	}

	cur, err := b.Cursor()
	if err != nil {
		return err
	}

	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		l := &influxdb.Label{}
		if err := json.Unmarshal(v, l); err != nil {
			return err
		}
		if !fn(l) {
			break
		}
	}

	return nil
}

// UpdateLabel updates a label.
func (s *Service) UpdateLabel(ctx context.Context, id influxdb.ID, upd influxdb.LabelUpdate) (*influxdb.Label, error) {
	var label *influxdb.Label
	err := s.kv.Update(ctx, func(tx Tx) error {
		labelResponse, pe := s.updateLabel(ctx, tx, id, upd)
		if pe != nil {
			return &influxdb.Error{
				Err: pe,
			}
		}
		label = labelResponse
		return nil
	})

	return label, err
}

func (s *Service) updateLabel(ctx context.Context, tx Tx, id influxdb.ID, upd influxdb.LabelUpdate) (*influxdb.Label, error) {
	label, err := s.findLabelByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	if len(upd.Properties) > 0 && label.Properties == nil {
		label.Properties = make(map[string]string)
	}

	for k, v := range upd.Properties {
		if v == "" {
			delete(label.Properties, k)
		} else {
			label.Properties[k] = v
		}
	}

	if upd.Name != "" {
		label.Name = upd.Name
	}

	if err := label.Validate(); err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	if err := s.putLabel(ctx, tx, label); err != nil {
		return nil, &influxdb.Error{
			Err: err,
		}
	}

	return label, nil
}

// set a label and overwrite any existing label
func (s *Service) putLabel(ctx context.Context, tx Tx, l *influxdb.Label) error {
	v, err := json.Marshal(l)
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	encodedID, err := l.ID.Encode()
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	b, err := tx.Bucket(labelBucket)
	if err != nil {
		return err
	}

	if err := b.Put(encodedID, v); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	return nil
}

// PutLabelMapping writes a label mapping to boltdb
func (s *Service) PutLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	return s.kv.Update(ctx, func(tx Tx) error {
		var err error
		pe := s.putLabelMapping(ctx, tx, m)
		if pe != nil {
			err = pe
		}
		return err
	})
}

func (s *Service) putLabelMapping(ctx context.Context, tx Tx, m *influxdb.LabelMapping) error {
	v, err := json.Marshal(m)
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	key, err := labelMappingKey(m)
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	idx, err := tx.Bucket(labelMappingBucket)
	if err != nil {
		return err
	}

	if err := idx.Put(key, v); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	return nil
}

// DeleteLabel deletes a label.
func (s *Service) DeleteLabel(ctx context.Context, id influxdb.ID) error {
	err := s.kv.Update(ctx, func(tx Tx) error {
		return s.deleteLabel(ctx, tx, id)
	})
	if err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}
	return nil
}

func (s *Service) deleteLabel(ctx context.Context, tx Tx, id influxdb.ID) error {
	_, err := s.findLabelByID(ctx, tx, id)
	if err != nil {
		return err
	}
	encodedID, idErr := id.Encode()
	if idErr != nil {
		return &influxdb.Error{
			Err: idErr,
		}
	}

	b, err := tx.Bucket(labelBucket)
	if err != nil {
		return err
	}

	if err := b.Delete(encodedID); err != nil {
		return &influxdb.Error{
			Err: err,
		}
	}

	if err := s.deleteUserResourceMappings(ctx, tx, influxdb.UserResourceMappingFilter{
		ResourceID:   id,
		ResourceType: influxdb.LabelsResourceType,
	}); err != nil {
		return err
	}

	return nil
}
