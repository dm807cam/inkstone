package ui

import (
	"net/http/httptest"
	"testing"

	"github.com/ddvk/rmfakecloud/internal/model"
	"github.com/ddvk/rmfakecloud/internal/storage"
	"github.com/gin-gonic/gin"
)

// stubUserStorer returns a fixed user for GetUser; the rest of the interface is
// unused by getUser.
type stubUserStorer struct{ user *model.User }

func (s *stubUserStorer) GetUsers() ([]*model.User, error)   { return []*model.User{s.user}, nil }
func (s *stubUserStorer) GetUser(id string) (*model.User, error) {
	return s.user, nil
}
func (s *stubUserStorer) RegisterUser(u *model.User) error { return nil }
func (s *stubUserStorer) UpdateUser(u *model.User) error   { return nil }
func (s *stubUserStorer) RemoveUser(uid string) error      { return nil }

var _ storage.UserStorer = (*stubUserStorer)(nil)

func newGetUserContext(requestedUID, callerUID string, admin bool) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/users/"+requestedUID, nil)
	c.Params = gin.Params{{Key: useridParam, Value: requestedUID}}
	c.Set(userIDContextKey, callerUID)
	if admin {
		c.Set(AdminRole, true)
	}
	return c, rec
}

// TestGetUserRejectsNonAdminCrossUser guards #34 item 2: a non-admin caller
// requesting a *different* user must be rejected. Pre-fix the guard compared
// user.ID (loaded from the same param) against itself, so it never fired and a
// non-admin would receive the other user's record.
func TestGetUserRejectsNonAdminCrossUser(t *testing.T) {
	app := &ReactAppWrapper{userStorer: &stubUserStorer{user: &model.User{ID: "targetuser", Email: "t@example.com"}}}

	c, _ := newGetUserContext("targetuser", "attacker", false)
	app.getUser(c)

	if got := c.Writer.Status(); got != 401 {
		t.Fatalf("non-admin cross-user request should be 401, got %d", got)
	}
}

// TestGetUserAllowsAdminCrossUser verifies an admin can still query any user.
func TestGetUserAllowsAdminCrossUser(t *testing.T) {
	app := &ReactAppWrapper{userStorer: &stubUserStorer{user: &model.User{ID: "targetuser", Email: "t@example.com"}}}

	c, _ := newGetUserContext("targetuser", "adminuser", true)
	app.getUser(c)

	if got := c.Writer.Status(); got != 200 {
		t.Fatalf("admin cross-user request should be 200, got %d", got)
	}
}

// TestGetUserAllowsSelf verifies a non-admin can still read their own record.
func TestGetUserAllowsSelf(t *testing.T) {
	app := &ReactAppWrapper{userStorer: &stubUserStorer{user: &model.User{ID: "self", Email: "s@example.com"}}}

	c, _ := newGetUserContext("self", "self", false)
	app.getUser(c)

	if got := c.Writer.Status(); got != 200 {
		t.Fatalf("self request should be 200, got %d", got)
	}
}
