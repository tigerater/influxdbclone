package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/influxdata/influxdb"
	icontext "github.com/influxdata/influxdb/context"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// UserBackend is all services and associated parameters required to construct
// the UserHandler.
type UserBackend struct {
	influxdb.HTTPErrorHandler
	Logger                  *zap.Logger
	UserService             influxdb.UserService
	UserOperationLogService influxdb.UserOperationLogService
	PasswordsService        influxdb.PasswordsService
}

// NewUserBackend creates a UserBackend using information in the APIBackend.
func NewUserBackend(b *APIBackend) *UserBackend {
	return &UserBackend{
		HTTPErrorHandler:        b.HTTPErrorHandler,
		Logger:                  b.Logger.With(zap.String("handler", "user")),
		UserService:             b.UserService,
		UserOperationLogService: b.UserOperationLogService,
		PasswordsService:        b.PasswordsService,
	}
}

// UserHandler represents an HTTP API handler for users.
type UserHandler struct {
	*httprouter.Router
	influxdb.HTTPErrorHandler
	Logger                  *zap.Logger
	UserService             influxdb.UserService
	UserOperationLogService influxdb.UserOperationLogService
	PasswordsService        influxdb.PasswordsService
}

const (
	usersPath         = "/api/v2/users"
	mePath            = "/api/v2/me"
	mePasswordPath    = "/api/v2/me/password"
	usersIDPath       = "/api/v2/users/:id"
	usersPasswordPath = "/api/v2/users/:id/password"
	usersLogPath      = "/api/v2/users/:id/logs"
)

// NewUserHandler returns a new instance of UserHandler.
func NewUserHandler(b *UserBackend) *UserHandler {
	h := &UserHandler{
		Router:           NewRouter(b.HTTPErrorHandler),
		HTTPErrorHandler: b.HTTPErrorHandler,
		Logger:           b.Logger,

		UserService:             b.UserService,
		UserOperationLogService: b.UserOperationLogService,
		PasswordsService:        b.PasswordsService,
	}

	h.HandlerFunc("POST", usersPath, h.handlePostUser)
	h.HandlerFunc("GET", usersPath, h.handleGetUsers)
	h.HandlerFunc("GET", usersIDPath, h.handleGetUser)
	h.HandlerFunc("GET", usersLogPath, h.handleGetUserLog)
	h.HandlerFunc("PATCH", usersIDPath, h.handlePatchUser)
	h.HandlerFunc("DELETE", usersIDPath, h.handleDeleteUser)
	h.HandlerFunc("PUT", usersPasswordPath, h.handlePutUserPassword)

	h.HandlerFunc("GET", mePath, h.handleGetMe)
	h.HandlerFunc("PUT", mePasswordPath, h.handlePutUserPassword)

	return h
}

func (h *UserHandler) putPassword(ctx context.Context, w http.ResponseWriter, r *http.Request) (username string, err error) {

	req, err := decodePasswordResetRequest(ctx, r)
	if err != nil {
		return "", err
	}

	err = h.PasswordsService.CompareAndSetPassword(ctx, req.Username, req.PasswordOld, req.PasswordNew)
	if err != nil {
		return "", err
	}
	return req.Username, nil
}

// handlePutPassword is the HTTP handler for the PUT /api/v2/users/:id/password
func (h *UserHandler) handlePutUserPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("user update password request", zap.String("r", fmt.Sprint(r)))
	_, err := h.putPassword(ctx, w, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("user password updated")
	w.WriteHeader(http.StatusNoContent)
}

type passwordResetRequest struct {
	Username    string
	PasswordOld string
	PasswordNew string
}

type passwordResetRequestBody struct {
	Password string `json:"password"`
}

func decodePasswordResetRequest(ctx context.Context, r *http.Request) (*passwordResetRequest, error) {
	u, o, ok := r.BasicAuth()
	if !ok {
		return nil, fmt.Errorf("invalid basic auth")
	}

	pr := new(passwordResetRequestBody)
	err := json.NewDecoder(r.Body).Decode(pr)
	if err != nil {
		return nil, &influxdb.Error{
			Code: influxdb.EInvalid,
			Err:  err,
		}
	}

	return &passwordResetRequest{
		Username:    u,
		PasswordOld: o,
		PasswordNew: pr.Password,
	}, nil
}

// handlePostUser is the HTTP handler for the POST /api/v2/users route.
func (h *UserHandler) handlePostUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("user create request", zap.String("r", fmt.Sprint(r)))
	req, err := decodePostUserRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := h.UserService.CreateUser(ctx, req.User); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("user created", zap.String("user", fmt.Sprint(req.User)))

	if err := encodeResponse(ctx, w, http.StatusCreated, newUserResponse(req.User)); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
}

type postUserRequest struct {
	User *influxdb.User
}

func decodePostUserRequest(ctx context.Context, r *http.Request) (*postUserRequest, error) {
	b := &influxdb.User{}
	if err := json.NewDecoder(r.Body).Decode(b); err != nil {
		return nil, err
	}

	return &postUserRequest{
		User: b,
	}, nil
}

// handleGetMe is the HTTP handler for the GET /api/v2/me.
func (h *UserHandler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	a, err := icontext.GetAuthorizer(ctx)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	id := a.GetUserID()
	user, err := h.UserService.FindUserByID(ctx, id)

	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := encodeResponse(ctx, w, http.StatusOK, newUserResponse(user)); err != nil {
		h.HandleHTTPError(ctx, err, w)
	}
}

// handleGetUser is the HTTP handler for the GET /api/v2/users/:id route.
func (h *UserHandler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("user retrieve request", zap.String("r", fmt.Sprint(r)))
	req, err := decodeGetUserRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	b, err := h.UserService.FindUserByID(ctx, req.UserID)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("user retrieved", zap.String("user", fmt.Sprint(b)))

	if err := encodeResponse(ctx, w, http.StatusOK, newUserResponse(b)); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
}

type getUserRequest struct {
	UserID influxdb.ID
}

func decodeGetUserRequest(ctx context.Context, r *http.Request) (*getUserRequest, error) {
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

	req := &getUserRequest{
		UserID: i,
	}

	return req, nil
}

// handleDeleteUser is the HTTP handler for the DELETE /api/v2/users/:id route.
func (h *UserHandler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("user delete request", zap.String("r", fmt.Sprint(r)))
	req, err := decodeDeleteUserRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	if err := h.UserService.DeleteUser(ctx, req.UserID); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("user deleted", zap.String("userID", fmt.Sprint(req.UserID)))

	w.WriteHeader(http.StatusNoContent)
}

type deleteUserRequest struct {
	UserID influxdb.ID
}

func decodeDeleteUserRequest(ctx context.Context, r *http.Request) (*deleteUserRequest, error) {
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

	return &deleteUserRequest{
		UserID: i,
	}, nil
}

type usersResponse struct {
	Links map[string]string `json:"links"`
	Users []*userResponse   `json:"users"`
}

func (us usersResponse) ToInfluxdb() []*influxdb.User {
	users := make([]*influxdb.User, len(us.Users))
	for i := range us.Users {
		users[i] = &us.Users[i].User
	}
	return users
}

func newUsersResponse(users []*influxdb.User) *usersResponse {
	res := usersResponse{
		Links: map[string]string{
			"self": "/api/v2/users",
		},
		Users: []*userResponse{},
	}
	for _, user := range users {
		res.Users = append(res.Users, newUserResponse(user))
	}
	return &res
}

type userResponse struct {
	Links map[string]string `json:"links"`
	influxdb.User
}

func newUserResponse(u *influxdb.User) *userResponse {
	return &userResponse{
		Links: map[string]string{
			"self": fmt.Sprintf("/api/v2/users/%s", u.ID),
			"logs": fmt.Sprintf("/api/v2/users/%s/logs", u.ID),
		},
		User: *u,
	}
}

// handleGetUsers is the HTTP handler for the GET /api/v2/users route.
func (h *UserHandler) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("users retrieve request", zap.String("r", fmt.Sprint(r)))
	req, err := decodeGetUsersRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	users, _, err := h.UserService.FindUsers(ctx, req.filter)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("users retrieved", zap.String("users", fmt.Sprint(users)))

	err = encodeResponse(ctx, w, http.StatusOK, newUsersResponse(users))
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
}

type getUsersRequest struct {
	filter influxdb.UserFilter
}

func decodeGetUsersRequest(ctx context.Context, r *http.Request) (*getUsersRequest, error) {
	qp := r.URL.Query()
	req := &getUsersRequest{}

	if userID := qp.Get("id"); userID != "" {
		id, err := influxdb.IDFromString(userID)
		if err != nil {
			return nil, err
		}
		req.filter.ID = id
	}

	if name := qp.Get("name"); name != "" {
		req.filter.Name = &name
	}

	return req, nil
}

// handlePatchUser is the HTTP handler for the PATCH /api/v2/users/:id route.
func (h *UserHandler) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("user update request", zap.String("r", fmt.Sprint(r)))
	req, err := decodePatchUserRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	b, err := h.UserService.UpdateUser(ctx, req.UserID, req.Update)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("users updated", zap.String("user", fmt.Sprint(b)))

	if err := encodeResponse(ctx, w, http.StatusOK, newUserResponse(b)); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
}

type patchUserRequest struct {
	Update influxdb.UserUpdate
	UserID influxdb.ID
}

func decodePatchUserRequest(ctx context.Context, r *http.Request) (*patchUserRequest, error) {
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

	var upd influxdb.UserUpdate
	if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
		return nil, err
	}

	return &patchUserRequest{
		Update: upd,
		UserID: i,
	}, nil
}

// UserService connects to Influx via HTTP using tokens to manage users
type UserService struct {
	Addr               string
	Token              string
	InsecureSkipVerify bool
	// OpPrefix is the ops of not found error.
	OpPrefix string
}

// FindMe returns user information about the owner of the token
func (s *UserService) FindMe(ctx context.Context, id influxdb.ID) (*influxdb.User, error) {
	url, err := NewURL(s.Addr, mePath)
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

	var res userResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res.User, nil
}

// FindUserByID returns a single user by ID.
func (s *UserService) FindUserByID(ctx context.Context, id influxdb.ID) (*influxdb.User, error) {
	url, err := NewURL(s.Addr, userIDPath(id))
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

	var res userResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &res.User, nil
}

// FindUser returns the first user that matches filter.
func (s *UserService) FindUser(ctx context.Context, filter influxdb.UserFilter) (*influxdb.User, error) {
	if filter.ID == nil && filter.Name == nil {
		return nil, &influxdb.Error{
			Code: influxdb.ENotFound,
			Msg:  "user not found",
		}
	}
	users, n, err := s.FindUsers(ctx, filter)
	if err != nil {
		return nil, &influxdb.Error{
			Op:  s.OpPrefix + influxdb.OpFindUser,
			Err: err,
		}
	}

	if n == 0 {
		return nil, &influxdb.Error{
			Code: influxdb.ENotFound,
			Op:   s.OpPrefix + influxdb.OpFindUser,
			Msg:  "no results found",
		}
	}

	return users[0], nil
}

// FindUsers returns a list of users that match filter and the total count of matching users.
// Additional options provide pagination & sorting.
func (s *UserService) FindUsers(ctx context.Context, filter influxdb.UserFilter, opt ...influxdb.FindOptions) ([]*influxdb.User, int, error) {
	url, err := NewURL(s.Addr, usersPath)
	if err != nil {
		return nil, 0, err
	}

	query := url.Query()

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	if filter.ID != nil {
		query.Add("id", filter.ID.String())
	}
	if filter.Name != nil {
		query.Add("name", *filter.Name)
	}

	req.URL.RawQuery = query.Encode()
	SetToken(s.Token, req)

	hc := NewClient(url.Scheme, s.InsecureSkipVerify)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if err := CheckError(resp); err != nil {
		return nil, 0, err
	}

	var r usersResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, 0, err
	}

	us := r.ToInfluxdb()
	return us, len(us), nil
}

// CreateUser creates a new user and sets u.ID with the new identifier.
func (s *UserService) CreateUser(ctx context.Context, u *influxdb.User) error {
	url, err := NewURL(s.Addr, usersPath)
	if err != nil {
		return err
	}

	octets, err := json.Marshal(u)
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

	// TODO(jsternberg): Should this check for a 201 explicitly?
	if err := CheckError(resp); err != nil {
		return err
	}

	if err := json.NewDecoder(resp.Body).Decode(u); err != nil {
		return err
	}

	return nil
}

// UpdateUser updates a single user with changeset.
// Returns the new user state after update.
func (s *UserService) UpdateUser(ctx context.Context, id influxdb.ID, upd influxdb.UserUpdate) (*influxdb.User, error) {
	url, err := NewURL(s.Addr, userIDPath(id))
	if err != nil {
		return nil, err
	}

	octets, err := json.Marshal(upd)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PATCH", url.String(), bytes.NewReader(octets))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
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

	var res userResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &res.User, nil
}

// DeleteUser removes a user by ID.
func (s *UserService) DeleteUser(ctx context.Context, id influxdb.ID) error {
	url, err := NewURL(s.Addr, userIDPath(id))
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

	return CheckErrorStatus(http.StatusNoContent, resp)
}

func userIDPath(id influxdb.ID) string {
	return path.Join(usersPath, id.String())
}

// hanldeGetUserLog retrieves a user log by the users ID.
func (h *UserHandler) handleGetUserLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.Logger.Debug("user log retrieve request", zap.String("r", fmt.Sprint(r)))
	req, err := decodeGetUserLogRequest(ctx, r)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}

	log, _, err := h.UserOperationLogService.GetUserOperationLog(ctx, req.UserID, req.opts)
	if err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
	h.Logger.Debug("user log retrieved", zap.String("log", fmt.Sprint(log)))

	if err := encodeResponse(ctx, w, http.StatusOK, newUserLogResponse(req.UserID, log)); err != nil {
		h.HandleHTTPError(ctx, err, w)
		return
	}
}

type getUserLogRequest struct {
	UserID influxdb.ID
	opts   influxdb.FindOptions
}

func decodeGetUserLogRequest(ctx context.Context, r *http.Request) (*getUserLogRequest, error) {
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

	opts, err := decodeFindOptions(ctx, r)
	if err != nil {
		return nil, err
	}

	return &getUserLogRequest{
		UserID: i,
		opts:   *opts,
	}, nil
}

func newUserLogResponse(id influxdb.ID, es []*influxdb.OperationLogEntry) *operationLogResponse {
	logs := make([]*operationLogEntryResponse, 0, len(es))
	for _, e := range es {
		logs = append(logs, newOperationLogEntryResponse(e))
	}
	return &operationLogResponse{
		Links: map[string]string{
			"self": fmt.Sprintf("/api/v2/users/%s/logs", id),
		},
		Logs: logs,
	}
}
