package http

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	platform "github.com/influxdata/influxdb"
	"github.com/pkg/errors"
)

const (
	// PlatformErrorCodeHeader shows the error code of platform error.
	PlatformErrorCodeHeader = "X-Platform-Error-Code"
)

// AuthzError is returned for authorization errors. When this error type is returned,
// the user can be presented with a generic "authorization failed" error, but
// the system can log the underlying AuthzError() so that operators have insight
// into what actually failed with authorization.
type AuthzError interface {
	error
	AuthzError() error
}

// CheckErrorStatus for status and any error in the response.
func CheckErrorStatus(code int, res *http.Response) error {
	err := CheckError(res)
	if err != nil {
		return err
	}

	if res.StatusCode != code {
		return fmt.Errorf("unexpected status code: %s", res.Status)
	}

	return nil
}

// CheckError reads the http.Response and returns an error if one exists.
// It will automatically recognize the errors returned by Influx services
// and decode the error into an internal error type. If the error cannot
// be determined in that way, it will create a generic error message.
//
// If there is no error, then this returns nil.
func CheckError(resp *http.Response) (err error) {
	switch resp.StatusCode / 100 {
	case 4, 5:
		// We will attempt to parse this error outside of this block.
	case 2:
		return nil
	default:
		// TODO(jsternberg): Figure out what to do here?
		return &platform.Error{
			Code: platform.EInternal,
			Msg:  fmt.Sprintf("unexpected status code: %d %s", resp.StatusCode, resp.Status),
		}
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// Assume JSON if there is no content-type.
		contentType = "application/json"
	}
	mediatype, _, _ := mime.ParseMediaType(contentType)

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return &platform.Error{
			Code: platform.EInternal,
			Msg:  err.Error(),
		}
	}

	switch mediatype {
	case "application/json":
		pe := new(platform.Error)

		parseErr := json.Unmarshal(buf.Bytes(), pe)
		if parseErr != nil {
			line, _ := buf.ReadString('\n')
			return errors.Wrap(stderrors.New(strings.TrimSuffix(line, "\n")), parseErr.Error())
		}
		return pe
	default:
		line, _ := buf.ReadString('\n')
		return stderrors.New(strings.TrimSuffix(line, "\n"))
	}
}

// ErrorHandler is the error handler in http package.
type ErrorHandler int

// HandleHTTPError encodes err with the appropriate status code and format,
// sets the X-Platform-Error-Code headers on the response.
// We're no longer using X-Influx-Error and X-Influx-Reference.
// and sets the response status to the corresponding status code.
func (h ErrorHandler) HandleHTTPError(ctx context.Context, err error, w http.ResponseWriter) {
	if err == nil {
		return
	}

	code := platform.ErrorCode(err)
	httpCode, ok := statusCodePlatformError[code]
	if !ok {
		httpCode = http.StatusBadRequest
	}
	w.Header().Set(PlatformErrorCodeHeader, code)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpCode)
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	e.Code = platform.ErrorCode(err)
	if err, ok := err.(*platform.Error); ok {
		e.Message = err.Error()
	} else {
		e.Message = "An internal error has occurred"
	}
	b, _ := json.Marshal(e)
	_, _ = w.Write(b)
}

// UnauthorizedError encodes a error message and status code for unauthorized access.
func UnauthorizedError(ctx context.Context, h platform.HTTPErrorHandler, w http.ResponseWriter) {
	h.HandleHTTPError(ctx, &platform.Error{
		Code: platform.EUnauthorized,
		Msg:  "unauthorized access",
	}, w)
}

// InactiveUserError encode a error message and status code for inactive users.
func InactiveUserError(ctx context.Context, h platform.HTTPErrorHandler, w http.ResponseWriter) {
	h.HandleHTTPError(ctx, &platform.Error{
		Code: platform.EForbidden,
		Msg:  "User is inactive",
	}, w)
}

// statusCodePlatformError is the map convert platform.Error to error
var statusCodePlatformError = map[string]int{
	platform.EInternal:            http.StatusInternalServerError,
	platform.EInvalid:             http.StatusBadRequest,
	platform.EUnprocessableEntity: http.StatusUnprocessableEntity,
	platform.EEmptyValue:          http.StatusBadRequest,
	platform.EConflict:            http.StatusUnprocessableEntity,
	platform.ENotFound:            http.StatusNotFound,
	platform.EUnavailable:         http.StatusServiceUnavailable,
	platform.EForbidden:           http.StatusForbidden,
	platform.ETooManyRequests:     http.StatusTooManyRequests,
	platform.EUnauthorized:        http.StatusUnauthorized,
	platform.EMethodNotAllowed:    http.StatusMethodNotAllowed,
	platform.ETooLarge:            http.StatusRequestEntityTooLarge,
}
