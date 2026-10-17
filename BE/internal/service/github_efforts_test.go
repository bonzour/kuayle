package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/realtime"
	"github.com/kuayle/kuayle-backend/internal/repository"
	"github.com/stretchr/testify/require"
)

// Exercise the actual GitHub service transition and parent/child automation,
// not just a repository call carrying a synthetic system context.
func TestGitHubEffortTransitionsPostgres(t *testing.T) {
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	db, err := repository.Open(raw)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	owner, ws, team, open, closed, parent, child := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(q string, args ...any) { t.Helper(); _, err := db.Exec(q, args...); require.NoError(t, err) }
	exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'GitHub Efforts','hash')`, owner, owner.String()+"@example.test")
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE id=$1`, owner) })
	exec(`INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'GitHub Efforts',$2,$3)`, ws, ws.String(), owner)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM workspaces WHERE id=$1`, ws) })
	exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'owner')`, ws, owner)
	exec(`INSERT INTO teams(id,workspace_id,name,key,parent_auto_close_enabled,sub_issue_auto_close_enabled) VALUES($1,$2,'GitHub Efforts','GHE',TRUE,TRUE)`, team, ws)
	exec(`INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','started','started'),($3,$2,'Done','done','completed')`, open, team, closed)
	for n, id := range []uuid.UUID{parent, child} {
		exec(`INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,status_id,status) VALUES($1,$2,$3,$4,$5,'GitHub Efforts',$6,$7,'started')`, id, ws, team, n+1, "GHE-"+id.String()[:8], owner, open)
	}
	exec(`UPDATE issues SET parent_id=$1 WHERE id=$2`, parent, child)
	effortRepo := repository.NewEffortRepository(db)
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: owner, WorkspaceID: ws})
	for _, id := range []uuid.UUID{parent, child} {
		rev, value := int64(0), int64(300)
		_, err := effortRepo.SetRemaining(ctx, ws, id, dto.SetEffortRemainingRequest{EffortMutationRequest: dto.EffortMutationRequest{RequestID: uuid.NewString(), ExpectedRevision: &rev}, RemainingMinutes: &value})
		require.NoError(t, err)
	}
	issueRepo := repository.NewIssueRepository(db)
	ghRepo := repository.NewGitHubRepository(db)
	require.NoError(t, ghRepo.UpsertAutoTransition(ctx, &domain.GitHubAutoTransition{ID: uuid.New(), WorkspaceID: ws, Event: "pr_merged", TargetStatus: "done", TargetStatusID: &closed, IsActive: true}))
	svc := NewGitHubService(ghRepo, issueRepo, repository.NewTeamRepository(db), repository.NewTeamStatusRepository(db), repository.NewIssueHistoryRepository(db), nil, realtime.NewHub(), "", "", nil)
	issue, err := issueRepo.GetByID(context.Background(), child)
	require.NoError(t, err)
	svc.applyAutoTransition(context.Background(), ws, "pr_merged", issue)
	for _, id := range []uuid.UUID{parent, child} {
		var value int64
		require.NoError(t, db.Get(&value, `SELECT remaining_minutes FROM issue_efforts WHERE issue_id=$1`, id))
		require.Zero(t, value)
		var source string
		require.NoError(t, db.Get(&source, `SELECT system_source FROM effort_events WHERE issue_id=$1 AND cause='closure' AND actor_kind='system' AND actor_user_id IS NULL`, id))
		require.Equal(t, "github", source)
	}
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM effort_events WHERE workspace_id=$1`, ws))
	require.Equal(t, 4, count)
	svc.applyAutoTransition(context.Background(), ws, "pr_merged", issue)
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM effort_events WHERE workspace_id=$1`, ws))
	require.Equal(t, 4, count, "repeated webhook cannot duplicate closure effects")
}
