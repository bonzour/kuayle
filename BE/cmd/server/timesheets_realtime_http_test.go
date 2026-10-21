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

func TestTimesheetRealtimeHTTP(t *testing.T) {
	db := newEffortHTTPDatabase(t)
	actor, owner, outsider := uuid.New(), uuid.New(), uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	for _, u := range []uuid.UUID{actor, owner, outsider} {
		exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Realtime person','hash')`, u, u.String()+"@example.test")
	}
	workspaces := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for i, w := range workspaces {
		team, status, issue := uuid.New(), uuid.New(), uuid.New()
		exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Realtime workspace',$2,$3)`, w, w.String(), owner)
		exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner'),($1,$3,$4)`, w, owner, actor, []string{"admin", "member", "guest"}[i])
		exec(`INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'Realtime team','RT')`, team, w)
		exec(`INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','open','started')`, status, team)
		exec(`INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id) VALUES($1,$2,$3,1,'RT-1','PRIVATE TITLE',$4,$5)`, issue, w, team, owner, status)
	}
	tokens := repository.NewPersonalAccessTokenRepository(db)
	workspacesRepo := repository.NewWorkspaceRepository(db)
	issuesRepo := repository.NewIssueRepository(db)
	hub := realtime.NewHub(workspacesRepo.RealtimeAccess)
	middleware := testMiddleware(mw.Auth("timesheet-realtime-secret", tokens))
	middleware.workspaceMembership = mw.WorkspaceMembership(workspacesRepo)
	e := echo.New()
	registerRoutes(e, &appHandlers{
		ws:        handler.NewWSHandler(hub),
		timesheet: handler.NewTimesheetHandler(service.NewTimesheetService(repository.NewTimesheetRepository(db))),
		effort:    handler.NewEffortHandler(service.NewEffortService(repository.NewEffortRepository(db), hub), issuesRepo),
		workspace: handler.NewWorkspaceHandler(service.NewWorkspaceService(workspacesRepo, repository.NewUserRepository(db), hub)),
		project:   handler.NewProjectHandler(service.NewProjectService(repository.NewProjectRepository(db), hub)),
	}, middleware)
	server := httptest.NewServer(e)
	defer server.Close()
	jwt := func(u uuid.UUID) string {
		token, err := jwtpkg.GenerateAccessToken(u, "timesheet-realtime-secret")
		require.NoError(t, err)
		return token
	}
	actorJWT, ownerJWT := jwt(actor), jwt(owner)
	request := func(token, method, path, body string, want int) []byte {
		t.Helper()
		r, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		require.NoError(t, err)
		r.Header.Set("Content-Type", "application/json")
		r.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		res, err := server.Client().Do(r)
		require.NoError(t, err)
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, want, res.StatusCode, "%s %s %s", method, path, data)
		return data
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	type connection struct {
		frames chan []byte
		errors chan error
	}
	dial := func(token, slug, query string, want int) *connection {
		t.Helper()
		headers := http.Header{}
		headers.Set("Cookie", "access_token="+token)
		socket, res, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/workspaces/"+slug+"/ws"+query, &websocket.DialOptions{HTTPHeader: headers})
		require.Equal(t, want, res.StatusCode)
		if want != 101 {
			require.Error(t, err)
			return nil
		}
		require.NoError(t, err)
		t.Cleanup(func() { _ = socket.CloseNow() })
		c := &connection{frames: make(chan []byte, 20), errors: make(chan error, 1)}
		go func() {
			for {
				_, data, err := socket.Read(ctx)
				if err != nil {
					c.errors <- err
					return
				}
				c.frames <- data
			}
		}()
		require.NoError(t, socket.Ping(ctx))
		return c
	}
	frame := func(c *connection) {
		t.Helper()
		select {
		case raw := <-c.frames:
			require.JSONEq(t, `{"type":"app.refresh","payload":{"resources":["timesheets"]}}`, string(raw))
		case err := <-c.errors:
			t.Fatalf("socket closed: %v", err)
		case <-ctx.Done():
			t.Fatal("notification missing")
		}
	}
	a := dial(actorJWT, workspaces[0].String(), "?channel=timesheets", 101)
	b := dial(actorJWT, workspaces[1].String(), "?channel=timesheets", 101)
	dial("", workspaces[0].String(), "?channel=timesheets", 401)
	dial(jwt(outsider), workspaces[0].String(), "?channel=timesheets", 403)
	dial(actorJWT, uuid.NewString(), "?channel=timesheets", 404)
	dial(actorJWT, workspaces[0].String(), "?channel=unknown", 400)
	raw, hash, prefix, err := domain.GenerateToken()
	require.NoError(t, err)
	require.NoError(t, tokens.Create(ctx, &domain.PersonalAccessToken{ID: uuid.New(), UserID: actor, Name: "no socket", TokenHash: hash, TokenPrefix: prefix, Scopes: []string{domain.PermEffortsRead}}))
	_, res, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/workspaces/"+workspaces[0].String()+"/ws?channel=timesheets", &websocket.DialOptions{HTTPHeader: http.Header{"Authorization": []string{"Bearer " + raw}}})
	require.Error(t, err)
	require.Equal(t, 403, res.StatusCode)
	base := "/api/workspaces/" + workspaces[1].String()
	request(ownerJWT, "POST", base+"/issues/RT-1/efforts/worklogs", `{"request_id":"`+uuid.NewString()+`","expected_revision":0,"owner_user_id":"`+actor.String()+`","work_date":"2025-10-06","end_date":"2025-10-06","duration_minutes":60,"description":"PRIVATE DESCRIPTION"}`, 200)
	frame(b)
	select {
	case message := <-a.frames:
		t.Fatalf("B event reached A: %s", message)
	default:
	}
	query := "/api/timesheets?period=week&from=2025-10-06&to=2025-10-12&reference_workspace_id=" + workspaces[0].String()
	require.Contains(t, string(request(actorJWT, "GET", query, "", 200)), `"total_minutes":60`)
	projectBody := request(ownerJWT, "POST", base+"/projects", `{"name":"PRIVATE PROJECT"}`, 201)
	frame(b)
	var project struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(projectBody, &project))
	request(ownerJWT, "PATCH", base+"/projects/"+project.ID, `{"name":"Changed"}`, 200)
	frame(b)
	request(ownerJWT, "DELETE", base+"/projects/"+project.ID, "", 200)
	frame(b)
	request(ownerJWT, "PATCH", "/api/workspaces/"+workspaces[0].String()+"/members/"+actor.String(), `{"role":"member"}`, 200)
	frame(a)
	request(actorJWT, "GET", query+"&mode=person", "", 403)
	// Idle removal is detected by the periodic policy check without any broadcast.
	exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, workspaces[1], actor)
	select {
	case err := <-b.errors:
		require.Equal(t, websocket.StatusPolicyViolation, websocket.CloseStatus(err))
	case <-ctx.Done():
		t.Fatal("idle socket did not close")
	}
	request(actorJWT, "GET", query+"&workspace_ids="+workspaces[1].String(), "", 404)
	dial(actorJWT, workspaces[1].String(), "?channel=timesheets", 403)
}
