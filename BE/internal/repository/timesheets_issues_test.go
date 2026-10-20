package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func TestTimesheetIssuesPostgres(t *testing.T) {
	f := newEffortFixture(t)
	repo := NewTimesheetRepository(f.db)
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: f.member})
	for n := 0; n < 51; n++ {
		f.exec(t, `INSERT INTO issues(workspace_id,team_id,number,identifier_text,title,creator_id,status_id,updated_at)
 VALUES($1,$2,$3,$4,'EP05 unlogged',$5,$6,'2026-01-01')`, f.ws, f.team, n+10, fmt.Sprintf("EP05-%02d", n), f.owner, f.status)
	}
	hiddenTeam, hiddenStatus := uuid.New(), uuid.New()
	f.exec(t, `INSERT INTO teams(id,workspace_id,name,key,is_private) VALUES($1,$2,'Secret EP05','SECRET',TRUE)`, hiddenTeam, f.ws)
	f.exec(t, `INSERT INTO team_statuses(id,team_id,name,slug,category) VALUES($1,$2,'Open','open','started')`, hiddenStatus, hiddenTeam)
	f.exec(t, `INSERT INTO issues(workspace_id,team_id,number,identifier_text,title,creator_id,status_id) VALUES($1,$2,1,'SECRET-1','EP05 secret',$3,$4)`, f.ws, hiddenTeam, f.owner, hiddenStatus)
	p := dto.TimesheetIssueParams{WorkspaceID: f.ws, Search: "ep05", Page: 1}
	first, err := repo.Issues(ctx, p)
	require.NoError(t, err)
	require.Equal(t, 51, first.TotalCount)
	require.Len(t, first.Data, 50)
	require.True(t, first.HasMore)
	require.True(t, first.Workspace.CanRecordWork)
	seen := map[uuid.UUID]bool{}
	for _, issue := range first.Data {
		require.Equal(t, f.team, issue.Team.ID)
		seen[issue.ID] = true
	}
	p.Page = 2
	second, err := repo.Issues(ctx, p)
	require.NoError(t, err)
	require.Len(t, second.Data, 1)
	require.False(t, second.HasMore)
	require.False(t, seen[second.Data[0].ID])
	again, err := repo.Issues(ctx, p)
	require.NoError(t, err)
	require.Equal(t, second.Data, again.Data)
	p.Search = second.Data[0].Identifier
	p.Page = 1
	matched, err := repo.Issues(ctx, p)
	require.NoError(t, err)
	require.Equal(t, 1, matched.TotalCount)
	p.Search = "SECRET"
	matched, err = repo.Issues(ctx, p)
	require.NoError(t, err)
	require.Empty(t, matched.Data)
	p.Search = ""
	f.exec(t, `INSERT INTO workspace_effort_settings(workspace_id,enabled,timezone) VALUES($1,FALSE,'Pacific/Kiritimati')`, f.ws)
	readonly, err := repo.Issues(ctx, p)
	require.NoError(t, err)
	require.False(t, readonly.Workspace.CanRecordWork)
	require.Equal(t, "Pacific/Kiritimati", readonly.Workspace.Timezone)
	require.Equal(t, 54, readonly.TotalCount)
	guest := domain.WithAccess(context.Background(), domain.Access{UserID: f.guest})
	readonly, err = repo.Issues(guest, p)
	require.NoError(t, err)
	require.False(t, readonly.Workspace.CanRecordWork)
	read := domain.WithTokenScopes(ctx, []string{domain.PermEffortsRead})
	_, err = repo.Issues(domain.WithTokenWorkspaces(read, []string{}), p)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = repo.Issues(domain.WithTokenWorkspaces(read, []string{"outside"}), p)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = repo.Issues(domain.WithTokenScopes(ctx, []string{domain.PermIssuesRead}), p)
	require.ErrorIs(t, err, domain.ErrEffortForbidden)
	_, err = repo.Issues(context.Background(), p)
	require.ErrorIs(t, err, domain.ErrEffortForbidden)
	p.WorkspaceID = uuid.New()
	_, err = repo.Issues(ctx, p)
	require.ErrorIs(t, err, sql.ErrNoRows)
	p.WorkspaceID = f.ws
	f.exec(t, `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
	_, err = repo.Issues(ctx, p)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
