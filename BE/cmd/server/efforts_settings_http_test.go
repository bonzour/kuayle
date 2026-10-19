package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/handler"
	mw "github.com/kuayle/kuayle-backend/internal/middleware"
	"github.com/kuayle/kuayle-backend/internal/repository"
	"github.com/kuayle/kuayle-backend/internal/service"
	jwtpkg "github.com/kuayle/kuayle-backend/pkg/jwt"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestEffortSettingsHTTP(t *testing.T) {
	db := newEffortHTTPDatabase(t)
	ws, owner, admin, member, guest := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	for _, user := range []uuid.UUID{owner, admin, member, guest} {
		exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'EP01','hash')`, user, user.String()+"@example.test")
	}
	exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'EP01',$2,$3)`, ws, ws.String(), owner)
	for user, role := range map[uuid.UUID]string{owner: "owner", admin: "admin", member: "member", guest: "guest"} {
		exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3)`, ws, user, role)
	}
	workspaces := repository.NewWorkspaceRepository(db)
	tokens := repository.NewPersonalAccessTokenRepository(db)
	e := echo.New()
	middleware := testMiddleware(mw.Auth("ep01-http-secret", tokens))
	middleware.workspaceMembership = mw.WorkspaceMembership(workspaces)
	registerRoutes(e, &appHandlers{effort: handler.NewEffortHandler(service.NewEffortService(repository.NewEffortRepository(db)), repository.NewIssueRepository(db))}, middleware)
	server := httptest.NewServer(e)
	defer server.Close()
	path := "/api/workspaces/" + ws.String() + "/efforts/settings"
	session := func(user uuid.UUID) string {
		value, err := jwtpkg.GenerateAccessToken(user, "ep01-http-secret")
		require.NoError(t, err)
		return value
	}
	pat := func(scopes []string) string {
		raw, hash, prefix, err := domain.GenerateToken()
		require.NoError(t, err)
		require.NoError(t, tokens.Create(context.Background(), &domain.PersonalAccessToken{ID: uuid.New(), UserID: owner, Name: "EP01", TokenHash: hash, TokenPrefix: prefix, Scopes: scopes}))
		return raw
	}
	request := func(token, method, body string, status int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		if strings.HasPrefix(token, domain.TokenPrefix) {
			req.Header.Set("Authorization", "Bearer "+token)
		} else if token != "" {
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		}
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, status, resp.StatusCode, "%s: %s", method, data)
		return data
	}
	payload := func(id string, rev int, enabled bool, timezone string) string {
		return fmt.Sprintf(`{"request_id":%q,"expected_revision":%d,"enabled":%t,"timezone":%q}`, id, rev, enabled, timezone)
	}
	ownerJWT, adminJWT := session(owner), session(admin)
	var settings dto.EffortSettingsResponse
	require.NoError(t, json.Unmarshal(request(session(guest), "GET", "", 200), &settings))
	require.Equal(t, domain.EffortSettings{Enabled: true, Timezone: "Europe/Rome", Revision: 0}, settings.EffortSettings)
	_, err := domain.ParseEffortDate(string(settings.Today))
	require.NoError(t, err)
	valid := payload(uuid.NewString(), 0, false, "UTC")
	request("", "PUT", valid, 401)
	for _, token := range []string{session(member), session(guest), pat([]string{}), pat([]string{domain.PermEffortsRead}), pat(domain.RolePermissions[domain.RoleOwner])} {
		request(token, "PUT", valid, 403)
	}
	for _, body := range []string{"", "{", "null", "[]", valid + " {}", strings.TrimSuffix(valid, "}") + `,"extra":1}`, strings.Replace(valid, `"enabled":false`, `"enabled":null`, 1), strings.Replace(valid, `"enabled":false,`, "", 1), strings.Replace(valid, `"expected_revision":0`, `"expected_revision":-1`, 1), strings.Replace(valid, `"expected_revision":0`, `"expected_revision":0.5`, 1), strings.Replace(valid, `"request_id":"`, `"request_id":"bad`, 1)} {
		request(ownerJWT, "PUT", body, 400)
	}
	for _, zone := range []string{"", "CET", "+02:00", "Europe/Nowhere", "posix/Europe/Rome", "right/Europe/Rome"} {
		body := request(ownerJWT, "PUT", payload(uuid.NewString(), 0, false, zone), 400)
		require.Contains(t, string(body), `"field":"timezone"`)
	}
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM workspace_effort_settings WHERE workspace_id=$1`, ws))
	require.Zero(t, count, "invalid and denied writes must not persist")
	first := request(adminJWT, "PUT", valid, 200)
	require.JSONEq(t, string(first), string(request(adminJWT, "PUT", valid, 200)), "replay returns the same receipt")
	request(adminJWT, "PUT", strings.Replace(valid, `"UTC"`, `"Europe/Rome"`, 1), 409)
	require.Contains(t, string(request(ownerJWT, "PUT", payload(uuid.NewString(), 0, true, "Europe/Rome"), 409)), dto.EffortRevisionConflict)
	require.NoError(t, json.Unmarshal(request(session(member), "GET", "", 200), &settings))
	require.False(t, settings.Enabled)
	require.Equal(t, "UTC", settings.Timezone)
	request(ownerJWT, "PUT", payload(uuid.NewString(), 1, false, "Pacific/Kiritimati"), 200)
	exec(`UPDATE workspace_members SET role='member' WHERE workspace_id=$1 AND user_id=$2`, ws, admin)
	request(adminJWT, "PUT", valid, 403)
	request(adminJWT, "PUT", payload(uuid.NewString(), 2, true, "UTC"), 403)
	request(ownerJWT, "PUT", payload(uuid.NewString(), 2, true, "Europe/Rome"), 200)
	exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, ws, admin)
	request(adminJWT, "GET", "", 403)
	request(adminJWT, "PUT", payload(uuid.NewString(), 3, false, "UTC"), 403)
}
