package repository

import (
	"context"
	"database/sql"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/stretchr/testify/require"
)

func TestTimesheetOptionsPostgres(t *testing.T) {
	f := newEffortFixture(t)
	repo := NewTimesheetRepository(f.db)
	ctx := domain.WithAccess(context.Background(), domain.Access{UserID: f.member})
	b, c, outside, bTeam, private, project, privateProject := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	f.exec(t, `UPDATE workspace_members SET role='admin' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
	for _, ws := range []uuid.UUID{b, c, outside} {
		f.exec(t, `INSERT INTO workspaces(id,name,slug,owner_id) VALUES($1,'Same name',$2,$3)`, ws, ws.String(), f.owner)
	}
	f.exec(t, `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$3,'member'),($2,$3,'guest')`, b, c, f.member)
	f.exec(t, `INSERT INTO teams(id,workspace_id,name,key) VALUES($1,$2,'Effort team','B'),($3,$2,'Hidden team','H')`, bTeam, b, private)
	f.exec(t, `UPDATE teams SET is_private=TRUE WHERE id=$1`, private)
	f.exec(t, `INSERT INTO projects(id,workspace_id,team_id,name,lead_id) VALUES($1,$2,$3,'Same project',$6),($4,$2,$5,'Hidden project',NULL)`, project, b, bTeam, privateProject, private, f.member)
	f.exec(t, `INSERT INTO workspace_effort_settings(workspace_id,enabled,timezone) VALUES($1,FALSE,'Pacific/Kiritimati'),($2,TRUE,'America/Adak')`, f.ws, b)
	params := func(query string) dto.TimesheetOptionsParams {
		values, err := url.ParseQuery(query)
		require.NoError(t, err)
		p, err := dto.ParseTimesheetOptions(values)
		require.NoError(t, err)
		return p
	}
	read := func(query string) *dto.TimesheetOptionsResponse {
		result, err := repo.Options(ctx, params(query))
		require.NoError(t, err)
		return result
	}
	result := read("")
	require.ElementsMatch(t, []uuid.UUID{f.ws, b, c}, result.Scope.WorkspaceIDs)
	require.Len(t, result.Workspaces, 3)
	require.Len(t, result.Teams, 2)
	require.Len(t, result.Projects, 1)
	require.Equal(t, project, result.Projects[0].ID)
	for _, w := range result.Workspaces {
		switch w.ID {
		case f.ws:
			require.True(t, w.CanAdminister)
			require.False(t, w.CanRecordWork)
			require.Equal(t, "Pacific/Kiritimati", w.Timezone)
		case b:
			require.False(t, w.CanAdminister)
			require.True(t, w.CanRecordWork)
		case c:
			require.False(t, w.CanRecordWork)
		}
		require.NotEmpty(t, w.Today)
	}
	for _, mode := range []string{"person", "project", "workspace", "team"} {
		result := read("mode=" + mode)
		require.Equal(t, []uuid.UUID{f.ws}, result.Scope.WorkspaceIDs)
		_, err := repo.Options(ctx, params("mode="+mode+"&workspace_ids="+f.ws.String()+","+b.String()))
		require.ErrorIs(t, err, sql.ErrNoRows)
	}
	for _, query := range []string{
		"workspace_ids=" + outside.String(), "workspace_ids=" + uuid.NewString(),
		"team_ids=" + private.String(), "project_ids=" + privateProject.String(),
		"workspace_ids=" + f.ws.String() + "&team_ids=" + bTeam.String(),
		"workspace_ids=&project_ids=" + project.String(),
	} {
		result, err := repo.Options(ctx, params(query))
		require.Nil(t, result)
		require.ErrorIs(t, err, sql.ErrNoRows, query)
	}
	require.Empty(t, read("workspace_ids=").Scope.WorkspaceIDs)
	require.Empty(t, read("workspace_ids=").Teams)
	require.Empty(t, read("team_ids=").Scope.TeamIDs)
	require.Empty(t, read("project_ids=").Scope.ProjectIDs)
	require.False(t, read("project_ids=").Scope.IncludeNoProject)
	require.True(t, read("project_ids=none").Scope.IncludeNoProject)
	require.Equal(t, []uuid.UUID{project}, read("project_ids="+project.String()+",none").Scope.ProjectIDs)

	for _, denied := range []context.Context{
		context.Background(), domain.PublicContext(ctx),
		domain.WithAccess(context.Background(), domain.Access{UserID: uuid.New()}),
		domain.WithTokenScopes(ctx, []string{domain.PermProjectsRead}),
	} {
		result, err := repo.Options(denied, params("workspace_ids="+f.ws.String()))
		require.Error(t, err)
		require.Nil(t, result)
	}
	// Direct repository calls retain both a workspace-scoped Access and PAT ceilings.
	scoped := domain.WithAccess(ctx, domain.Access{UserID: f.member, WorkspaceID: b})
	result, err := repo.Options(scoped, params(""))
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{b}, result.Scope.WorkspaceIDs)
	pat := domain.WithTokenScopes(ctx, []string{domain.PermEffortsRead})
	pat = domain.WithTokenWorkspaces(pat, []string{b.String()})
	result, err = repo.Options(pat, params(""))
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{b}, result.Scope.WorkspaceIDs)
	require.False(t, result.Workspaces[0].CanRecordWork)
	_, err = repo.Options(pat, params("workspace_ids="+f.ws.String()))
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = repo.Options(pat, params("mode=person"))
	require.ErrorIs(t, err, domain.ErrEffortForbidden)
	result, err = repo.Options(domain.WithTokenWorkspaces(pat, []string{}), params(""))
	require.NoError(t, err)
	require.Empty(t, result.Workspaces)
	result, err = repo.Options(domain.WithTokenWorkspaces(pat, nil), params(""))
	require.NoError(t, err)
	require.Len(t, result.Workspaces, 3)

	// No cached role or prior successful selection survives a downgrade or revocation.
	f.exec(t, `UPDATE workspace_members SET role='member' WHERE workspace_id=$1 AND user_id=$2`, f.ws, f.member)
	_, err = repo.Options(ctx, params("mode=person"))
	require.ErrorIs(t, err, domain.ErrEffortForbidden)
	f.exec(t, `DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, b, f.member)
	_, err = repo.Options(ctx, params("workspace_ids="+b.String()))
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Len(t, read("").Workspaces, 2)
}
