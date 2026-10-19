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
	"github.com/kuayle/kuayle-backend/internal/handler"
	mw "github.com/kuayle/kuayle-backend/internal/middleware"
	"github.com/kuayle/kuayle-backend/internal/realtime"
	"github.com/kuayle/kuayle-backend/internal/repository"
	"github.com/kuayle/kuayle-backend/internal/service"
	jwtpkg "github.com/kuayle/kuayle-backend/pkg/jwt"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestEffortsV1HTTP(t *testing.T) {
	db := newEffortHTTPDatabase(t)
	owner, ws, team, open, closed, cancelled := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	exec := func(q string, args ...any) { t.Helper(); _, err := db.Exec(q, args...); require.NoError(t, err) }
	exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'T02 owner','hash')`, owner, owner.String()+"@example.test")
	exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'T02 HTTP',$2,$3)`, ws, ws.String(), owner)
	exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, ws, owner)
	exec(`INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'T02 HTTP','T02')`, team, ws)
	exec(`INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','started','started'),($3,$2,'Done','done','completed'),($4,$2,'Cancelled','cancelled','cancelled')`, open, team, closed, cancelled)
	for n, id := range ids {
		exec(`INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id,status,triaged) VALUES($1,$2,$3,$4,$5,'T02 HTTP',$6,$7,'started',FALSE)`, id, ws, team, n+1, fmt.Sprintf("T02-%d", n+1), owner, open)
	}
	users := repository.NewUserRepository(db)
	workspaces := repository.NewWorkspaceRepository(db)
	teams := repository.NewTeamRepository(db)
	statuses := repository.NewTeamStatusRepository(db)
	issues := repository.NewIssueRepository(db)
	projects := repository.NewProjectRepository(db)
	cycles := repository.NewCycleRepository(db)
	tokens := repository.NewPersonalAccessTokenRepository(db)
	hub := realtime.NewHub(workspaces.RealtimeAccess)
	repo := repository.NewEffortRepository(db)
	efforts := service.NewEffortService(repo, hub)
	notifications := service.NewNotificationService(repository.NewNotificationRepository(db))
	issueService := service.NewIssueService(issues, teams, statuses, repository.NewIssueHistoryRepository(db), hub, notifications, projects)
	h := &appHandlers{effort: handler.NewEffortHandler(efforts, issues), issue: handler.NewIssueHandler(issueService, service.NewCommentService(repository.NewCommentRepository(db), issues, hub, notifications), users, statuses, projects, cycles, service.NewIssueRelationService(repository.NewIssueRelationRepository(db), issues))}
	secret := "efforts-t02-http-secret"
	middleware := testMiddleware(mw.Auth(secret, tokens))
	middleware.workspaceMembership = mw.WorkspaceMembership(workspaces)
	e := echo.New()
	registerRoutes(e, h, middleware)
	server := httptest.NewServer(e)
	defer server.Close()
	pat := func(scopes []string) string {
		raw, hash, prefix, err := domain.GenerateToken()
		require.NoError(t, err)
		require.NoError(t, tokens.Create(context.Background(), &domain.PersonalAccessToken{ID: uuid.New(), UserID: owner, Name: "T02 token", TokenHash: hash, TokenPrefix: prefix, Scopes: scopes}))
		return raw
	}
	legacy := pat([]string{domain.PermIssueUpdate, domain.PermIssuesRead})
	full := pat([]string{domain.PermIssueUpdate, domain.PermIssuesRead, domain.PermEffortsRead, domain.PermEffortsWrite})
	base := "/api/workspaces/" + ws.String()
	request := func(token, method, path, body string, status int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+base+path, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		if strings.HasPrefix(token, domain.TokenPrefix) {
			req.Header.Set("Authorization", "Bearer "+token)
		} else {
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		}
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, status, resp.StatusCode, "%s %s: %s", method, path, data)
		return data
	}
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: owner, WorkspaceID: ws})

	envelope := func(rev int) string {
		return fmt.Sprintf(`"request_id":%q,"expected_revision":%d`, uuid.NewString(), rev)
	}
	writeOnly := pat([]string{domain.PermEffortsWrite})
	readOnly := pat([]string{domain.PermEffortsRead})
	for _, token := range []string{legacy, writeOnly, readOnly} {
		request(token, "PUT", "/issues/T02-8/efforts/estimate", `{`+envelope(0)+`,"estimate_minutes":480}`, 403)
	}
	request(full, "PUT", "/issues/T02-8/efforts/estimate", `{`+envelope(0)+`,"estimate_minutes":480}`, 200)
	request(full, "PUT", "/issues/T02-8/efforts/remaining", `{`+envelope(1)+`,"remaining_minutes":300}`, 200)
	body := request(full, "POST", "/issues/T02-8/efforts/worklogs", `{`+envelope(2)+`,"owner_user_id":"`+owner.String()+`","duration_minutes":60,"work_date":"2020-01-01","description":"secret work"}`, 200)
	var receipt struct {
		ResourceID string `json:"resource_id"`
	}
	require.NoError(t, json.Unmarshal(body, &receipt))
	request(full, "PUT", "/issues/T02-8/efforts/worklogs/"+receipt.ResourceID, `{`+envelope(3)+`,"duration_minutes":90,"work_date":"2020-01-02","description":"corrected work"}`, 200)
	request(full, "DELETE", "/issues/T02-8/efforts/worklogs/"+receipt.ResourceID, `{`+envelope(4)+`}`, 200)
	history := request(legacy, "GET", "/issues/T02-8/history", "", 200)
	require.NotContains(t, string(history), "secret work")
	require.NotContains(t, string(history), "effort")
	history = request(full, "GET", "/issues/T02-8/history", "", 200)
	require.Contains(t, string(history), "secret work")
	require.Contains(t, string(history), "worklog_deleted")
	require.Contains(t, string(history), "T02 owner")
	request(full, "PUT", "/efforts/settings", `{`+envelope(0)+`,"enabled":false,"timezone":"UTC"}`, 403)
	limited := pat([]string{domain.PermIssueCreate, domain.PermIssueDeleteOwn, "workspace:transfer"})
	request(limited, "POST", "/issues/T02-8/duplicate", `{}`, 403)
	request(limited, "DELETE", "/issues/T02-8", "", 403)
	request(limited, "DELETE", "/issues/bulk", `{"issue_ids":["`+ids[7].String()+`"]}`, 403)
	request(limited, "GET", "/export", "", 403)
	complete := pat(domain.RolePermissions[domain.RoleOwner])
	body = request(complete, "POST", "/issues/T02-8/duplicate", `{}`, 201)
	var copied struct {
		Identifier string `json:"identifier"`
	}
	require.NoError(t, json.Unmarshal(body, &copied))
	summary, err := repo.Summary(ctx, ws, copied.Identifier)
	require.NoError(t, err)
	require.EqualValues(t, 480, *summary.Own.EstimateMinutes)
	require.EqualValues(t, 480, *summary.Own.RemainingMinutes)
	require.Zero(t, summary.Own.SpentMinutes)

	exec(`UPDATE issues SET parent_id=$1 WHERE id=$2`, ids[7], ids[6])
	exec(`INSERT INTO issue_efforts(workspace_id,issue_id,estimate_minutes,remaining_minutes) VALUES($1,$2,60,30)`, ws, ids[6])
	request(full, "PATCH", "/issues/T02-8", `{"status_id":"`+closed.String()+`"}`, 200)
	body = request(complete, "POST", "/issues/T02-8/duplicate", `{"include_sub_issues":true}`, 201)
	require.NoError(t, json.Unmarshal(body, &copied))
	summary, err = repo.Summary(ctx, ws, copied.Identifier)
	require.NoError(t, err)
	require.Zero(t, *summary.Own.RemainingMinutes)
	require.True(t, summary.HasVisibleDescendants)
	require.EqualValues(t, 540, *summary.Total.EstimateMinutes)
	// Conversion retains original IDs, planning and history.
	request(complete, "POST", "/issues/T02-8/convert-to-project", `{}`, 201)
	summary, err = repo.Summary(ctx, ws, "T02-8")
	require.NoError(t, err)
	require.EqualValues(t, 480, *summary.Own.EstimateMinutes)
	history = request(full, "GET", "/issues/T02-8/history", "", 200)
	require.Contains(t, string(history), "secret work")
	jwt, err := jwtpkg.GenerateAccessToken(owner, secret)
	require.NoError(t, err)
	request(jwt, "PUT", "/efforts/settings", `{`+envelope(0)+`,"enabled":false,"timezone":"UTC"}`, 200)
	body = request(complete, "POST", "/issues/T02-8/duplicate", `{}`, 201)
	require.NoError(t, json.Unmarshal(body, &copied))
	summary, err = repo.Summary(ctx, ws, copied.Identifier)
	require.NoError(t, err)
	require.False(t, summary.HasData)
	// Scopes cannot outlive the owner's current role.
	exec(`UPDATE workspace_members SET role='guest' WHERE workspace_id=$1 AND user_id=$2`, ws, owner)
	request(complete, "POST", "/issues/T02-8/efforts/worklogs", `{`+envelope(5)+`,"owner_user_id":"`+owner.String()+`","duration_minutes":1,"work_date":"2020-01-01"}`, 403)
}
