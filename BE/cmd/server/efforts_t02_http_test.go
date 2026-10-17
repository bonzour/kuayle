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
	"time"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/handler"
	mw "github.com/kuayle/kuayle-backend/internal/middleware"
	"github.com/kuayle/kuayle-backend/internal/realtime"
	"github.com/kuayle/kuayle-backend/internal/repository"
	"github.com/kuayle/kuayle-backend/internal/service"
	jwtpkg "github.com/kuayle/kuayle-backend/pkg/jwt"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

func TestEffortsT02HTTPAndRealtime(t *testing.T) {
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
	h := &appHandlers{effort: handler.NewEffortHandler(efforts), issue: handler.NewIssueHandler(issueService, service.NewCommentService(repository.NewCommentRepository(db), issues, hub, notifications), users, statuses, projects, cycles, service.NewIssueRelationService(repository.NewIssueRelationRepository(db), issues))}
	secret := "efforts-t02-http-secret"
	middleware := testMiddleware(mw.Auth(secret, tokens))
	middleware.workspaceMembership = mw.WorkspaceMembership(workspaces)
	e := echo.New()
	registerRoutes(e, h, middleware)
	server := httptest.NewServer(e)
	defer server.Close()
	jwt, err := jwtpkg.GenerateAccessToken(owner, secret)
	require.NoError(t, err)
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
	rev := int64(0)
	remaining := int64(300)
	for _, id := range ids[:3] {
		_, err = repo.SetRemaining(ctx, ws, id, dto.SetEffortRemainingRequest{EffortMutationRequest: dto.EffortMutationRequest{RequestID: uuid.NewString(), ExpectedRevision: &rev}, RemainingMinutes: &remaining})
		require.NoError(t, err)
	}
	// Denial occurs before history, issue fields, lifecycle or effort data change.
	request(legacy, "PATCH", "/issues/T02-1", `{"title":"must not save","status":"done"}`, 403)
	var history int
	require.NoError(t, db.Get(&history, `SELECT COUNT(*) FROM issue_history WHERE issue_id=$1`, ids[0]))
	require.Zero(t, history)
	saved, err := issues.GetByID(ctx, ids[0])
	require.NoError(t, err)
	require.Equal(t, "T02 HTTP", saved.Title)
	require.Equal(t, open, *saved.StatusID)
	request(legacy, "PATCH", "/issues/T02-1", `{"title":"ordinary title"}`, 200)
	request(full, "PATCH", "/issues/T02-1", `{"status_id":"`+closed.String()+`"}`, 200)
	summary, err := repo.Summary(ctx, ws, "T02-1")
	require.NoError(t, err)
	require.EqualValues(t, 0, *summary.Own.RemainingMinutes)
	request(legacy, "PATCH", "/issues/bulk", `{"issue_ids":["`+ids[1].String()+`"],"status":"done"}`, 403)
	request(full, "PATCH", "/issues/bulk", `{"issue_ids":["`+ids[1].String()+`"],"status":"done"}`, 200)
	request(legacy, "POST", "/issues/T02-3/triage/decline", `{}`, 403)
	request(jwt, "POST", "/issues/T02-3/triage/decline", `{}`, 200)
	for _, id := range ids[1:3] {
		var value int64
		require.NoError(t, db.Get(&value, `SELECT remaining_minutes FROM issue_efforts WHERE issue_id=$1`, id))
		require.Zero(t, value)
	}

	// Both parent and child automation use the same transactional closure hook.
	exec(`UPDATE teams SET parent_auto_close_enabled=TRUE,sub_issue_auto_close_enabled=TRUE WHERE id=$1`, team)
	exec(`UPDATE issues SET parent_id=$1 WHERE id=$2`, ids[4], ids[5])
	exec(`UPDATE issues SET parent_id=$1 WHERE id=$2`, ids[6], ids[7])
	for _, id := range ids[4:] {
		_, err = repo.SetRemaining(ctx, ws, id, dto.SetEffortRemainingRequest{EffortMutationRequest: dto.EffortMutationRequest{RequestID: uuid.NewString(), ExpectedRevision: &rev}, RemainingMinutes: &remaining})
		require.NoError(t, err)
	}
	request(jwt, "PATCH", "/issues/T02-6", `{"status":"done"}`, 200)
	request(jwt, "PATCH", "/issues/T02-7", `{"status":"done"}`, 200)
	for _, id := range ids[4:] {
		var value int64
		require.NoError(t, db.Get(&value, `SELECT remaining_minutes FROM issue_efforts WHERE issue_id=$1`, id))
		require.Zero(t, value)
	}
	for _, route := range e.Routes() {
		if strings.Contains(route.Path, "/efforts") {
			require.Equal(t, "GET", route.Method)
		}
	}

	// A failure in Efforts history/revision aborts closure, including lifecycle
	// events; the HTTP response must not disclose database details.
	request(jwt, "PATCH", "/issues/T02-1", `{"status":"started"}`, 200)
	exec(`UPDATE issue_efforts SET revision=$1,remaining_minutes=300 WHERE issue_id=$2`, domain.EffortMaxSafeInteger, ids[0])
	require.NoError(t, db.Get(&history, `SELECT COUNT(*) FROM issue_history WHERE issue_id=$1`, ids[0]))
	body := request(jwt, "PATCH", "/issues/T02-1", `{"status":"done"}`, 500)
	require.Contains(t, string(body), "INTERNAL_ERROR")
	require.NotContains(t, string(body), "SQLSTATE")
	saved, err = issues.GetByID(ctx, ids[0])
	require.NoError(t, err)
	require.Equal(t, open, *saved.StatusID)
	var afterHistory int
	require.NoError(t, db.Get(&afterHistory, `SELECT COUNT(*) FROM issue_history WHERE issue_id=$1`, ids[0]))
	require.Equal(t, history, afterHistory)

	// A live socket receives only a generic post-commit refresh. A marker after
	// rollback/replay establishes queue ordering without timing-based no-message tests.
	ready := make(chan struct{}, 1)
	socketServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		client := hub.Register(conn, ws, owner)
		defer hub.Unregister(client)
		ready <- struct{}{}
		go hub.WritePump(r.Context(), client)
		hub.ReadPump(r.Context(), client)
	}))
	defer socketServer.Close()
	socketCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	socket, _, err := websocket.Dial(socketCtx, "ws"+strings.TrimPrefix(socketServer.URL, "http"), nil)
	require.NoError(t, err)
	defer socket.CloseNow()
	<-ready
	read := func() realtime.Event {
		t.Helper()
		_, data, err := socket.Read(socketCtx)
		require.NoError(t, err)
		require.NotContains(t, string(data), ids[3].String())
		require.NotContains(t, string(data), "work_date")
		var event realtime.Event
		require.NoError(t, json.Unmarshal(data, &event))
		return event
	}
	log := dto.CreateEffortWorklogRequest{EffortMutationRequest: dto.EffortMutationRequest{RequestID: uuid.NewString(), ExpectedRevision: &rev}, OwnerUserID: owner.String(), WorkDate: "2026-01-01", DurationMinutes: 60}
	receipt, err := efforts.CreateWorklog(ctx, ws, ids[3], log)
	require.NoError(t, err)
	require.Equal(t, "app.refresh", read().Type)
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM effort_worklogs WHERE id=$1`, receipt.ResourceID))
	require.Equal(t, 1, count, "refresh cannot precede commit")
	_, err = efforts.CreateWorklog(ctx, ws, ids[3], log)
	require.NoError(t, err)
	hub.Broadcast(ws, realtime.Event{Type: "test.barrier"})
	require.Equal(t, "test.barrier", read().Type, "replay must not publish")
	log.RequestID = uuid.NewString()
	_, err = efforts.CreateWorklog(ctx, ws, ids[3], log)
	require.Error(t, err)
	hub.Broadcast(ws, realtime.Event{Type: "test.barrier"})
	require.Equal(t, "test.barrier", read().Type, "rollback must not publish")
}
