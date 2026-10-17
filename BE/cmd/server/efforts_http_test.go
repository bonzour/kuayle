package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
)

func newEffortHTTPDatabase(t *testing.T) *sqlx.DB {
	t.Helper()
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	admin, err := sql.Open("pgx", raw)
	require.NoError(t, err)
	name := "kuayle_effort_http_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(`CREATE DATABASE "` + name + `"`)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = admin.Exec(`DROP DATABASE "` + name + `" WITH (FORCE)`); _ = admin.Close() })
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	parsed.Path = "/" + name
	m, err := migrate.New("file://../../migrations", parsed.String())
	require.NoError(t, err)
	require.NoError(t, m.Up())
	_, err = m.Close()
	require.NoError(t, err)
	db, err := repository.Open(parsed.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestEffortsHTTP(t *testing.T) {
	db := newEffortHTTPDatabase(t)
	owner, guest, former := uuid.New(), uuid.New(), uuid.New()
	ws, other, pub, private, otherTeam := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	root, child, grandchild, hidden, otherIssue, empty := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	for _, user := range []uuid.UUID{owner, guest, former} {
		exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Efforts person','hash')`, user, user.String()+"@example.test")
	}
	for _, wid := range []uuid.UUID{ws, other} {
		exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Efforts workspace',$2,$3)`, wid, wid.String(), owner)
		for user, role := range map[uuid.UUID]string{owner: "owner", guest: "guest"} {
			exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,$3)`, wid, user, role)
		}
	}
	exec(`INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'Public','PUB'),($3,$2,'Private','PRIV'),($4,$5,'Other','OTHER')`, pub, ws, private, otherTeam, other)
	exec(`UPDATE teams SET is_private=TRUE WHERE id=$1`, private)
	for _, team := range []uuid.UUID{pub, private, otherTeam} {
		exec(`INSERT INTO team_statuses(team_id,name,slug,category,is_default) VALUES($1,'Todo','todo','unstarted',TRUE)`, team)
	}
	issue := func(id, wid, team uuid.UUID, number int, identifier string, parent *uuid.UUID) {
		exec(`INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,parent_id,status_id)
		SELECT $1,$2,$3,$4,$5,$5,$6,$7,id FROM team_statuses WHERE team_id=$3 AND is_default LIMIT 1`, id, wid, team, number, identifier, owner, parent)
	}
	issue(root, ws, pub, 1, "PUB-1", nil)
	issue(child, ws, pub, 2, "PUB-2", &root)
	issue(grandchild, ws, pub, 3, "PUB-3", &child)
	issue(hidden, ws, private, 1, "PRIV-1", nil)
	issue(otherIssue, other, otherTeam, 1, "OTHER-1", nil)
	issue(empty, ws, pub, 4, "PUB-4", nil)
	for _, v := range []struct {
		id                  uuid.UUID
		estimate, remaining int
	}{
		{root, 480, 360}, {child, 240, 180}, {hidden, 9999, 9999}, {otherIssue, 50, 50},
	} {
		exec(`INSERT INTO issue_efforts(workspace_id,issue_id,estimate_minutes,remaining_minutes,revision)
  SELECT workspace_id,id,$2,$3,1 FROM issues WHERE id=$1`, v.id, v.estimate, v.remaining)
	}
	insertLog := func(id, issue, who uuid.UUID, minutes int) {
		exec(`INSERT INTO effort_worklogs(id,workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes,description,created_at)
  SELECT $1,workspace_id,id,$3,$4,'2026-10-16',$5,'Private work description','2026-10-17T10:00:00Z' FROM issues WHERE id=$2`, id, issue, who, owner, minutes)
	}
	insertLog(uuid.New(), child, former, 60)
	// Equal dates and insertion times force the UUID tie-breaker over 50 rows.
	ids := make([]uuid.UUID, 55)
	for i := range ids {
		ids[i] = uuid.MustParse(fmt.Sprintf("10000000-0000-0000-0000-%012d", i+1))
		insertLog(ids[i], root, former, 1)
	}
	insertLog(uuid.New(), hidden, former, 900)
	exec(`INSERT INTO effort_events(workspace_id,issue_id,event_type,actor_kind,actor_user_id,cause,after_snapshot,revision)
	VALUES($1,$2,'worklog_created','user',$3,'manual','{"worklog":{"description":"Private work description"}}',1)`, ws, root, owner)
	exec(`DELETE FROM users WHERE id=$1`, former)
	repo := repository.NewEffortRepository(db)
	_, err := repo.Settings(context.Background(), ws)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = repo.Summary(context.Background(), ws, "PUB-1")
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = repo.Worklogs(context.Background(), ws, "PUB-1", 1)
	require.ErrorIs(t, err, sql.ErrNoRows)
	denied := domain.WithAccess(context.Background(), domain.Access{UserID: owner, WorkspaceID: other})
	_, err = repo.Summary(denied, ws, "PUB-1")
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = repo.Summary(domain.WithAccess(context.Background(), domain.Access{UserID: former, WorkspaceID: ws}), ws, "PUB-1")
	require.ErrorIs(t, err, sql.ErrNoRows, "historical identity is not workspace access")

	users := repository.NewUserRepository(db)
	workspaces := repository.NewWorkspaceRepository(db)
	teams := repository.NewTeamRepository(db)
	statuses := repository.NewTeamStatusRepository(db)
	issues := repository.NewIssueRepository(db)
	projects := repository.NewProjectRepository(db)
	cycles := repository.NewCycleRepository(db)
	tokens := repository.NewPersonalAccessTokenRepository(db)
	hub := realtime.NewHub(workspaces.RealtimeAccess)
	notifications := service.NewNotificationService(repository.NewNotificationRepository(db))
	issueService := service.NewIssueService(issues, teams, statuses, repository.NewIssueHistoryRepository(db), hub, notifications, projects)
	shared := service.NewSharedLinkService(repository.NewSharedLinkRepository(db), workspaces, teams, projects, repository.NewViewRepository(db), issues, users, statuses, "effort-http-test-secret")
	h := &appHandlers{
		effort:     handler.NewEffortHandler(service.NewEffortService(repo)),
		issue:      handler.NewIssueHandler(issueService, service.NewCommentService(repository.NewCommentRepository(db), issues, hub, notifications), users, statuses, projects, cycles, service.NewIssueRelationService(repository.NewIssueRelationRepository(db), issues)),
		sharedLink: handler.NewSharedLinkHandler(shared, "http://example.test"),
	}
	middleware := testMiddleware(mw.Auth("effort-http-test-secret", tokens))
	middleware.workspaceMembership = mw.WorkspaceMembership(workspaces)
	e := echo.New()
	registerRoutes(e, h, middleware)
	for _, route := range e.Routes() {
		if strings.Contains(route.Path, "/efforts") {
			require.Equal(t, "GET", route.Method, "no Efforts mutation routes in T-01")
		}
	}
	server := httptest.NewServer(e)
	defer server.Close()
	session := func(user uuid.UUID) string {
		token, err := jwtpkg.GenerateAccessToken(user, "effort-http-test-secret")
		require.NoError(t, err)
		return token
	}
	pat := func(scopes []string) string {
		raw, hash, prefix, err := domain.GenerateToken()
		require.NoError(t, err)
		require.NoError(t, tokens.Create(context.Background(), &domain.PersonalAccessToken{ID: uuid.New(), UserID: guest, Name: "Efforts test", TokenHash: hash, TokenPrefix: prefix, Scopes: scopes}))
		return raw
	}
	jwt, readPAT, legacyPAT, writePAT := session(guest), pat([]string{domain.PermEffortsRead}), pat([]string{domain.PermIssuesRead}), pat([]string{domain.PermEffortsWrite})
	request := func(token, path string, status int) []byte {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
		require.NoError(t, err)
		if strings.HasPrefix(token, domain.TokenPrefix) {
			req.Header.Set("Authorization", "Bearer "+token)
		} else if token != "" {
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		}
		resp, err := server.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, status, resp.StatusCode, "GET %s: %s", path, body)
		return body
	}
	base := "/api/workspaces/" + ws.String()
	summaryPath := base + "/issues/PUB-1/efforts"
	worklogPath := summaryPath + "/worklogs"
	for _, path := range []string{base + "/efforts/settings", summaryPath, worklogPath} {
		request("", path, http.StatusUnauthorized)
		request(legacyPAT, path, http.StatusForbidden)
		request(writePAT, path, http.StatusForbidden)
		request(readPAT, path, http.StatusOK)
	}
	var settings domain.EffortSettings
	require.NoError(t, json.Unmarshal(request(jwt, base+"/efforts/settings", 200), &settings))
	require.Equal(t, domain.EffortSettings{Enabled: true, Timezone: "Europe/Rome", Revision: 0}, settings)
	var summary domain.EffortSummary
	require.NoError(t, json.Unmarshal(request(jwt, summaryPath, 200), &summary))
	require.EqualValues(t, 480, *summary.Own.EstimateMinutes)
	require.EqualValues(t, 55, summary.Own.SpentMinutes)
	require.EqualValues(t, 720, *summary.Total.EstimateMinutes)
	require.EqualValues(t, 115, summary.Total.SpentMinutes)
	require.EqualValues(t, 540, summary.Total.RemainingMinutes)
	require.True(t, summary.Total.EstimatePartial, "unestimated grandchild")
	require.True(t, summary.Total.RemainingPartial)
	var logs dto.ListResponse[domain.EffortWorklog]
	require.NoError(t, json.Unmarshal(request(jwt, worklogPath, 200), &logs))
	require.Len(t, logs.Data, 50)
	require.Equal(t, 55, logs.TotalCount)
	require.True(t, logs.HasMore)
	require.Equal(t, ids[54], logs.Data[0].ID)
	require.Equal(t, ids[5], logs.Data[49].ID)
	require.Equal(t, former, logs.Data[0].Owner.ID)
	require.False(t, logs.Data[0].Owner.Available)
	require.Nil(t, logs.Data[0].Owner.Name)
	require.True(t, logs.Data[0].Operator.Available)
	require.Equal(t, domain.EffortDate("2026-10-16"), logs.Data[0].WorkDate)
	require.NoError(t, json.Unmarshal(request(jwt, worklogPath+"?page=2&per_page=50", 200), &logs))
	require.Len(t, logs.Data, 5)
	require.Equal(t, ids[4], logs.Data[0].ID)
	require.False(t, logs.HasMore)
	require.NoError(t, json.Unmarshal(request(jwt, worklogPath+"?page=3", 200), &logs))
	require.Empty(t, logs.Data)
	require.Equal(t, 55, logs.TotalCount)
	for _, query := range []string{"?page=0", "?page=-1", "?page=abc", "?page=999999999999999999999", "?per_page=100", "?per_page="} {
		request(jwt, worklogPath+query, 400)
	}
	request(jwt, base+"/issues/PRIV-1/efforts", 404)
	request(jwt, base+"/issues/PRIV-1/efforts/worklogs", 404)
	request(jwt, base+"/issues/OTHER-1/efforts", 404)
	request(jwt, base+"/issues/"+hidden.String()+"/efforts", 404)
	request(jwt, "/api/workspaces/"+other.String()+"/issues/PUB-1/efforts", 404)
	require.NoError(t, json.Unmarshal(request(jwt, base+"/issues/PUB-4/efforts", 200), &summary))
	require.Nil(t, summary.Own.EstimateMinutes)
	require.Nil(t, summary.Own.RemainingMinutes)
	require.Nil(t, summary.Total.EstimateMinutes)
	require.Zero(t, summary.Total.RemainingMinutes)
	exec(`INSERT INTO issue_efforts(workspace_id,issue_id,estimate_minutes,remaining_minutes) VALUES($1,$2,0,0)`, ws, empty)
	require.NoError(t, json.Unmarshal(request(jwt, base+"/issues/PUB-4/efforts", 200), &summary))
	require.NotNil(t, summary.Own.EstimateMinutes)
	require.Zero(t, *summary.Own.EstimateMinutes)
	require.False(t, summary.Total.EstimatePartial)
	require.False(t, summary.Total.RemainingPartial)
	exec(`INSERT INTO workspace_effort_settings(workspace_id,enabled,timezone,revision) VALUES($1,FALSE,'Pacific/Kiritimati',1)`, ws)
	require.NoError(t, json.Unmarshal(request(jwt, base+"/efforts/settings", 200), &settings))
	require.False(t, settings.Enabled)
	require.Equal(t, "Pacific/Kiritimati", settings.Timezone)
	require.NoError(t, json.Unmarshal(request(jwt, worklogPath, 200), &logs))
	require.Equal(t, domain.EffortDate("2026-10-16"), logs.Data[0].WorkDate)

	// Public and generic projections remain explicit and never join Efforts.
	for _, path := range []string{base + "/issues/PUB-1", base + "/issues", base + "/issues/PUB-1/history"} {
		body := string(request(legacyPAT, path, 200))
		for _, secret := range []string{"estimate_minutes", "spent_minutes", "remaining_minutes", "Private work description", "effort_events"} {
			require.NotContains(t, body, secret)
		}
	}
	shareToken := uuid.NewString()
	insertLog(uuid.New(), otherIssue, owner, 15)
	exec(`INSERT INTO shared_links(token,workspace_id,created_by,scope,scope_id) VALUES($1,$2,$3,'team',$4)`, shareToken, other, owner, otherTeam)
	publicBody := string(request("", "/api/public/share/"+shareToken+"/issues", 200))
	require.Contains(t, publicBody, "OTHER-1")
	require.NotContains(t, publicBody, "estimate_minutes")
	require.NotContains(t, publicBody, "remaining_minutes")
	require.NotContains(t, publicBody, "Private work description")
	request("", summaryPath+"?token="+shareToken, 401)

	// Defensive coverage of a legacy graph disallowed by today's write guards:
	// visibility must filter contributions, not prune a visible grandchild.
	exec(`ALTER TABLE issues DISABLE TRIGGER private_boundary_check`)
	exec(`UPDATE issues SET parent_id=$1 WHERE id=$2`, root, hidden)
	exec(`UPDATE issues SET parent_id=$1 WHERE id=$2`, hidden, grandchild)
	exec(`ALTER TABLE issues ENABLE TRIGGER private_boundary_check`)
	require.NoError(t, json.Unmarshal(request(jwt, summaryPath, 200), &summary))
	require.EqualValues(t, 720, *summary.Total.EstimateMinutes)
	require.EqualValues(t, 115, summary.Total.SpentMinutes)
	require.True(t, summary.Total.EstimatePartial, "visible grandchild behind hidden parent is still included")
	require.NoError(t, json.Unmarshal(request(session(owner), summaryPath, 200), &summary))
	require.EqualValues(t, 10719, *summary.Total.EstimateMinutes)
	require.EqualValues(t, 1015, summary.Total.SpentMinutes)
	exec(`ALTER TABLE issues DISABLE TRIGGER private_boundary_check`)
	exec(`UPDATE issues SET parent_id=NULL WHERE id=$1`, hidden)
	exec(`UPDATE issues SET parent_id=$1 WHERE id=$2`, child, grandchild)
	exec(`ALTER TABLE issues ENABLE TRIGGER private_boundary_check`)

	// The same old session/PAT loses reads after current membership is revoked.
	exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, ws, guest)
	request(jwt, summaryPath, 403)
	request(readPAT, summaryPath, 403)
}
