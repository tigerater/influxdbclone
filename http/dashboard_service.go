package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	platform "github.com/influxdata/influxdb"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// DashboardBackend is all services and associated parameters required to construct
// the DashboardHandler.
type DashboardBackend struct {
	platform.HTTPErrorHandler
	Logger *zap.Logger

	DashboardService             platform.DashboardService
	DashboardOperationLogService platform.DashboardOperationLogService
	UserResourceMappingService   platform.UserResourceMappingService
	LabelService                 platform.LabelService
	UserService                  platform.UserService
}

// NewDashboardBackend creates a backend used by the dashboard handler.
func NewDashboardBackend(b *APIBackend) *DashboardBackend {
	return &DashboardBackend{
		HTTPErrorHandler: b.HTTPErrorHandler,
		Logger:           b.Logger.With(zap.String("handler", "dashboard")),

		DashboardService:             b.DashboardService,
		DashboardOperationLogService: b.DashboardOperationLogService,
		UserResourceMappingService:   b.UserResourceMappingService,
		LabelService:                 b.LabelService,
		UserService:                  b.UserService,
	}
}

// DashboardHandler is the handler for the dashboard service
type DashboardHandler struct {
	*httprouter.Router

	platform.HTTPErrorHandler
	Logger *zap.Logger

	DashboardService             platform.DashboardService
	DashboardOperationLogService platform.DashboardOperationLogService
	UserResourceMappingService   platform.UserResourceMappingService
	LabelService                 platform.LabelService
	UserService                  platform.UserService
}

const (
	dashboardsPath              = "/api/v2/dashboards"
	dashboardsIDPath            = "/api/v2/dashboards/:id"
	dashboardsIDCellsPath       = "/api/v2/dashboards/:id/cells"
	dashboardsIDCellsIDPath     = "/api/v2/dashboards/:id/cells/:cellID"
	dashboardsIDCellsIDViewPath = "/api/v2/dashboards/:id/cells/:cellID/view"
	dashboardsIDMembersPath     = "/api/v2/dashboards/:id/members"
	dashboardsIDLogPath         = "/api/v2/dashboards/:id/logs"
	dashboardsIDMembersIDPath   = "/api/v2/dashboards/:id/members/:userID"
	dashboardsIDOwnersPath      = "/api/v2/dashboards/:id/owners"
	dashboardsIDOwnersIDPath    = "/api/v2/dashboards/:id/owners/:userID"
	dashboardsIDLabelsPath      = "/api/v2/dashboards/:id/labels"
	dashboardsIDLabelsIDPath    = "/api/v2/dashboards/:id/labels/:lid"
)

// NewDashboardHandler returns a new instance of DashboardHandler.
func NewDashboardHandler(b *DashboardBackend) *DashboardHandler {
	h := &DashboardHandler{
		Router:           NewRouter(b.HTTPErrorHandler),
		HTTPErrorHandler: b.HTTPErrorHandler,
		Logger:           b.Logger,

		DashboardService:             b.DashboardService,
		DashboardOperationLogService: b.DashboardOperationLogService,
		UserResourceMappingService:   b.UserResourceMappingService,
		LabelService:                 b.LabelService,
		UserService:                  b.UserService,
	}

	h.HandlerFunc("POST", dashboardsPath, h.handlePostDashboard)
	h.HandlerFunc("GET", dashboardsPath, h.handleGetDashboards)
	h.HandlerFunc("GET", dashboardsIDPath, h.handleGetDashboard)
	h.HandlerFunc("GET", dashboardsIDLogPath, h.handleGetDashboardLog)
	h.HandlerFunc("DELETE", dashboardsIDPath, h.handleDeleteDashboard)
	h.HandlerFunc("PATCH", dashboardsIDPath, h.handlePatchDashboard)

	h.HandlerFunc("PUT", dashboardsIDCellsPath, h.handlePutDashboardCells)
	h.HandlerFunc("POST", dashboardsIDCellsPath, h.handlePostDashboardCell)
	h.HandlerFunc("DELETE", dashboardsIDCellsIDPath, h.handleDeleteDashboardCell)
	h.HandlerFunc("PATCH", dashboardsIDCellsIDPath, h.handlePatchDashboardCell)

	h.HandlerFunc("GET", dashboardsIDCellsIDViewPath, h.handleGetDashboardCellView)
	h.HandlerFunc("PATCH", dashboardsIDCellsIDViewPath, h.handlePatchDashboardCellView)

	memberBackend := MemberBackend{
		HTTPErrorHandler:           b.HTTPErrorHandler,
		Logger:                     b.Logger.With(zap.String("handler", "member")),
		ResourceType:               platform.DashboardsResourceType,
		UserType:                   platform.Member,
		UserResourceMappingService: b.UserResourceMappingService,
		UserService:                b.UserService,
	}
	h.HandlerFunc("POST", dashboardsIDMembersPath, newPostMemberHandler(memberBackend))
	h.HandlerFunc("GET", dashboardsIDMembersPath, newGetMembersHandler(memberBackend))
	h.HandlerFunc("DELETE", dashboardsIDMembersIDPath, newDeleteMemberHandler(memberBackend))

	ownerBackend := MemberBackend{
		HTTPErrorHandler:           b.HTTPErrorHandler,
		Logger:                     b.Logger.With(zap.String("handler", "member")),
		ResourceType:               platform.DashboardsResourceType,
		UserType:                   platform.Owner,
		UserResourceMappingService: b.UserResourceMappingService,
		UserService:                b.UserService,
	}
	h.HandlerFunc("POST", dashboardsIDOwnersPath, newPostMemberHandler(ownerBackend))
	h.HandlerFunc("GET", dashboardsIDOwnersPath, newGetMembersHandler(ownerBackend))
	h.HandlerFunc("DELETE", dashboardsIDOwnersIDPath, newDeleteMemberHandler(ownerBackend))

	labelBackend := &LabelBackend{
		HTTPErrorHandler: b.HTTPErrorHandler,
		Logger:           b.Logger.With(zap.String("handler", "label")),
		LabelService:     b.LabelService,
		ResourceType:     platform.DashboardsResourceType,
	}
	h.HandlerFunc("GET", dashboardsIDLabelsPath, newGetLabelsHandler(labelBackend))
	h.HandlerFunc("POST", dashboardsIDLabelsPath, newPostLabelHandler(labelBackend))
	h.HandlerFunc("DELETE", dashboardsIDLabelsIDPath, newDeleteLabelHandler(labelBackend))

	return h
}

type dashboardLinks struct {
	Self         string `json:"self"`
	Members      string `json:"members"`
	Owners       string `json:"owners"`
	Cells        string `json:"cells"`
	Logs         string `json:"logs"`
	Labels       string `json:"labels"`
	Organization string `json:"org"`
}

type dashboardResponse struct {
	ID             platform.ID             `json:"id,omitempty"`
	OrganizationID platform.ID             `json:"orgID,omitempty"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description"`
	Meta           platform.DashboardMeta  `json:"meta"`
	Cells          []dashboardCellResponse `json:"cells"`
	Labels         []platform.Label        `json:"labels"`
	Links          dashboardLinks          `json:"links"`
}

func (d dashboardResponse) toPlatform() *platform.Dashboard {
	var cells []*platform.Cell
	if len(d.Cells) > 0 {
		cells = make([]*platform.Cell, len(d.Cells))
	}
	for i := range d.Cells {
		cells[i] = d.Cells[i].toPlatform()
	}
	return &platform.Dashboard{
		ID:             d.ID,
		OrganizationID: d.OrganizationID,
		Name:           d.Name,
		Meta:           d.Meta,
		Cells:          cells,
	}
}

func newDashboardResponse(d *platform.Dashboard, labels []*platform.Label) dashboardResponse {
	res := dashboardResponse{
		Links: dashboardLinks{
			Self:         fmt.Sprintf("/api/v2/dashboards/%s", d.ID),
			Members:      fmt.Sprintf("/api/v2/dashboards/%s/members", d.ID),
			Owners:       fmt.Sprintf("/api/v2/dashboards/%s/owners", d.ID),
			Cells:        fmt.Sprintf("/api/v2/dashboards/%s/cells", d.ID),
			Logs:         fmt.Sprintf("/api/v2/dashboards/%s/logs", d.ID),
			Labels:       fmt.Sprintf("/api/v2/dashboards/%s/labels", d.ID),
			Organization: fmt.Sprintf("/api/v2/orgs/%s", d.OrganizationID),
		},
		ID:             d.ID,
		OrganizationID: d.OrganizationID,
		Name:           d.Name,
		Description:    d.Description,
		Meta:           d.Meta,
		Labels:         []platform.Label{},
		Cells:          []dashboardCellResponse{},
	}

	for _, l := range labels {
		res.Labels = append(res.Labels, *l)
	}

	for _, cell := range d.Cells {
		res.Cells = append(res.Cells, newDashboardCellResponse(d.ID, cell))
	}

	return res
}

type dashboardCellResponse struct {
	platform.Cell
	Links map[string]string `json:"links"`
}

func (c dashboardCellResponse) toPlatform() *platform.Cell {
	return &c.Cell
}

func newDashboardCellResponse(dashboardID platform.ID, c *platform.Cell) dashboardCellResponse {
	return dashboardCellResponse{
		Cell: *c,
		Links: map[string]string{
			"self": fmt.Sprintf("/api/v2/dashboards/%s/cells/%s", dashboardID, c.ID),
			"view": fmt.Sprintf("/api/v2/dashboards/%s/cells/%s/view", dashboardID, c.ID),
		},
	}
}

type dashboardCellsResponse struct {
	Cells []dashboardCellResponse `json:"cells"`
	Links map[string]string       `json:"links"`
}

func newDashboardCellsResponse(dashboardID platform.ID, cs []*platform.Cell) dashboardCellsResponse {
	res := dashboardCellsResponse{
		Cells: []dashboardCellResponse{},
		Links: map[string]string{
			"self": fmt.Sprintf("/api/v2/dashboards/%s/cells", dashboardID),
		},
	}

	for _, cell := range cs {
		res.Cells = append(res.Cells, newDashboardCellResponse(dashboardID, cell))
	}

	return res
}

type viewLinks struct {
	Self string `json:"self"`
}

type dashboardCellViewResponse struct {
	platform.View
	Links viewLinks `json:"links"`
}

func (r dashboardCellViewResponse) MarshalJSON() ([]byte, error) {
	props, err := platform.MarshalViewPropertiesJSON(r.Properties)
	if err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		platform.ViewContents
		Links      viewLinks       `json:"links"`
		Properties json.RawMessage `json:"properties"`
	}{
		ViewContents: r.ViewContents,
		Links:        r.Links,
		Properties:   props,
	})
}

func newDashboardCellViewResponse(dashID, cellID platform.ID, v *platform.View) dashboardCellViewResponse {
	return dashboardCellViewResponse{
		Links: viewLinks{
			Self: fmt.Sprintf("/api/v2/dashboards/%s/cells/%s", dashID, cellID),
		},
		View: *v,
	}
}

type operationLogResponse struct {
	Links map[string]string            `json:"links"`
	Logs  []*operationLogEntryResponse `json:"logs"`
}

func newDashboardLogResponse(id platform.ID, es []*platform.OperationLogEntry) *operationLogResponse {
	logs := make([]*operationLogEntryResponse, 0, len(es))
	for _, e := range es {
		logs = append(logs, newOperationLogEntryResponse(e))
	}
	return &operationLogResponse{
		Links: map[string]string{
			"self": fmt.Sprintf("/api/v2/dashboards/%s/logs", id),
		},
		Logs: logs,
	}
}

type operationLogEntryResponse struct {
	Links map[string]string `json:"links"`
	*platform.OperationLogEntry
}

func newOperationLogEntryResponse(e *platform.OperationLogEntry) *operationLogEntryResponse {
	links := map[string]string{}
	if e.UserID.Valid() {
		links["user"] = fmt.Sprintf("/api/v2/users/%s", e.UserID)
	}
	return &operationLogEntryResponse{
		Links:             links,
		OperationLogEntry: e,
	}
}

// handleGetDashboards returns all dashboards within the store.
func (h *DashboardHandler) handleGetDashboards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("get dashboards request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeGetDashboardsRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if req.ownerID != nil {
		filter := platform.UserResourceMappingFilter{
			UserID:       *req.ownerID,
			UserType:     platform.Owner,
			ResourceType: platform.DashboardsResourceType,
		}

		mappings, _, err := h.UserResourceMappingService.FindUserResourceMappings(ctx, filter)
		if err != nil {
			h.HandleHTTPError(ctx, &platform.Error{
				Code: platform.EInternal,
				Msg:  "Error loading dashboard owners",
				Err:  err,
			}, w)
			return
		}

		for _, mapping := range mappings {
			req.filter.IDs = append(req.filter.IDs, &mapping.ResourceID)
		}
	}

	dashboards, _, err := h.DashboardService.FindDashboards(ctx, req.filter, req.opts)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.Logger.Debug("dashboards retrieved", zap.String("dashboards", fmt.Sprint(dashboards)))

	if err := encodeResponse(ctx, w, http.StatusOK, newGetDashboardsResponse(ctx, dashboards, req.filter, req.opts, h.LabelService)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type getDashboardsRequest struct {
	filter  platform.DashboardFilter
	opts    platform.FindOptions
	ownerID *platform.ID
}

func decodeGetDashboardsRequest(ctx context.Context, r *http.Request) (*getDashboardsRequest, error) {
	qp := r.URL.Query()
	req := &getDashboardsRequest{}

	opts, err := decodeFindOptions(ctx, r)
	if err != nil {
		return nil, err
	}
	req.opts = *opts

	initialID := platform.InvalidID()
	if ids, ok := qp["id"]; ok {
		for _, id := range ids {
			i := initialID
			if err := i.DecodeFromString(id); err != nil {
				return nil, err
			}
			req.filter.IDs = append(req.filter.IDs, &i)
		}
	} else if ownerID := qp.Get("ownerID"); ownerID != "" {
		req.ownerID = &initialID
		if err := req.ownerID.DecodeFromString(ownerID); err != nil {
			return nil, err
		}
	} else if orgID := qp.Get("orgID"); orgID != "" {
		id := platform.InvalidID()
		if err := id.DecodeFromString(orgID); err != nil {
			return nil, err
		}
		req.filter.OrganizationID = &id
	} else if org := qp.Get("org"); org != "" {
		req.filter.Organization = &org
	}

	return req, nil
}

type getDashboardsResponse struct {
	Links      *platform.PagingLinks `json:"links"`
	Dashboards []dashboardResponse   `json:"dashboards"`
}

func (d getDashboardsResponse) toPlatform() []*platform.Dashboard {
	res := make([]*platform.Dashboard, len(d.Dashboards))
	for i := range d.Dashboards {
		res[i] = d.Dashboards[i].toPlatform()
	}
	return res
}

func newGetDashboardsResponse(ctx context.Context, dashboards []*platform.Dashboard, filter platform.DashboardFilter, opts platform.FindOptions, labelService platform.LabelService) getDashboardsResponse {
	res := getDashboardsResponse{
		Links:      newPagingLinks(dashboardsPath, opts, filter, len(dashboards)),
		Dashboards: make([]dashboardResponse, 0, len(dashboards)),
	}

	for _, dashboard := range dashboards {
		if dashboard != nil {
			labels, _ := labelService.FindResourceLabels(ctx, platform.LabelMappingFilter{ResourceID: dashboard.ID})
			res.Dashboards = append(res.Dashboards, newDashboardResponse(dashboard, labels))
		}
	}

	return res
}

// handlePostDashboard creates a new dashboard.
func (h *DashboardHandler) handlePostDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("create dashboard request", zap.String("r", fmt.Sprint(r)))

	req, err := decodePostDashboardRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	if err := h.DashboardService.CreateDashboard(ctx, req.Dashboard); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusCreated, newDashboardResponse(req.Dashboard, []*platform.Label{})); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type postDashboardRequest struct {
	Dashboard *platform.Dashboard
}

func decodePostDashboardRequest(ctx context.Context, r *http.Request) (*postDashboardRequest, error) {
	c := &platform.Dashboard{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		return nil, err
	}
	return &postDashboardRequest{
		Dashboard: c,
	}, nil
}

// hanldeGetDashboard retrieves a dashboard by ID.
func (h *DashboardHandler) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("get dashboard request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeGetDashboardRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	dashboard, err := h.DashboardService.FindDashboardByID(ctx, req.DashboardID)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	labels, err := h.LabelService.FindResourceLabels(ctx, platform.LabelMappingFilter{ResourceID: dashboard.ID})
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.Logger.Debug("dashboard retrieved", zap.String("dashboard", fmt.Sprint(dashboard)))

	if err := encodeResponse(ctx, w, http.StatusOK, newDashboardResponse(dashboard, labels)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type getDashboardRequest struct {
	DashboardID platform.ID
}

func decodeGetDashboardRequest(ctx context.Context, r *http.Request) (*getDashboardRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "url missing id",
		}
	}

	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	return &getDashboardRequest{
		DashboardID: i,
	}, nil
}

// hanldeGetDashboardLog retrieves a dashboard log by the dashboards ID.
func (h *DashboardHandler) handleGetDashboardLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("get dashboard log request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeGetDashboardLogRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	log, _, err := h.DashboardOperationLogService.GetDashboardOperationLog(ctx, req.DashboardID, req.opts)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.Logger.Debug("dashboard log retrieved", zap.String("log", fmt.Sprint(log)))

	if err := encodeResponse(ctx, w, http.StatusOK, newDashboardLogResponse(req.DashboardID, log)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type getDashboardLogRequest struct {
	DashboardID platform.ID
	opts        platform.FindOptions
}

func decodeGetDashboardLogRequest(ctx context.Context, r *http.Request) (*getDashboardLogRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "url missing id",
		}
	}

	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	opts, err := decodeFindOptions(ctx, r)
	if err != nil {
		return nil, err
	}

	return &getDashboardLogRequest{
		DashboardID: i,
		opts:        *opts,
	}, nil
}

// handleDeleteDashboard removes a dashboard by ID.
func (h *DashboardHandler) handleDeleteDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("delete dashboard request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeDeleteDashboardRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := h.DashboardService.DeleteDashboard(ctx, req.DashboardID); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.Logger.Debug("dashboard deleted", zap.String("dashboardID", req.DashboardID.String()))

	w.WriteHeader(http.StatusNoContent)
}

type deleteDashboardRequest struct {
	DashboardID platform.ID
}

func decodeDeleteDashboardRequest(ctx context.Context, r *http.Request) (*deleteDashboardRequest, error) {
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "url missing id",
		}
	}

	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	return &deleteDashboardRequest{
		DashboardID: i,
	}, nil
}

// handlePatchDashboard updates a dashboard.
func (h *DashboardHandler) handlePatchDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("update dashboard request", zap.String("r", fmt.Sprint(r)))

	req, err := decodePatchDashboardRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	dashboard, err := h.DashboardService.UpdateDashboard(ctx, req.DashboardID, req.Upd)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	labels, err := h.LabelService.FindResourceLabels(ctx, platform.LabelMappingFilter{ResourceID: dashboard.ID})
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.Logger.Debug("dashboard updated", zap.String("dashboard", fmt.Sprint(dashboard)))

	if err := encodeResponse(ctx, w, http.StatusOK, newDashboardResponse(dashboard, labels)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type patchDashboardRequest struct {
	DashboardID platform.ID
	Upd         platform.DashboardUpdate
}

func decodePatchDashboardRequest(ctx context.Context, r *http.Request) (*patchDashboardRequest, error) {
	req := &patchDashboardRequest{}
	upd := platform.DashboardUpdate{}
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Err:  err,
		}
	}
	req.Upd = upd

	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "url missing id",
		}
	}
	var i platform.ID
	if err := i.DecodeFromString(id); err != nil {
		return nil, err
	}

	req.DashboardID = i

	if err := req.Valid(); err != nil {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Err:  err,
		}
	}

	return req, nil
}

// Valid validates that the dashboard ID is non zero valued and update has expected values set.
func (r *patchDashboardRequest) Valid() error {
	if !r.DashboardID.Valid() {
		return &platform.Error{
			Code: platform.EInvalid,
			Msg:  "missing dashboard ID",
		}
	}

	if pe := r.Upd.Valid(); pe != nil {
		return pe
	}
	return nil
}

type postDashboardCellRequest struct {
	dashboardID platform.ID
	*platform.CellProperty
	UsingView *platform.ID `json:"usingView"`
	Name      *string      `json:"name"`
}

func decodePostDashboardCellRequest(ctx context.Context, r *http.Request) (*postDashboardCellRequest, error) {
	req := &postDashboardCellRequest{}
	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "url missing id",
		}
	}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "bad request json body",
			Err:  err,
		}
	}

	if err := req.dashboardID.DecodeFromString(id); err != nil {
		return nil, err
	}

	return req, nil
}

// handlePostDashboardCell creates a dashboard cell.
func (h *DashboardHandler) handlePostDashboardCell(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("create dashboard cell request", zap.String("r", fmt.Sprint(r)))

	req, err := decodePostDashboardCellRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	cell := new(platform.Cell)

	opts := new(platform.AddDashboardCellOptions)
	if req.UsingView != nil || req.Name != nil {
		opts.View = new(platform.View)
		if req.UsingView != nil {
			// load the view
			opts.View, err = h.DashboardService.GetDashboardCellView(ctx, req.dashboardID, *req.UsingView)
			if err != nil {
				h.HandleHTTPError(ctx, err, w)
				return
			}
		}
		if req.Name != nil {
			opts.View.Name = *req.Name
		}
	} else if req.CellProperty == nil {
		h.HandleHTTPError(ctx, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "req body is empty",
		}, w)
		return
	}

	if req.CellProperty != nil {
		cell.CellProperty = *req.CellProperty
	}

	if err := h.DashboardService.AddDashboardCell(ctx, req.dashboardID, cell, *opts); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.Logger.Debug("dashboard cell created", zap.String("dashboardID", req.dashboardID.String()), zap.String("cell", fmt.Sprint(cell)))

	if err := encodeResponse(ctx, w, http.StatusCreated, newDashboardCellResponse(req.dashboardID, cell)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type putDashboardCellRequest struct {
	dashboardID platform.ID
	cells       []*platform.Cell
}

func decodePutDashboardCellRequest(ctx context.Context, r *http.Request) (*putDashboardCellRequest, error) {
	req := &putDashboardCellRequest{}

	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "url missing id",
		}
	}
	if err := req.dashboardID.DecodeFromString(id); err != nil {
		return nil, err
	}

	req.cells = []*platform.Cell{}
	if err := json.NewDecoder(r.Body).Decode(&req.cells); err != nil {
		return nil, err
	}

	return req, nil
}

// handlePutDashboardCells replaces a dashboards cells.
func (h *DashboardHandler) handlePutDashboardCells(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("replace dashboard cell request", zap.String("r", fmt.Sprint(r)))

	req, err := decodePutDashboardCellRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := h.DashboardService.ReplaceDashboardCells(ctx, req.dashboardID, req.cells); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.Logger.Debug("dashboard cell replaced", zap.String("dashboardID", req.dashboardID.String()), zap.String("cells", fmt.Sprint(req.cells)))

	if err := encodeResponse(ctx, w, http.StatusCreated, newDashboardCellsResponse(req.dashboardID, req.cells)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type deleteDashboardCellRequest struct {
	dashboardID platform.ID
	cellID      platform.ID
}

func decodeDeleteDashboardCellRequest(ctx context.Context, r *http.Request) (*deleteDashboardCellRequest, error) {
	req := &deleteDashboardCellRequest{}

	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "url missing id",
		}
	}
	if err := req.dashboardID.DecodeFromString(id); err != nil {
		return nil, err
	}

	cellID := params.ByName("cellID")
	if cellID == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "url missing cellID",
		}
	}
	if err := req.cellID.DecodeFromString(cellID); err != nil {
		return nil, err
	}

	return req, nil
}

type getDashboardCellViewRequest struct {
	dashboardID platform.ID
	cellID      platform.ID
}

func decodeGetDashboardCellViewRequest(ctx context.Context, r *http.Request) (*getDashboardCellViewRequest, error) {
	req := &getDashboardCellViewRequest{}

	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, platform.NewError(platform.WithErrorMsg("url missing id"), platform.WithErrorCode(platform.EInvalid))
	}
	if err := req.dashboardID.DecodeFromString(id); err != nil {
		return nil, err
	}

	cellID := params.ByName("cellID")
	if cellID == "" {
		return nil, platform.NewError(platform.WithErrorMsg("url missing cellID"), platform.WithErrorCode(platform.EInvalid))
	}
	if err := req.cellID.DecodeFromString(cellID); err != nil {
		return nil, err
	}

	return req, nil
}

func (h *DashboardHandler) handleGetDashboardCellView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("get dashboard cell view request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeGetDashboardCellViewRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	view, err := h.DashboardService.GetDashboardCellView(ctx, req.dashboardID, req.cellID)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.Logger.Debug("dashboard cell view retrieved", zap.String("dashboardID", req.dashboardID.String()), zap.String("cellID", req.cellID.String()), zap.String("view", fmt.Sprint(view)))

	if err := encodeResponse(ctx, w, http.StatusOK, newDashboardCellViewResponse(req.dashboardID, req.cellID, view)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

type patchDashboardCellViewRequest struct {
	dashboardID platform.ID
	cellID      platform.ID
	upd         platform.ViewUpdate
}

func decodePatchDashboardCellViewRequest(ctx context.Context, r *http.Request) (*patchDashboardCellViewRequest, error) {
	req := &patchDashboardCellViewRequest{}

	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, platform.NewError(platform.WithErrorMsg("url missing id"), platform.WithErrorCode(platform.EInvalid))
	}
	if err := req.dashboardID.DecodeFromString(id); err != nil {
		return nil, err
	}

	cellID := params.ByName("cellID")
	if cellID == "" {
		return nil, platform.NewError(platform.WithErrorMsg("url missing cellID"), platform.WithErrorCode(platform.EInvalid))
	}
	if err := req.cellID.DecodeFromString(cellID); err != nil {
		return nil, err
	}

	if err := json.NewDecoder(r.Body).Decode(&req.upd); err != nil {
		return nil, err
	}

	return req, nil
}

func (h *DashboardHandler) handlePatchDashboardCellView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("update dashboard cell view request", zap.String("r", fmt.Sprint(r)))

	req, err := decodePatchDashboardCellViewRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	view, err := h.DashboardService.UpdateDashboardCellView(ctx, req.dashboardID, req.cellID, req.upd)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("dashboard cell view updated", zap.String("dashboardID", req.dashboardID.String()), zap.String("cellID", req.cellID.String()), zap.String("view", fmt.Sprint(view)))

	if err := encodeResponse(ctx, w, http.StatusOK, newDashboardCellViewResponse(req.dashboardID, req.cellID, view)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

// handleDeleteDashboardCell deletes a dashboard cell.
func (h *DashboardHandler) handleDeleteDashboardCell(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("delete dashboard cell request", zap.String("r", fmt.Sprint(r)))

	req, err := decodeDeleteDashboardCellRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	if err := h.DashboardService.RemoveDashboardCell(ctx, req.dashboardID, req.cellID); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("dashboard cell deleted", zap.String("dashboardID", req.dashboardID.String()), zap.String("cellID", req.cellID.String()))

	w.WriteHeader(http.StatusNoContent)
}

type patchDashboardCellRequest struct {
	dashboardID platform.ID
	cellID      platform.ID
	upd         platform.CellUpdate
}

func decodePatchDashboardCellRequest(ctx context.Context, r *http.Request) (*patchDashboardCellRequest, error) {
	req := &patchDashboardCellRequest{}

	params := httprouter.ParamsFromContext(ctx)
	id := params.ByName("id")
	if id == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "url missing id",
		}
	}
	if err := req.dashboardID.DecodeFromString(id); err != nil {
		return nil, err
	}

	cellID := params.ByName("cellID")
	if cellID == "" {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Msg:  "cannot provide empty cell id",
		}
	}
	if err := req.cellID.DecodeFromString(cellID); err != nil {
		return nil, err
	}

	if err := json.NewDecoder(r.Body).Decode(&req.upd); err != nil {
		return nil, &platform.Error{
			Code: platform.EInvalid,
			Err:  err,
		}
	}

	if pe := req.upd.Valid(); pe != nil {
		return nil, pe
	}

	return req, nil
}

// handlePatchDashboardCell updates a dashboard cell.
func (h *DashboardHandler) handlePatchDashboardCell(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("update dashboard cell request", zap.String("r", fmt.Sprint(r)))

	req, err := decodePatchDashboardCellRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	cell, err := h.DashboardService.UpdateDashboardCell(ctx, req.dashboardID, req.cellID, req.upd)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	h.Logger.Debug("dashboard cell updated", zap.String("dashboardID", req.dashboardID.String()), zap.String("cell", fmt.Sprint(cell)))

	if err := encodeResponse(ctx, w, http.StatusOK, newDashboardCellResponse(req.dashboardID, cell)); err != nil {
		logEncodingError(h.Logger, r, err)
		return
	}
}

// DashboardService is a dashboard service over HTTP to the influxdb server.
type DashboardService struct {
	Addr               string
	Token              string
	InsecureSkipVerify bool
	// OpPrefix is the op prefix for certain errors op.
	OpPrefix string
}

// FindDashboardByID returns a single dashboard by ID.
func (s *DashboardService) FindDashboardByID(ctx context.Context, id platform.ID) (*platform.Dashboard, error) {
	path := dashboardIDPath(id)
	url, err := NewURL(s.Addr, path)
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

	var dr dashboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return nil, err
	}

	dashboard := dr.toPlatform()
	return dashboard, nil
}

// FindDashboards returns a list of dashboards that match filter and the total count of matching dashboards.
// Additional options provide pagination & sorting.
func (s *DashboardService) FindDashboards(ctx context.Context, filter platform.DashboardFilter, opts platform.FindOptions) ([]*platform.Dashboard, int, error) {
	dashboards := []*platform.Dashboard{}
	url, err := NewURL(s.Addr, dashboardsPath)

	if err != nil {
		return dashboards, 0, err
	}

	qp := url.Query()
	for _, id := range filter.IDs {
		qp.Add("id", id.String())
	}
	if filter.OrganizationID != nil {
		qp.Add("orgID", filter.OrganizationID.String())
	}
	if filter.Organization != nil {
		qp.Add("org", *filter.Organization)
	}
	for k, vs := range opts.QueryParams() {
		for _, v := range vs {
			qp.Add(k, v)
		}
	}
	url.RawQuery = qp.Encode()

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return dashboards, 0, err
	}

	SetToken(s.Token, req)
	hc := NewClient(url.Scheme, s.InsecureSkipVerify)

	resp, err := hc.Do(req)
	if err != nil {
		return dashboards, 0, err
	}
	defer resp.Body.Close()

	if err := CheckError(resp); err != nil {
		return dashboards, 0, err
	}

	var dr getDashboardsResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return dashboards, 0, err
	}

	dashboards = dr.toPlatform()
	return dashboards, len(dashboards), nil
}

// CreateDashboard creates a new dashboard and sets b.ID with the new identifier.
func (s *DashboardService) CreateDashboard(ctx context.Context, d *platform.Dashboard) error {
	url, err := NewURL(s.Addr, dashboardsPath)
	if err != nil {
		return err
	}

	b, err := json.Marshal(d)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(b))
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

	if err := json.NewDecoder(resp.Body).Decode(d); err != nil {
		return err
	}

	return nil
}

// UpdateDashboard updates a single dashboard with changeset.
// Returns the new dashboard state after update.
func (s *DashboardService) UpdateDashboard(ctx context.Context, id platform.ID, upd platform.DashboardUpdate) (*platform.Dashboard, error) {
	//op := s.OpPrefix + platform.OpUpdateDashboard
	u, err := NewURL(s.Addr, dashboardIDPath(id))
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(upd)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", u.String(), bytes.NewReader(b))
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

	var d platform.Dashboard
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	if len(d.Cells) == 0 {
		d.Cells = nil
	}

	return &d, nil
}

// DeleteDashboard removes a dashboard by ID.
func (s *DashboardService) DeleteDashboard(ctx context.Context, id platform.ID) error {
	u, err := NewURL(s.Addr, dashboardIDPath(id))
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

// AddDashboardCell adds a cell to a dashboard.
func (s *DashboardService) AddDashboardCell(ctx context.Context, id platform.ID, c *platform.Cell, opts platform.AddDashboardCellOptions) error {
	url, err := NewURL(s.Addr, cellPath(id))
	if err != nil {
		return err
	}

	// fixme > in case c does not contain a valid ID this errors out
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(b))
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

	// TODO (goller): deal with the dashboard cell options
	return json.NewDecoder(resp.Body).Decode(c)
}

// RemoveDashboardCell removes a dashboard.
func (s *DashboardService) RemoveDashboardCell(ctx context.Context, dashboardID, cellID platform.ID) error {
	u, err := NewURL(s.Addr, dashboardCellIDPath(dashboardID, cellID))
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

// UpdateDashboardCell replaces the dashboard cell with the provided ID.
func (s *DashboardService) UpdateDashboardCell(ctx context.Context, dashboardID, cellID platform.ID, upd platform.CellUpdate) (*platform.Cell, error) {
	op := s.OpPrefix + platform.OpUpdateDashboardCell
	if err := upd.Valid(); err != nil {
		return nil, &platform.Error{
			Op:  op,
			Err: err,
		}
	}

	u, err := NewURL(s.Addr, dashboardCellIDPath(dashboardID, cellID))
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(upd)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", u.String(), bytes.NewReader(b))
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

	var c platform.Cell
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

// GetDashboardCellView retrieves the view for a dashboard cell.
func (s *DashboardService) GetDashboardCellView(ctx context.Context, dashboardID, cellID platform.ID) (*platform.View, error) {
	u, err := NewURL(s.Addr, cellViewPath(dashboardID, cellID))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", u.String(), nil)
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

	res := dashboardCellViewResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &res.View, nil
}

// UpdateDashboardCellView updates the view for a dashboard cell.
func (s *DashboardService) UpdateDashboardCellView(ctx context.Context, dashboardID, cellID platform.ID, upd platform.ViewUpdate) (*platform.View, error) {
	u, err := NewURL(s.Addr, cellViewPath(dashboardID, cellID))
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(upd)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", u.String(), bytes.NewReader(b))
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

	res := dashboardCellViewResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &res.View, nil
}

// ReplaceDashboardCells replaces all cells in a dashboard
func (s *DashboardService) ReplaceDashboardCells(ctx context.Context, id platform.ID, cs []*platform.Cell) error {
	u, err := NewURL(s.Addr, cellPath(id))
	if err != nil {
		return err
	}

	// TODO(goller): I think this should be {"cells":[]}
	b, err := json.Marshal(cs)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", u.String(), bytes.NewReader(b))
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

	if err := CheckError(resp); err != nil {
		return err
	}

	cells := dashboardCellsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&cells); err != nil {
		return err
	}

	return nil
}

func dashboardIDPath(id platform.ID) string {
	return path.Join(dashboardsPath, id.String())
}

func cellPath(id platform.ID) string {
	return path.Join(dashboardIDPath(id), "cells")
}

func cellViewPath(dashboardID, cellID platform.ID) string {
	return path.Join(dashboardIDPath(dashboardID), "cells", cellID.String(), "view")
}

func dashboardCellIDPath(id platform.ID, cellID platform.ID) string {
	return path.Join(cellPath(id), cellID.String())
}
