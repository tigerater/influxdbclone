package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"go.uber.org/zap"

	"github.com/influxdata/influxdb"
	"github.com/julienschmidt/httprouter"
)

// LabelHandler represents an HTTP API handler for labels
type LabelHandler struct {
	*httprouter.Router
	influxdb.HTTPErrorHandler
	Logger *zap.Logger

	LabelService influxdb.LabelService
}

const (
	labelsPath   = "/api/v2/labels"
	labelsIDPath = "/api/v2/labels/:id"
)

// NewLabelHandler returns a new instance of LabelHandler
func NewLabelHandler(s influxdb.LabelService, he influxdb.HTTPErrorHandler) *LabelHandler {
	h := &LabelHandler{
		Router:           NewRouter(he),
		HTTPErrorHandler: he,
		Logger:           zap.NewNop(),
		LabelService:     s,
	}

	h.HandlerFunc("POST", labelsPath, h.handlePostLabel)
	h.HandlerFunc("GET", labelsPath, h.handleGetLabels)

	h.HandlerFunc("GET", labelsIDPath, h.handleGetLabel)
	h.HandlerFunc("PATCH", labelsIDPath, h.handlePatchLabel)
	h.HandlerFunc("DELETE", labelsIDPath, h.handleDeleteLabel)

	return h
}

// handlePostLabel is the HTTP handler for the POST /api/v2/labels route.
func (h *LabelHandler) handlePostLabel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("label create request", zap.String("r", fmt.Sprint(r)))

	req, err := decodePostLabelRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := h.LabelService.CreateLabel(ctx, req.Label); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("label created", zap.String("label", fmt.Sprint(req.Label)))
	if err := encodeResponse(ctx, w, http.StatusCreated, newLabelResponse(req.Label)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type postLabelRequest struct {
	Label *influxdb.Label
}

func (b postLabelRequest) Validate() error {
	if b.Label.Name == "" {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "label requires a name",
		}
	}
	if !b.Label.OrgID.Valid() {
		return &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "label requires a valid orgID",
		}
	}
	return nil
}

// TODO(jm): ensure that the specified org actually exists
func decodePostLabelRequest(ctx context.Context, r *http.Request) (*postLabelRequest, error) {
	l := &influxdb.Label{}
	if err := json.NewDecoder(r.Body).Decode(l); err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "unable to decode label request",
			Err:  err,
		}
	}

	req := &postLabelRequest{
		Label: l,
	}

	return req, req.Validate()
}

// handleGetLabels is the HTTP handler for the GET /api/v2/labels route.
func (h *LabelHandler) handleGetLabels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("labels retrieve request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeGetLabelsRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	labels, err := h.LabelService.FindLabels(ctx, req.filter)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("labels retrived", zap.String("labels", fmt.Sprint(labels)))
	err = encodeResponse(ctx, w, http.StatusOK, newLabelsResponse(labels))
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
}

type getLabelsRequest struct {
	filter influxdb.LabelFilter
}

func decodeGetLabelsRequest(ctx context.Context, r *http.Request) (*getLabelsRequest, error) {
	qp := r.URL.Query()
	req := &getLabelsRequest{}

	if orgID := qp.Get("orgID"); orgID != "" {
		id, err := influxdb.IDFromString(orgID)
		if err != nil {
			return nil, err
		}
		req.filter.OrgID = id
	}

	return req, nil
}

// handleGetLabel is the HTTP handler for the GET /api/v2/labels/id route.
func (h *LabelHandler) handleGetLabel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("label retrieve request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeGetLabelRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	l, err := h.LabelService.FindLabelByID(ctx, req.LabelID)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("label retrieved", zap.String("label", fmt.Sprint(l)))
	if err := encodeResponse(ctx, w, http.StatusOK, newLabelResponse(l)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type getLabelRequest struct {
	LabelID influxdb.ID
}

func decodeGetLabelRequest(ctx context.Context, r *http.Request) (*getLabelRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "label id is not valid",
		}
	}

	var i influxdb.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}
	req := &getLabelRequest{
		LabelID: i,
	}

	return req, nil
}

// handleDeleteLabel is the HTTP handler for the DELETE /api/v2/labels/:id route.
func (h *LabelHandler) handleDeleteLabel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("label delete request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeDeleteLabelRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := h.LabelService.DeleteLabel(ctx, req.LabelID); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("label deleted", zap.String("labelID", fmt.Sprint(req.LabelID)))
	w.WriteHeader(http.StatusNoContent)
}

type deleteLabelRequest struct {
	LabelID influxdb.ID
}

func decodeDeleteLabelRequest(ctx context.Context, r *http.Request) (*deleteLabelRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "url missing id",
		}
	}

	var i influxdb.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}
	req := &deleteLabelRequest{
		LabelID: i,
	}

	return req, nil
}

// handlePatchLabel is the HTTP handler for the PATCH /api/v2/labels route.
func (h *LabelHandler) handlePatchLabel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("label update request", zap.String("r", fmt.Sprint(r)))

	req, err := decodePatchLabelRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	l, err := h.LabelService.UpdateLabel(ctx, req.LabelID, req.Update)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("label updated", zap.String("label", fmt.Sprint(l)))
	if err := encodeResponse(ctx, w, http.StatusOK, newLabelResponse(l)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type patchLabelRequest struct {
	Update  influxdb.LabelUpdate
	LabelID influxdb.ID
}

func decodePatchLabelRequest(ctx context.Context, r *http.Request) (*patchLabelRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "url missing id",
		}
	}

	var i influxdb.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	upd := &influxdb.LabelUpdate{}
	if err := json.NewDecoder(r.Body).Decode(upd); err != nil {
		return nil, err
	}

	return &patchLabelRequest{
		Update:  *upd,
		LabelID: i,
	}, nil
}

// LabelService connects to Influx via HTTP using tokens to manage labels
type LabelService struct {
	Addr               string
	Token              string
	InsecureSkipVerify bool
	BasePath           string
	OpPrefix           string
}

type labelResponse struct {
	Links map[string]string `json:"links"`
	Label influxdb.Label    `json:"label"`
}

func newLabelResponse(l *influxdb.Label) *labelResponse {
	return &labelResponse{
		Links: map[string]string{
			"self": fmt.Sprintf("/api/v2/labels/%s", l.ID),
		},
		Label: *l,
	}
}

type labelsResponse struct {
	Links  map[string]string `json:"links"`
	Labels []*influxdb.Label `json:"labels"`
}

func newLabelsResponse(ls []*influxdb.Label) *labelsResponse {
	return &labelsResponse{
		Links: map[string]string{
			"self": fmt.Sprintf("/api/v2/labels"),
		},
		Labels: ls,
	}
}

// LabelBackend is all services and associated parameters required to construct
// label handlers.
type LabelBackend struct {
	Logger *zap.Logger
	influxdb.HTTPErrorHandler
	LabelService influxdb.LabelService
	ResourceType influxdb.ResourceType
}

// newGetLabelsHandler returns a handler func for a GET to /labels endpoints
func newGetLabelsHandler(b *LabelBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		req, err := decodeGetLabelMappingsRequest(ctx, r, b.ResourceType)
		if err != nil {
			b.HandleHTTPError(ctx, err, w)
			return
		}

		labels, err := b.LabelService.FindResourceLabels(ctx, req.filter)
		if err != nil {
			b.HandleHTTPError(ctx, err, w)
			return
		}

		if err := encodeResponse(ctx, w, http.StatusOK, newLabelsResponse(labels)); err != nil {
			logEncodingError(b.Logger, r, err)
			return
		}
	}
}

type getLabelMappingsRequest struct {
	filter influxdb.LabelMappingFilter
}

func decodeGetLabelMappingsRequest(ctx context.Context, r *http.Request, rt influxdb.ResourceType) (*getLabelMappingsRequest, error) {
	req := &getLabelMappingsRequest{}

	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "url missing id",
		}
	}

	var i influxdb.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}
	req.filter.ResourceID = i
	req.filter.ResourceType = rt

	return req, nil
}

// newPostLabelHandler returns a handler func for a POST to /labels endpoints
func newPostLabelHandler(b *LabelBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		req, err := decodePostLabelMappingRequest(ctx, r, b.ResourceType)
		if err != nil {
			b.HandleHTTPError(ctx, err, w)
			return
		}

		if err := req.Mapping.Validate(); err != nil {
			b.HandleHTTPError(ctx, err, w)
			return
		}

		if err := b.LabelService.CreateLabelMapping(ctx, &req.Mapping); err != nil {
			b.HandleHTTPError(ctx, err, w)
			return
		}

		label, err := b.LabelService.FindLabelByID(ctx, req.Mapping.LabelID)
		if err != nil {
			b.HandleHTTPError(ctx, err, w)
			return
		}

		if err := encodeResponse(ctx, w, http.StatusCreated, newLabelResponse(label)); err != nil {
			logEncodingError(b.Logger, r, err)
			return
		}
	}
}

type postLabelMappingRequest struct {
	Mapping influxdb.LabelMapping
}

func decodePostLabelMappingRequest(ctx context.Context, r *http.Request, rt influxdb.ResourceType) (*postLabelMappingRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "url missing id",
		}
	}

	var rid influxdb.ID
	if err := rid.DecodeFromString(id); err != nil {
		return nil, err
	}

	mapping := &influxdb.LabelMapping{}
	if err := json.NewDecoder(r.Body).Decode(mapping); err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "Invalid post label map request",
		}
	}

	mapping.ResourceID = rid
	mapping.ResourceType = rt

	if err := mapping.Validate(); err != nil {
		return nil, err
	}

	req := &postLabelMappingRequest{
		Mapping: *mapping,
	}

	return req, nil
}

// newDeleteLabelHandler returns a handler func for a DELETE to /labels endpoints
func newDeleteLabelHandler(b *LabelBackend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		req, err := decodeDeleteLabelMappingRequest(ctx, r)
		if err != nil {
			b.HandleHTTPError(ctx, err, w)
			return
		}

		mapping := &influxdb.LabelMapping{
			LabelID:      req.LabelID,
			ResourceID:   req.ResourceID,
			ResourceType: b.ResourceType,
		}

		if err := b.LabelService.DeleteLabelMapping(ctx, mapping); err != nil {
			b.HandleHTTPError(ctx, err, w)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type deleteLabelMappingRequest struct {
	ResourceID influxdb.ID
	LabelID    influxdb.ID
}

func decodeDeleteLabelMappingRequest(ctx context.Context, r *http.Request) (*deleteLabelMappingRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "url missing resource id",
		}
	}

	var rid influxdb.ID
	if err := rid.DecodeFromString(id); err != nil {
		return nil, err
	}

	id = params.ByName("lid")
	if id == "" {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Msg:  "label id is missing",
		}
	}

	var lid influxdb.ID
	if err := lid.DecodeFromString(id); err != nil {
		return nil, err
	}

	return &deleteLabelMappingRequest{
		LabelID:    lid,
		ResourceID: rid,
	}, nil
}

func labelIDPath(id influxdb.ID) string {
	return path.Join(labelsPath, id.String())
}

// FindLabelByID returns a single label by ID.
func (s *LabelService) FindLabelByID(ctx context.Context, id influxdb.ID) (*influxdb.Label, error) {
	u, err := NewURL(s.Addr, labelIDPath(id))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	SetToken(s.Token, req)

	hc := NewClient(u.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := CheckError(resp); err != nil {
		return nil, err
	}

	var lr labelResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return &lr.Label, nil
}

func (s *LabelService) FindLabels(ctx context.Context, filter influxdb.LabelFilter, opt ...influxdb.FindOptions) ([]*influxdb.Label, error) {
	return nil, nil
}

// FindResourceLabels returns a list of labels, derived from a label mapping filter.
func (s *LabelService) FindResourceLabels(ctx context.Context, filter influxdb.LabelMappingFilter) ([]*influxdb.Label, error) {
	url, err := NewURL(s.Addr, resourceIDPath(s.BasePath, filter.ResourceID))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	SetToken(s.Token, req)

	hc := NewClient(url.Scheme, s.InsecureSkipVerify)

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := CheckError(resp); err != nil {
		return nil, err
	}

	var r labelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return r.Labels, nil
}

// CreateLabel creates a new label.
func (s *LabelService) CreateLabel(ctx context.Context, l *influxdb.Label) error {
	u, err := NewURL(s.Addr, labelsPath)
	if err != nil {
		return err
	}

	octets, err := json.Marshal(l)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(octets))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	SetToken(s.Token, req)

	hc := NewClient(u.Scheme, s.InsecureSkipVerify)

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// TODO(jsternberg): Should this check for a 201 explicitly?
	if err := CheckError(resp); err != nil {
		return err
	}

	var lr labelResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return err
	}

	return nil
}

func (s *LabelService) CreateLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	if err := m.Validate(); err != nil {
		return err
	}

	url, err := NewURL(s.Addr, resourceIDPath(s.BasePath, m.ResourceID))
	if err != nil {
		return err
	}

	octets, err := json.Marshal(m)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(octets))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	SetToken(s.Token, req)

	hc := NewClient(url.Scheme, s.InsecureSkipVerify)

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := CheckError(resp); err != nil {
		return err
	}

	if err := json.NewDecoder(resp.Body).Decode(m); err != nil {
		return err
	}

	return nil
}

// UpdateLabel updates a label and returns the updated label.
func (s *LabelService) UpdateLabel(ctx context.Context, id influxdb.ID, upd influxdb.LabelUpdate) (*influxdb.Label, error) {
	u, err := NewURL(s.Addr, labelIDPath(id))
	if err != nil {
		return nil, err
	}

	octets, err := json.Marshal(upd)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", u.String(), bytes.NewReader(octets))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	SetToken(s.Token, req)

	hc := NewClient(u.Scheme, s.InsecureSkipVerify)

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := CheckError(resp); err != nil {
		return nil, err
	}

	var lr labelResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return &lr.Label, nil
}

// DeleteLabel removes a label by ID.
func (s *LabelService) DeleteLabel(ctx context.Context, id influxdb.ID) error {
	u, err := NewURL(s.Addr, labelIDPath(id))
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}
	SetToken(s.Token, req)

	hc := NewClient(u.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return CheckError(resp)
}

func (s *LabelService) DeleteLabelMapping(ctx context.Context, m *influxdb.LabelMapping) error {
	url, err := NewURL(s.Addr, labelNamePath(s.BasePath, m.ResourceID, m.LabelID))
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", url.String(), nil)
	if err != nil {
		return err
	}
	SetToken(s.Token, req)

	hc := NewClient(url.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return CheckError(resp)
}

func labelNamePath(basePath string, resourceID influxdb.ID, labelID influxdb.ID) string {
	return path.Join(basePath, resourceID.String(), "labels", labelID.String())
}
