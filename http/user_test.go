package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/inmem"
	"github.com/influxdata/influxdb/mock"
	"github.com/influxdata/influxdb/pkg/testttp"
	platformtesting "github.com/influxdata/influxdb/testing"
	"go.uber.org/zap/zaptest"
)

// NewMockUserBackend returns a UserBackend with mock services.
func NewMockUserBackend(t *testing.T) *UserBackend {
	return &UserBackend{
		log:                     zaptest.NewLogger(t),
		UserService:             mock.NewUserService(),
		UserOperationLogService: mock.NewUserOperationLogService(),
		PasswordsService:        mock.NewPasswordsService(),
		HTTPErrorHandler:        ErrorHandler(0),
	}
}

func initUserService(f platformtesting.UserFields, t *testing.T) (platform.UserService, string, func()) {
	t.Helper()
	svc := inmem.NewService()
	svc.IDGenerator = f.IDGenerator

	ctx := context.Background()
	for _, u := range f.Users {
		if err := svc.PutUser(ctx, u); err != nil {
			t.Fatalf("failed to populate users")
		}
	}

	userBackend := NewMockUserBackend(t)
	userBackend.HTTPErrorHandler = ErrorHandler(0)
	userBackend.UserService = svc
	handler := NewUserHandler(zaptest.NewLogger(t), userBackend)
	server := httptest.NewServer(handler)

	httpClient, err := NewHTTPClient(server.URL, "", false)
	if err != nil {
		t.Fatal(err)
	}

	client := UserService{
		Client:   httpClient,
		OpPrefix: inmem.OpPrefix,
	}

	return &client, inmem.OpPrefix, server.Close
}

func TestUserService(t *testing.T) {
	t.Parallel()
	platformtesting.UserService(initUserService, t)
}

func TestUserHandler_SettingPassword(t *testing.T) {
	be := NewMockUserBackend(t)
	fakePassSVC := mock.NewPasswordsService()

	userID := platform.ID(1)
	fakePassSVC.SetPasswordFn = func(_ context.Context, id platform.ID, newPass string) error {
		if id != userID {
			return errors.New("unexpected id: " + id.String())
		}
		if newPass == "" {
			return errors.New("no password provided")
		}
		return nil
	}
	be.PasswordsService = fakePassSVC

	h := NewUserHandler(zaptest.NewLogger(t), be)

	addr := path.Join("/api/v2/users", userID.String(), "/password")

	testttp.
		PostJSON(t, addr, passwordSetRequest{Password: "newpassword"}).
		Do(h).
		ExpectStatus(http.StatusNoContent)
}
