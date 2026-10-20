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

func TestEffortWorklogsHTTP(t *testing.T) {
	db := newEffortHTTPDatabase(t)
	ws, other, owner, admin, member, guest := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	team, hiddenTeam, otherTeam, issue, hidden, foreign, second := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	for _, user := range []uuid.UUID{owner, admin, member, guest} {
		exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'EP03','hash')`, user, user.String()+"@example.test")
	}
	for _, wid := range []uuid.UUID{ws, other} {
		exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'EP03',$2,$3)`, wid, wid.String(), owner)
	}
	for user, role := range map[uuid.UUID]string{owner: "owner", admin: "admin", member: "member", guest: "guest"} {
		exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3)`, ws, user, role)
	}
	exec(`INSERT INTO teams(id,workspace_id,name,key,is_private) VALUES($1,$2,'Public','EP',FALSE),($3,$2,'Hidden','HID',TRUE),($4,$5,'Foreign','OTHER',FALSE)`, team, ws, hiddenTeam, otherTeam, other)
	for _, tid := range []uuid.UUID{team, hiddenTeam, otherTeam} {
		exec(`INSERT INTO team_statuses(team_id,name,slug,category) VALUES($1,'Open','open','started'),($1,'Closed','closed','completed')`, tid)
	}
	for _, row := range []struct {
		id, ws, team uuid.UUID
		number       int
		identifier   string
	}{{issue, ws, team, 1, "EP-1"}, {second, ws, team, 2, "EP-2"}, {hidden, ws, hiddenTeam, 1, "HID-1"}, {foreign, other, otherTeam, 1, "OTHER-1"}} {
		exec(`INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id) VALUES($1,$2,$3,$4,$5,'Worklog',$6,(SELECT id FROM team_statuses WHERE team_id=$3 AND category='started'))`, row.id, row.ws, row.team, row.number, row.identifier, owner)
	}
	exec(`INSERT INTO issue_efforts(workspace_id,issue_id,estimate_minutes,remaining_minutes,revision) VALUES($1,$2,480,480,0)`, ws, issue)
	tokens := repository.NewPersonalAccessTokenRepository(db)
	e := echo.New()
	middleware := testMiddleware(mw.Auth("ep03-secret", tokens))
	middleware.workspaceMembership = mw.WorkspaceMembership(repository.NewWorkspaceRepository(db))
	repo := repository.NewEffortRepository(db)
	registerRoutes(e, &appHandlers{effort: handler.NewEffortHandler(service.NewEffortService(repo), repository.NewIssueRepository(db))}, middleware)
	session := func(user uuid.UUID) string {
		value, err := jwtpkg.GenerateAccessToken(user, "ep03-secret")
		require.NoError(t, err)
		return value
	}
	ownerJWT, adminJWT, memberJWT, guestJWT := session(owner), session(admin), session(member), session(guest)
	request := func(token, method, path string, body any, status int) []byte {
		t.Helper()
		var data []byte
		if raw, ok := body.(string); ok {
			data = []byte(raw)
		} else if body != nil {
			var err error
			data, err = json.Marshal(body)
			require.NoError(t, err)
		}
		req := httptest.NewRequest(method, "/api/workspaces/"+ws.String()+"/issues/"+path, strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")
		if strings.HasPrefix(token, domain.TokenPrefix) {
			req.Header.Set("Authorization", "Bearer "+token)
		} else if token != "" {
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		require.Equal(t, status, rec.Code, "%s %s: %s", method, path, rec.Body.String())
		return rec.Body.Bytes()
	}
	revision := func() int64 {
		var rev int64
		require.NoError(t, db.Get(&rev, `SELECT revision FROM issue_efforts WHERE issue_id=$1`, issue))
		return rev
	}
	envelope := func() map[string]any {
		return map[string]any{"request_id": uuid.NewString(), "expected_revision": revision()}
	}
	payload := func(who uuid.UUID, duration int, date string) map[string]any {
		b := envelope()
		b["owner_user_id"] = who
		b["work_date"] = date
		b["duration_minutes"] = duration
		return b
	}
	decodeReceipt := func(data []byte) dto.EffortMutationReceipt {
		var receipt dto.EffortMutationReceipt
		require.NoError(t, json.Unmarshal(data, &receipt))
		return receipt
	}
	worklogs := "EP-1/efforts/worklogs"
	summary := func() *dto.EffortSummaryResponse {
		result, err := repo.Summary(domain.WithAccess(context.Background(), domain.Access{UserID: owner, WorkspaceID: ws}), ws, "EP-1")
		require.NoError(t, err)
		return result
	}

	valid := payload(member, 120, "2026-01-01")
	request("", "POST", worklogs, valid, 401)
	request(guestJWT, "POST", worklogs, valid, 403)
	request(memberJWT, "POST", worklogs, payload(owner, 120, "2026-01-01"), 403)
	request(ownerJWT, "POST", worklogs, payload(guest, 120, "2026-01-01"), 403)
	request(memberJWT, "POST", "HID-1/efforts/worklogs", valid, 404)
	request(ownerJWT, "POST", "OTHER-1/efforts/worklogs", valid, 404)
	created := decodeReceipt(request(memberJWT, "POST", worklogs, valid, 200)) // no assignment required
	require.EqualValues(t, 120, summary().Own.SpentMinutes)
	require.EqualValues(t, 360, *summary().Own.RemainingMinutes)
	require.Equal(t, created, decodeReceipt(request(memberJWT, "POST", worklogs, valid, 200)))
	changed := map[string]any{}
	for k, v := range valid {
		changed[k] = v
	}
	changed["duration_minutes"] = 121
	request(memberJWT, "POST", worklogs, changed, 409)
	var page dto.EffortWorklogListResponse
	require.NoError(t, json.Unmarshal(request(guestJWT, "GET", worklogs, nil, 200), &page))
	require.Equal(t, revision(), page.Revision)
	require.Len(t, page.Data, 1)
	request(memberJWT, "GET", "EP-1/efforts/worklog-owners", nil, 403)
	var candidates []domain.EffortUser
	candidatesJSON := request(adminJWT, "GET", "EP-1/efforts/worklog-owners", nil, 200)
	require.NoError(t, json.Unmarshal(candidatesJSON, &candidates))
	require.Len(t, candidates, 3)
	require.NotContains(t, string(candidatesJSON), "email")
	// Workspace admins retain access; unadmitted members are not candidates.
	exec(`INSERT INTO team_members(team_id,user_id) VALUES($1,$2)`, hiddenTeam, admin)
	require.NoError(t, json.Unmarshal(request(adminJWT, "GET", "HID-1/efforts/worklog-owners", nil, 200), &candidates))
	require.Len(t, candidates, 2)
	require.ElementsMatch(t, []uuid.UUID{owner, admin}, []uuid.UUID{candidates[0].ID, candidates[1].ID})
	request(memberJWT, "GET", "HID-1/efforts/worklog-owners", nil, 404)

	update := envelope()
	update["work_date"] = "2026-01-02"
	update["duration_minutes"] = 180
	update["description"] = "Line one\n🙂"
	path := worklogs + "/" + created.ResourceID
	request(guestJWT, "PUT", path, update, 403)
	request(adminJWT, "PUT", path, update, 200)
	require.EqualValues(t, 180, summary().Own.SpentMinutes)
	require.EqualValues(t, 360, *summary().Own.RemainingMinutes)
	request(memberJWT, "PUT", path, update, 409)
	update["expected_revision"] = revision()
	update["request_id"] = uuid.NewString()
	update["owner_user_id"] = owner
	request(ownerJWT, "PUT", path, update, 400)
	delete(update, "owner_user_id")
	delegated := decodeReceipt(request(ownerJWT, "POST", worklogs, payload(owner, 1, "2026-01-03"), 200))
	request(memberJWT, "DELETE", worklogs+"/"+delegated.ResourceID, envelope(), 403)
	request(ownerJWT, "PUT", "EP-2/efforts/worklogs/"+created.ResourceID, update, 409) // stale revision on another issue
	update["expected_revision"] = 0
	request(ownerJWT, "PUT", "EP-2/efforts/worklogs/"+created.ResourceID, update, 404)
	request(ownerJWT, "PUT", worklogs+"/invalid", update, 400)

	for _, body := range []string{"", "null", "[]", "{", `{"request_id":"` + uuid.NewString() + `","expected_revision":0,"owner_user_id":"` + owner.String() + `","work_date":"2026-02-30","duration_minutes":1}`} {
		data := request(ownerJWT, "POST", worklogs, body, 400)
		if strings.Contains(body, "2026-02-30") {
			require.Contains(t, string(data), `"field":"work_date"`)
		}
	}
	for _, duration := range []int{0, -1, 1441} {
		request(ownerJWT, "POST", worklogs, payload(owner, duration, "2026-01-01"), 400)
	}
	for _, field := range []string{"work_date", "duration_minutes", "owner_user_id"} {
		bad := payload(owner, 1, "2026-01-01")
		delete(bad, field)
		request(ownerJWT, "POST", worklogs, bad, 400)
		bad[field] = nil
		request(ownerJWT, "POST", worklogs, bad, 400)
	}
	for _, value := range []any{1.5, "1h"} {
		bad := payload(owner, 1, "2026-01-01")
		bad["duration_minutes"] = value
		request(ownerJWT, "POST", worklogs, bad, 400)
	}
	bad := payload(owner, 1, "9999-01-01")
	request(ownerJWT, "POST", worklogs, bad, 400)
	bad = payload(owner, 1, "2026-01-01")
	bad["description"] = strings.Repeat("🙂", 2001)
	request(ownerJWT, "POST", worklogs, bad, 400)
	bad["description"] = strings.Repeat("🙂", 2000)
	request(ownerJWT, "POST", worklogs, bad, 200)
	// One HTTP command creates an inclusive period and one persistent receipt.
	period := payload(owner, 480, "2026-03-27")
	period["end_date"], period["skip_weekends"] = "2026-03-30", true
	periodReceipt := decodeReceipt(request(ownerJWT, "POST", worklogs, period, 200))
	require.Equal(t, issue.String(), periodReceipt.ResourceID)
	require.Equal(t, periodReceipt, decodeReceipt(request(ownerJWT, "POST", worklogs, period, 200)))
	var periodRows int
	require.NoError(t, db.Get(&periodRows, `SELECT count(*) FROM effort_worklogs WHERE issue_id=$1 AND work_date BETWEEN '2026-03-27' AND '2026-03-30'`, issue))
	require.Equal(t, 2, periodRows)
	for _, end := range []any{"2026-02-30", "2026-01-01", 42} {
		invalid := payload(owner, 480, "2026-03-27")
		invalid["end_date"] = end
		data := request(ownerJWT, "POST", worklogs, invalid, 400)
		require.Contains(t, string(data), `"field":"end_date"`)
	}
	invalidPeriod := payload(owner, 480, "2026-03-28")
	invalidPeriod["end_date"], invalidPeriod["skip_weekends"] = "2026-03-29", true
	request(ownerJWT, "POST", worklogs, invalidPeriod, 400)
	// Hidden work counts in the daily budget without appearing in the error.
	exec(`INSERT INTO effort_worklogs(workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes) VALUES($1,$2,$3,$4,'2026-01-04',1380)`, ws, hidden, member, owner)
	dailyError := request(memberJWT, "POST", worklogs, payload(member, 61, "2026-01-04"), 400)
	require.Contains(t, string(dailyError), dto.EffortDailyLimit)
	require.NotContains(t, string(dailyError), "1380")
	require.NotContains(t, string(dailyError), hidden.String())
	require.NotContains(t, string(dailyError), "details")
	request(memberJWT, "POST", worklogs, payload(member, 60, "2026-01-04"), 200)
	// Closed issues still accept work, and never raise remaining effort.
	exec(`UPDATE issues SET status_id=(SELECT id FROM team_statuses WHERE team_id=$1 AND category='completed') WHERE id=$2`, team, issue)
	request(memberJWT, "POST", worklogs, payload(member, 1, "2026-01-05"), 200)
	require.Zero(t, *summary().Own.RemainingMinutes)
	exec(`UPDATE workspace_members SET role='guest' WHERE workspace_id=$1 AND user_id=$2`, ws, member)
	request(memberJWT, "DELETE", path, envelope(), 403)
	request(ownerJWT, "POST", worklogs, payload(member, 1, "2026-01-06"), 403)
	exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, ws, member)
	update = envelope()
	update["work_date"] = "2026-01-02"
	update["duration_minutes"] = 60
	request(adminJWT, "PUT", path, update, 200) // historical owner need not remain a member
	remove := envelope()
	removed := decodeReceipt(request(adminJWT, "DELETE", path, remove, 200))
	require.Equal(t, removed, decodeReceipt(request(adminJWT, "DELETE", path, remove, 200)))
	require.Zero(t, *summary().Own.RemainingMinutes)
	var events int
	require.NoError(t, db.Get(&events, `SELECT COUNT(*) FROM effort_events WHERE issue_id=$1 AND event_type='worklog_deleted'`, issue))
	require.Equal(t, 1, events)
	exec(`INSERT INTO workspace_effort_settings(workspace_id,enabled) VALUES($1,FALSE)`, ws)
	request(ownerJWT, "POST", worklogs, payload(owner, 1, "2026-01-07"), 409)
	request(adminJWT, "DELETE", worklogs+"/"+delegated.ResourceID, envelope(), 409)
	request(adminJWT, "PUT", worklogs+"/"+delegated.ResourceID, update, 409)
	require.Equal(t, removed, decodeReceipt(request(adminJWT, "DELETE", path, remove, 200)))
	request(guestJWT, "GET", worklogs, nil, 200)
	raw, hash, prefix, err := domain.GenerateToken()
	require.NoError(t, err)
	require.NoError(t, tokens.Create(context.Background(), &domain.PersonalAccessToken{ID: uuid.New(), UserID: owner, Name: "EP03", TokenHash: hash, TokenPrefix: prefix, Scopes: []string{domain.PermEffortsRead}}))
	for _, route := range []struct{ method, path string }{{"POST", worklogs}, {"PUT", path}, {"DELETE", path}, {"GET", "EP-1/efforts/worklog-owners"}} {
		request(raw, route.method, route.path, valid, 403)
	}
}
