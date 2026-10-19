package main

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestEffortPlanningHTTP(t *testing.T) {
	db := newEffortHTTPDatabase(t)
	ws, otherWS, owner, admin, member, guest := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	team, hiddenTeam, otherTeam, issue, hidden, foreign, open, closed := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	for _, user := range []uuid.UUID{owner, admin, member, guest} {
		exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'EP02','hash')`, user, user.String()+"@example.test")
	}
	for _, workspace := range []uuid.UUID{ws, otherWS} {
		exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'EP02',$2,$3)`, workspace, workspace.String(), owner)
	}
	for user, role := range map[uuid.UUID]string{owner: "owner", admin: "admin", member: "member", guest: "guest"} {
		exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3)`, ws, user, role)
	}
	exec(`INSERT INTO teams(id,workspace_id,name,key,is_private) VALUES($1,$2,'EP02','EP',FALSE),($3,$2,'Hidden','HID',TRUE),($4,$5,'Other','OTHER',FALSE)`, team, ws, hiddenTeam, otherTeam, otherWS)
	exec(`INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','open','started'),($3,$2,'Closed','closed','completed')`, open, team, closed)
	for _, teamID := range []uuid.UUID{hiddenTeam, otherTeam} {
		exec(`INSERT INTO team_statuses(team_id,name,slug,category) VALUES($1,'Open','open','started')`, teamID)
	}
	for _, row := range []struct {
		id, ws, team uuid.UUID
		identifier   string
	}{{issue, ws, team, "EP-1"}, {hidden, ws, hiddenTeam, "HID-1"}, {foreign, otherWS, otherTeam, "OTHER-1"}} {
		exec(`INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status,status_id) VALUES($1,$2,$3,1,$4,'Planning',$5,'todo',(SELECT id FROM team_statuses WHERE team_id=$3 AND category='started' LIMIT 1))`, row.id, row.ws, row.team, row.identifier, owner)
	}
	exec(`UPDATE issues SET status_id=$1 WHERE id=$2`, open, issue)
	exec(`INSERT INTO issue_assignees(issue_id,user_id) VALUES($1,$2)`, issue, member)
	exec(`INSERT INTO effort_worklogs(workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes) VALUES($1,$2,$3,$3,'2026-01-01',120)`, ws, issue, owner)
	workspaces := repository.NewWorkspaceRepository(db)
	issues := repository.NewIssueRepository(db)
	efforts := repository.NewEffortRepository(db)
	tokens := repository.NewPersonalAccessTokenRepository(db)
	e := echo.New()
	middleware := testMiddleware(mw.Auth("ep02-http-secret", tokens))
	middleware.workspaceMembership = mw.WorkspaceMembership(workspaces)
	registerRoutes(e, &appHandlers{effort: handler.NewEffortHandler(service.NewEffortService(efforts), issues)}, middleware)
	session := func(user uuid.UUID) string {
		token, err := jwtpkg.GenerateAccessToken(user, "ep02-http-secret")
		require.NoError(t, err)
		return token
	}
	request := func(token, identifier, field, body string, status int) []byte {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut, "/api/workspaces/"+ws.String()+"/issues/"+identifier+"/efforts/"+field, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if strings.HasPrefix(token, domain.TokenPrefix) {
			req.Header.Set("Authorization", "Bearer "+token)
		} else if token != "" {
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		require.Equal(t, status, rec.Code, rec.Body.String())
		return rec.Body.Bytes()
	}
	payload := func(rev int, field, value string) string {
		return fmt.Sprintf(`{"request_id":%q,"expected_revision":%d,%q:%s}`, uuid.NewString(), rev, field+"_minutes", value)
	}
	ownerJWT, adminJWT, memberJWT, guestJWT := session(owner), session(admin), session(member), session(guest)
	valid := payload(0, "estimate", "480")
	request("", "EP-1", "estimate", valid, 401)
	request(guestJWT, "EP-1", "estimate", valid, 403)
	request(memberJWT, "EP-1", "estimate", valid, 200) // secondary assignment is sufficient
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: owner, WorkspaceID: ws})
	summary, err := efforts.Summary(ctx, ws, "EP-1")
	require.NoError(t, err)
	require.EqualValues(t, 480, *summary.Own.EstimateMinutes)
	require.EqualValues(t, 360, *summary.Own.RemainingMinutes)
	require.EqualValues(t, 120, summary.Own.SpentMinutes)
	var receipt dto.EffortMutationReceipt
	require.NoError(t, json.Unmarshal(request(memberJWT, "EP-1", "estimate", valid, 200), &receipt))
	require.EqualValues(t, 1, receipt.Revision)
	request(memberJWT, "EP-1", "estimate", strings.Replace(valid, "480", "481", 1), 409)
	request(ownerJWT, "EP-1", "remaining", payload(0, "remaining", "60"), 409)
	request(adminJWT, "EP-1", "remaining", payload(1, "remaining", "0"), 200)
	request(ownerJWT, "EP-1", "estimate", payload(2, "estimate", "null"), 200)
	summary, err = efforts.Summary(ctx, ws, "EP-1")
	require.NoError(t, err)
	require.Nil(t, summary.Own.EstimateMinutes)
	require.Zero(t, *summary.Own.RemainingMinutes)
	require.EqualValues(t, 120, summary.Own.SpentMinutes)
	for _, field := range []string{"estimate", "remaining"} {
		valid := payload(3, field, "0")
		for _, body := range []string{"", "{", "null", "[]", valid + " {}", strings.TrimSuffix(valid, "}") + `,"extra":1}`, strings.Replace(valid, "0}", "0.5}", 1), strings.Replace(valid, "0}", "-1}", 1), strings.Replace(valid, "0}", "2147483648}", 1), `{"request_id":"` + uuid.NewString() + `","expected_revision":3}`} {
			request(ownerJWT, "EP-1", field, body, 400)
		}
		if field == "remaining" {
			request(ownerJWT, "EP-1", field, payload(3, field, "null"), 400)
		}
		request(ownerJWT, "missing", field, valid, 404)
		request(ownerJWT, "OTHER-1", field, valid, 404)
		request(memberJWT, "HID-1", field, valid, 404)
		raw, hash, prefix, err := domain.GenerateToken()
		require.NoError(t, err)
		require.NoError(t, tokens.Create(context.Background(), &domain.PersonalAccessToken{ID: uuid.New(), UserID: owner, Name: "EP02", TokenHash: hash, TokenPrefix: prefix, Scopes: []string{domain.PermEffortsRead}}))
		request(raw, "EP-1", field, valid, 403)
	}
	exec(`DELETE FROM issue_assignees WHERE issue_id=$1`, issue)
	request(memberJWT, "EP-1", "remaining", payload(3, "remaining", "10"), 403)
	exec(`INSERT INTO workspace_effort_settings(workspace_id,enabled) VALUES($1,FALSE)`, ws)
	request(ownerJWT, "EP-1", "estimate", payload(3, "estimate", "60"), 409)
	request(ownerJWT, "EP-1", "remaining", payload(3, "remaining", "60"), 409)
	// Committed attempts remain recognizable after disabling, without executing again.
	request(memberJWT, "EP-1", "estimate", valid, 403) // revoked assignment also blocks replay
	exec(`INSERT INTO issue_assignees(issue_id,user_id) VALUES($1,$2)`, issue, member)
	request(memberJWT, "EP-1", "estimate", valid, 200)
	exec(`UPDATE workspace_effort_settings SET enabled=TRUE WHERE workspace_id=$1`, ws)
	exec(`UPDATE issues SET status_id=$1 WHERE id=$2`, closed, issue)
	for _, value := range []string{"0", "60"} {
		request(ownerJWT, "EP-1", "remaining", payload(3, "remaining", value), 400)
	}
	request(ownerJWT, "EP-1", "estimate", payload(3, "estimate", "120"), 200)
	exec(`UPDATE issues SET status_id=$1 WHERE id=$2`, open, issue)
	request(adminJWT, "EP-1", "remaining", payload(4, "remaining", "60"), 200)
	summary, err = efforts.Summary(ctx, ws, "EP-1")
	require.NoError(t, err)
	require.EqualValues(t, 5, summary.Revision)
	require.EqualValues(t, 60, *summary.Own.RemainingMinutes)
	require.EqualValues(t, 120, summary.Own.SpentMinutes)
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM effort_operations WHERE workspace_id=$1`, ws))
	require.Equal(t, 5, count)
}
