package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestTimesheetOptionsHTTP(t *testing.T) {
	db := newEffortHTTPDatabase(t)
	actor, outsider, a, b, c := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, user := range []uuid.UUID{actor, outsider} {
		_, err := db.Exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Timesheet person','hash')`, user, user.String()+"@example.test")
		require.NoError(t, err)
	}
	for ws, role := range map[uuid.UUID]string{a: "admin", b: "member", c: "guest"} {
		_, err := db.Exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Timesheet workspace',$2,$3)`, ws, ws.String(), outsider)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3)`, ws, actor, role)
		require.NoError(t, err)
	}
	tokens := repository.NewPersonalAccessTokenRepository(db)
	e := echo.New()
	registerRoutes(e, &appHandlers{
		timesheet: handler.NewTimesheetHandler(service.NewTimesheetService(repository.NewTimesheetRepository(db))),
	}, testMiddleware(mw.Auth("timesheet-test-secret", tokens)))
	server := httptest.NewServer(e)
	defer server.Close()
	jwt, err := jwtpkg.GenerateAccessToken(actor, "timesheet-test-secret")
	require.NoError(t, err)
	otherJWT, err := jwtpkg.GenerateAccessToken(outsider, "timesheet-test-secret")
	require.NoError(t, err)
	pat := func(scopes, slugs []string) (string, uuid.UUID) {
		plaintext, hash, prefix, err := domain.GenerateToken()
		require.NoError(t, err)
		id := uuid.New()
		require.NoError(t, tokens.Create(context.Background(), &domain.PersonalAccessToken{
			ID: id, UserID: actor, Name: "Timesheet test", TokenHash: hash, TokenPrefix: prefix, Scopes: scopes, WorkspaceSlugs: slugs,
		}))
		return plaintext, id
	}
	read, readID := pat([]string{domain.PermEffortsRead}, []string{a.String()})
	write, _ := pat([]string{domain.PermEffortsWrite}, nil)
	metadata, _ := pat([]string{domain.PermWorkspacesRead, domain.PermTeamsRead, domain.PermProjectsRead}, nil)
	unrestricted, _ := pat([]string{domain.PermEffortsRead, domain.PermEffortsWrite}, nil)
	empty, _ := pat([]string{domain.PermEffortsRead}, []string{})
	request := func(token, query string, status int) []byte {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, server.URL+"/api/timesheets/options"+query, nil)
		require.NoError(t, err)
		if strings.HasPrefix(token, domain.TokenPrefix) {
			req.Header.Set("Authorization", "Bearer "+token)
			// Explicit PAT credentials must win over the ambient unrestricted session.
			req.AddCookie(&http.Cookie{Name: "access_token", Value: jwt})
		} else if token != "" {
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		}
		res, err := server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, status, res.StatusCode, "%s", body)
		require.Equal(t, "private, no-store", res.Header.Get("Cache-Control"))
		if status != 200 {
			require.NotContains(t, string(body), "workspaces")
			require.NotContains(t, string(body), a.String())
		}
		return body
	}
	decode := func(body []byte) dto.TimesheetOptionsResponse {
		var result dto.TimesheetOptionsResponse
		require.NoError(t, json.Unmarshal(body, &result))
		return result
	}
	request("", "", 401)
	request(write, "", 403)
	request(metadata, "", 403)
	require.Len(t, decode(request(jwt, "", 200)).Workspaces, 3)
	require.Empty(t, decode(request(otherJWT, "", 200)).Workspaces, "workspace owner_id alone is not membership")
	require.Equal(t, []uuid.UUID{a}, decode(request(read, "", 200)).Scope.WorkspaceIDs)
	require.Empty(t, decode(request(empty, "", 200)).Workspaces)
	require.Len(t, decode(request(unrestricted, "", 200)).Workspaces, 3)
	for _, mode := range []string{"person", "project", "workspace", "team"} {
		require.Equal(t, []uuid.UUID{a}, decode(request(jwt, "?mode="+mode, 200)).Scope.WorkspaceIDs)
		request(jwt, "?mode="+mode+"&workspace_ids="+a.String()+","+b.String(), 404)
	}
	for _, ws := range []uuid.UUID{b, uuid.New()} {
		request(read, "?workspace_ids="+ws.String(), 404)
	}
	for _, query := range []string{"?mode=other", "?workspace_ids=none", "?team_ids=bad", "?mode=personal&mode=team", "?project_ids=all,none"} {
		request(jwt, query, 400)
	}
	require.Empty(t, decode(request(jwt, "?workspace_ids=", 200)).Scope.WorkspaceIDs)

	_, err = db.Exec(`UPDATE workspace_members SET role='member' WHERE workspace_id=$1 AND user_id=$2`, a, actor)
	require.NoError(t, err)
	request(read, "?mode=person", 403)
	require.Equal(t, []string{"personal"}, decode(request(jwt, "", 200)).AvailableModes)
	_, err = db.Exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, a, actor)
	require.NoError(t, err)
	require.Empty(t, decode(request(read, "", 200)).Workspaces)
	request(read, "?workspace_ids="+a.String(), 404)
	require.NoError(t, tokens.Revoke(context.Background(), readID, actor))
	request(read, "", 401)
	expired, expiredID := pat([]string{domain.PermEffortsRead}, nil)
	_, err = db.Exec(`UPDATE personal_access_tokens SET expires_at=$2 WHERE id=$1`, expiredID, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	request(expired, "", 401)
	// An operational failure cannot masquerade as an authorized empty result.
	require.NoError(t, db.Close())
	request(jwt, "", 500)
}

func TestTimesheetPersonalHTTP(t *testing.T) {
	db := newEffortHTTPDatabase(t)
	actor, other := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{actor, other} {
		_, err := db.Exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Timesheet user','hash')`, id, id.String()+"@example.test")
		require.NoError(t, err)
	}
	workspaces := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	issues := []uuid.UUID{}
	for n, ws := range workspaces {
		team, status, issue := uuid.New(), uuid.New(), uuid.New()
		issues = append(issues, issue)
		_, err := db.Exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,$2,$3,$4)`, ws, []string{"A", "B", "C"}[n], ws.String(), other)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3)`, ws, actor, []string{"admin", "member", "guest"}[n])
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'Team','T')`, team, ws)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','open','started')`, status, team)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id) VALUES($1,$2,$3,1,'T-1','Issue',$4,$5)`, issue, ws, team, actor, status)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO effort_worklogs(workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes) VALUES($1,$2,$3,$4,'2026-01-05',$5),($1,$2,$4,$3,'2026-01-05',100)`, ws, issue, actor, other, []int{1440, 60, 30}[n])
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO workspace_effort_settings(workspace_id,enabled,timezone) VALUES($1,$2,$3)`, ws, n != 0, []string{"Pacific/Kiritimati", "America/Adak", "Europe/Rome"}[n])
		require.NoError(t, err)
	}
	tokens := repository.NewPersonalAccessTokenRepository(db)
	e := echo.New()
	registerRoutes(e, &appHandlers{timesheet: handler.NewTimesheetHandler(service.NewTimesheetService(repository.NewTimesheetRepository(db)))}, testMiddleware(mw.Auth("timesheet-read-secret", tokens)))
	server := httptest.NewServer(e)
	defer server.Close()
	jwt, err := jwtpkg.GenerateAccessToken(actor, "timesheet-read-secret")
	require.NoError(t, err)
	pat := func(scopes, slugs []string) (string, uuid.UUID) {
		plain, hash, prefix, err := domain.GenerateToken()
		require.NoError(t, err)
		id := uuid.New()
		require.NoError(t, tokens.Create(context.Background(), &domain.PersonalAccessToken{ID: id, UserID: actor, Name: "read", TokenHash: hash, TokenPrefix: prefix, Scopes: scopes, WorkspaceSlugs: slugs}))
		return plain, id
	}
	read, readID := pat([]string{domain.PermEffortsRead}, []string{workspaces[0].String()})
	unrestricted, _ := pat([]string{domain.PermEffortsRead}, nil)
	empty, _ := pat([]string{domain.PermEffortsRead}, []string{})
	write, _ := pat([]string{domain.PermEffortsWrite}, nil)
	query := "?period=week&from=2026-01-05&to=2026-01-11&reference_workspace_id=" + workspaces[0].String()
	request := func(token, path, extra string, status int) []byte {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, server.URL+path+query+extra, nil)
		require.NoError(t, err)
		if strings.HasPrefix(token, domain.TokenPrefix) {
			req.Header.Set("Authorization", "Bearer "+token)
			req.AddCookie(&http.Cookie{Name: "access_token", Value: jwt})
		} else if token != "" {
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		}
		res, err := server.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, status, res.StatusCode, "%s", body)
		require.Equal(t, "private, no-store", res.Header.Get("Cache-Control"))
		if status != 200 {
			require.NotContains(t, string(body), "total_minutes")
			require.NotContains(t, string(body), workspaces[0].String())
		}
		return body
	}
	for _, path := range []string{"/api/timesheets", "/api/timesheets/worklogs"} {
		request("", path, "", 401)
		request(write, path, "", 403)
		request(empty, path, "", 404)
		body := request(unrestricted, path, "", 200)
		var result dto.TimesheetReadResponse
		require.NoError(t, json.Unmarshal(body, &result))
		require.EqualValues(t, 1530, result.TotalMinutes)
		require.Len(t, result.Days, 7)
		require.Equal(t, "2026-01-05", string(result.Days[0]))
		require.False(t, result.Workspaces[0].CanRecordWork)
		require.Equal(t, "Pacific/Kiritimati", result.Reference.Timezone)
		var today string
		require.NoError(t, db.Get(&today, `SELECT (statement_timestamp() AT TIME ZONE 'Pacific/Kiritimati')::date::text`))
		require.Equal(t, today, string(result.Reference.Today))
		require.NoError(t, json.Unmarshal(request(read, path, "", 200), &result))
		require.EqualValues(t, 1440, result.TotalMinutes)
		request(read, path, "&workspace_ids="+workspaces[1].String(), 404)
		request(read, path, "&owner_user_ids="+other.String(), 403)
		request(jwt, path, "&mode=person", 200)
		request(jwt, path, "&page=0", 400)
		request(jwt, path, "&from=2026-01-06", 400)
		request(jwt, path, "&project_ids="+uuid.NewString(), 404)
		require.NoError(t, json.Unmarshal(request(jwt, path, "&workspace_ids=", 200), &result))
		require.Zero(t, result.TotalMinutes)
		require.Empty(t, result.Scope.WorkspaceIDs)
	}
	for _, mode := range []string{"person", "project", "workspace", "team"} {
		for _, path := range []string{"/api/timesheets", "/api/timesheets/worklogs"} {
			var admin dto.TimesheetReadResponse
			require.NoError(t, json.Unmarshal(request(read, path, "&mode="+mode, 200), &admin))
			require.EqualValues(t, 1540, admin.TotalMinutes)
			require.Equal(t, []uuid.UUID{workspaces[0]}, admin.Scope.WorkspaceIDs)
			require.NoError(t, json.Unmarshal(request(jwt, path, "&mode="+mode+"&owner_user_ids="+other.String(), 200), &admin))
			require.EqualValues(t, 100, admin.TotalMinutes)
			require.NoError(t, json.Unmarshal(request(jwt, path, "&mode="+mode+"&owner_user_ids=", 200), &admin))
			require.Zero(t, admin.TotalMinutes)
			request(jwt, path, "&mode="+mode+"&workspace_ids="+workspaces[0].String()+","+workspaces[1].String(), 404)
			request(jwt, path, "&mode="+mode+"&workspace_ids="+workspaces[2].String(), 404)
			request(empty, path, "&mode="+mode, 403)
			request(write, path, "&mode="+mode, 403)
		}
	}
	var detail dto.TimesheetWorklogsResponse
	var selected dto.TimesheetGridResponse
	require.NoError(t, json.Unmarshal(request(jwt, "/api/timesheets", "&workspace_ids="+workspaces[1].String(), 200), &selected))
	require.EqualValues(t, 60, selected.TotalMinutes)
	require.Equal(t, workspaces[0], selected.Reference.ID)
	query = strings.Replace(query, workspaces[0].String(), workspaces[1].String(), 1)
	request(read, "/api/timesheets", "&workspace_ids="+workspaces[0].String(), 404)
	request(read, "/api/timesheets/worklogs", "&workspace_ids="+workspaces[0].String(), 404)
	query = strings.Replace(query, workspaces[1].String(), workspaces[0].String(), 1)
	require.NoError(t, json.Unmarshal(request(jwt, "/api/timesheets/worklogs", "&issue_id="+issues[1].String()+"&work_date=2026-01-05", 200), &detail))
	require.Len(t, detail.Data, 1)
	require.Equal(t, actor, detail.Data[0].Owner.ID)
	require.Equal(t, other, detail.Data[0].Operator.ID)
	require.Equal(t, workspaces[1], detail.Data[0].Workspace.ID)
	request(read, "/api/timesheets/worklogs", "&issue_id="+issues[1].String(), 404)
	request(jwt, "/api/timesheets/worklogs", "&work_date=2026-01-12", 400)
	_, err = db.Exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, workspaces[1], actor)
	require.NoError(t, err)
	var grid dto.TimesheetGridResponse
	require.NoError(t, json.Unmarshal(request(jwt, "/api/timesheets", "", 200), &grid))
	require.EqualValues(t, 1470, grid.TotalMinutes)
	require.NoError(t, tokens.Revoke(context.Background(), readID, actor))
	request(read, "/api/timesheets", "", 401)
	request(read, "/api/timesheets/worklogs", "", 401)
	_, err = db.Exec(`UPDATE workspace_members SET role='member' WHERE workspace_id=$1 AND user_id=$2`, workspaces[0], actor)
	require.NoError(t, err)
	for _, mode := range []string{"person", "project", "workspace", "team"} {
		request(jwt, "/api/timesheets", "&mode="+mode, 403)
		request(jwt, "/api/timesheets/worklogs", "&mode="+mode, 403)
	}
	_, err = db.Exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, workspaces[0], actor)
	require.NoError(t, err)
	request(jwt, "/api/timesheets", "", 404)
	require.NoError(t, db.Close())
	request(jwt, "/api/timesheets", "", 500)
}
