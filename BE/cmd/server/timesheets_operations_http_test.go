package main

import (
	"context"
	"encoding/json"
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

func TestTimesheetOperationsHTTP(t *testing.T) {
	db := newEffortHTTPDatabase(t)
	actor, recipient := uuid.New(), uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	for _, user := range []uuid.UUID{actor, recipient} {
		exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'EP05','hash')`, user, user.String()+"@example.test")
	}
	workspaces := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	issues := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	teams := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for n, ws := range workspaces {
		exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'EP05',$2,$3)`, ws, ws.String(), recipient)
		exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3),($1,$4,'owner')`, ws, actor, []string{"admin", "member", "guest"}[n], recipient)
		exec(`INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'EP05 Team','T')`, teams[n], ws)
		exec(`INSERT INTO team_statuses(team_id,name,slug,category) VALUES($1,'Open','open','started')`, teams[n])
		exec(`INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id)
 VALUES($1,$2,$3,1,'T-1','Unlogged EP05',$4,(SELECT id FROM team_statuses WHERE team_id=$3))`, issues[n], ws, teams[n], recipient)
		exec(`INSERT INTO issue_efforts(workspace_id,issue_id,estimate_minutes,remaining_minutes,revision) VALUES($1,$2,480,480,0)`, ws, issues[n])
		exec(`INSERT INTO workspace_effort_settings(workspace_id,enabled,timezone) VALUES($1,TRUE,$2)`, ws, []string{"Pacific/Kiritimati", "America/Adak", "Europe/Rome"}[n])
	}
	tokens := repository.NewPersonalAccessTokenRepository(db)
	middleware := testMiddleware(mw.Auth("timesheet-ep05", tokens))
	middleware.workspaceMembership = mw.WorkspaceMembership(repository.NewWorkspaceRepository(db))
	e := echo.New()
	registerRoutes(e, &appHandlers{
		timesheet: handler.NewTimesheetHandler(service.NewTimesheetService(repository.NewTimesheetRepository(db))),
		effort:    handler.NewEffortHandler(service.NewEffortService(repository.NewEffortRepository(db)), repository.NewIssueRepository(db)),
	}, middleware)
	jwt, err := jwtpkg.GenerateAccessToken(actor, "timesheet-ep05")
	require.NoError(t, err)
	pat := func(scopes, slugs []string) string {
		plain, hash, prefix, err := domain.GenerateToken()
		require.NoError(t, err)
		require.NoError(t, tokens.Create(context.Background(), &domain.PersonalAccessToken{ID: uuid.New(), UserID: actor, Name: "EP05", TokenHash: hash, TokenPrefix: prefix, Scopes: scopes, WorkspaceSlugs: slugs}))
		return plain
	}
	read := pat([]string{domain.PermEffortsRead}, []string{workspaces[0].String()})
	full := pat([]string{domain.PermEffortsRead, domain.PermEffortsWrite}, []string{workspaces[1].String()})
	writeOnly := pat([]string{domain.PermEffortsWrite}, nil)
	request := func(token, method, path string, body any, status int) []byte {
		t.Helper()
		data, err := json.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(method, path, strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")
		if strings.HasPrefix(token, domain.TokenPrefix) {
			req.Header.Set("Authorization", "Bearer "+token)
		} else if token != "" {
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		require.Equal(t, status, rec.Code, "%s %s: %s", method, path, rec.Body.String())
		require.Equal(t, "private, no-store", rec.Header().Get("Cache-Control"))
		return rec.Body.Bytes()
	}
	suggestions := "/api/timesheets/issues?workspace_id=" + workspaces[0].String()
	request("", "GET", suggestions, nil, 401)
	request(writeOnly, "GET", suggestions, nil, 403)
	var options dto.TimesheetIssuesResponse
	require.NoError(t, json.Unmarshal(request(read, "GET", suggestions+"&search=unlogged", nil, 200), &options))
	require.Len(t, options.Data, 1)
	require.False(t, options.Workspace.CanRecordWork)
	request(read, "GET", "/api/timesheets/issues?workspace_id="+workspaces[1].String(), nil, 404)
	request(jwt, "GET", suggestions+"&page=0", nil, 400)
	request(jwt, "GET", suggestions+"&search=a&search=b", nil, 400)
	request(jwt, "GET", "/api/timesheets/issues?workspace_id="+uuid.NewString(), nil, 404)
	base := func(n int) string { return "/api/workspaces/" + workspaces[n].String() + "/issues/T-1/efforts" }
	create := func(revision int, owner uuid.UUID, duration int, date string) map[string]any {
		return map[string]any{"request_id": uuid.NewString(), "expected_revision": revision, "owner_user_id": owner.String(), "work_date": date, "duration_minutes": duration}
	}
	a := create(0, actor, 1440, "2026-01-05")
	request(jwt, "POST", base(0)+"/worklogs", a, 200)
	b := create(0, actor, 60, "2026-01-05")
	first := request(full, "POST", base(1)+"/worklogs", b, 200)
	query := "/api/timesheets?period=week&from=2026-01-05&to=2026-01-11&reference_workspace_id=" + workspaces[0].String()
	var grid dto.TimesheetGridResponse
	require.NoError(t, json.Unmarshal(request(jwt, "GET", query, nil, 200), &grid))
	require.EqualValues(t, 1500, grid.TotalMinutes, "the daily limit is per workspace")
	var logs dto.EffortWorklogListResponse
	require.NoError(t, json.Unmarshal(request(full, "GET", base(1)+"/worklogs", nil, 200), &logs))
	require.Len(t, logs.Data, 1)
	id := logs.Data[0].ID.String()
	update := map[string]any{"request_id": uuid.NewString(), "expected_revision": 1, "work_date": "2026-01-06", "duration_minutes": 30}
	request(full, "PUT", base(1)+"/worklogs/"+id, update, 200)
	require.NoError(t, json.Unmarshal(request(jwt, "GET", query, nil, 200), &grid))
	require.EqualValues(t, 1440, grid.DailyTotals[0].Minutes)
	require.EqualValues(t, 30, grid.DailyTotals[1].Minutes)
	request(full, "PUT", base(1)+"/worklogs/"+id, map[string]any{"request_id": uuid.NewString(), "expected_revision": 1, "work_date": "2026-01-05", "duration_minutes": 10}, 409)
	request(full, "DELETE", base(1)+"/worklogs/"+id, map[string]any{"request_id": uuid.NewString(), "expected_revision": 2}, 200)
	require.JSONEq(t, string(first), string(request(full, "POST", base(1)+"/worklogs", b, 200)), "replay cannot resurrect deleted work")
	var remaining, count int
	require.NoError(t, db.Get(&remaining, `SELECT remaining_minutes FROM issue_efforts WHERE issue_id=$1`, issues[1]))
	require.Equal(t, 420, remaining)
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM effort_events WHERE issue_id=$1 AND event_type LIKE 'worklog_%'`, issues[1]))
	require.Equal(t, 3, count)
	request(jwt, "POST", base(2)+"/worklogs", create(0, actor, 1, "2026-01-05"), 403)
	request(jwt, "POST", base(1)+"/worklogs", create(3, recipient, 1, "2026-01-05"), 403)
	request(full, "GET", base(1)+"/worklog-owners", nil, 403)
	request(jwt, "POST", base(0)+"/worklogs", create(1, recipient, 60, "2026-01-05"), 200)
	exec(`UPDATE workspace_effort_settings SET enabled=FALSE WHERE workspace_id=$1`, workspaces[0])
	request(jwt, "POST", base(0)+"/worklogs", create(2, actor, 1, "2026-01-06"), 409)
	request(jwt, "GET", suggestions, nil, 200)
	request(full, "POST", base(1)+"/worklogs", create(3, actor, 1, "2026-01-05"), 200)
	exec(`UPDATE workspace_members SET role='guest' WHERE workspace_id=$1 AND user_id=$2`, workspaces[1], actor)
	request(full, "POST", base(1)+"/worklogs", create(4, actor, 1, "2026-01-05"), 403)
	exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, workspaces[1], actor)
	request(full, "GET", "/api/timesheets/issues?workspace_id="+workspaces[1].String(), nil, 404)
}
