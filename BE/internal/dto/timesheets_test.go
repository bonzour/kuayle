package dto

import (
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTimesheetIssueParams(t *testing.T) {
	ws := uuid.NewString()
	values, err := url.ParseQuery("workspace_id=" + ws + "&search=+EP-05+&page=2&per_page=50")
	require.NoError(t, err)
	p, err := ParseTimesheetIssues(values)
	require.NoError(t, err)
	require.Equal(t, uuid.MustParse(ws), p.WorkspaceID)
	require.Equal(t, "EP-05", p.Search)
	require.Equal(t, 2, p.Page)
	for _, query := range []string{"", "workspace_id=bad", "workspace_id=" + uuid.Nil.String(),
		"workspace_id=" + ws + "&page=0", "workspace_id=" + ws + "&page=1.5", "workspace_id=" + ws + "&page=9007199254740991",
		"workspace_id=" + ws + "&per_page=51", "workspace_id=" + ws + "&search=a&search=b", "workspace_id=" + ws + "&mode=person"} {
		values, err := url.ParseQuery(query)
		require.NoError(t, err)
		_, err = ParseTimesheetIssues(values)
		require.ErrorIs(t, err, ErrTimesheetParams, query)
	}
}

func TestTimesheetSelections(t *testing.T) {
	id := uuid.New()
	for _, tc := range []struct {
		query string
		all   bool
		ids   []uuid.UUID
		none  bool
	}{
		{"", true, []uuid.UUID{}, false},
		{"project_ids=all", true, []uuid.UUID{}, false},
		{"project_ids=", false, []uuid.UUID{}, false},
		{"project_ids=none", false, []uuid.UUID{}, true},
		{"project_ids=" + id.String() + ",none," + id.String(), false, []uuid.UUID{id}, true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			values, err := url.ParseQuery(tc.query)
			require.NoError(t, err)
			p, err := ParseTimesheetOptions(values)
			require.NoError(t, err)
			require.Equal(t, "personal", p.Mode)
			require.Equal(t, TimesheetSelection{All: tc.all, IDs: tc.ids, IncludeNone: tc.none}, p.Projects)
		})
	}
	for _, query := range []string{
		"mode=", "mode=admin", "mode=personal&mode=team", "team_ids=none", "workspace_ids=none",
		"project_ids=all,none", "team_ids=bad", "workspace_ids=" + uuid.Nil.String(),
		"project_ids=none,", "workspace_ids=&workspace_ids=all", "owner_user_ids=" + id.String(),
	} {
		values, err := url.ParseQuery(query)
		require.NoError(t, err)
		_, err = ParseTimesheetOptions(values)
		require.ErrorIs(t, err, ErrTimesheetParams, query)
	}
}

func TestTimesheetReadPeriods(t *testing.T) {
	base := "reference_workspace_id=" + uuid.NewString()
	for _, query := range []string{
		"period=week&from=2025-12-29&to=2026-01-04",
		"period=week&from=2026-03-23&to=2026-03-29",
		"period=month&from=2024-02-01&to=2024-02-29",
		"period=month&from=2026-02-01&to=2026-02-28",
	} {
		values, _ := url.ParseQuery(base + "&" + query)
		p, err := ParseTimesheetRead(values, false)
		require.NoError(t, err, query)
		require.Equal(t, 1, p.Page)
	}
	valid := base + "&period=week&from=2026-10-19&to=2026-10-25"
	for _, suffix := range []string{
		"&page=0", "&page=-1", "&page=9007199254740991", "&per_page=51", "&unknown=x", "&period=month",
		"&issue_id=" + uuid.NewString(), "&work_date=2026-10-20", "&owner_user_ids=all",
	} {
		values, _ := url.ParseQuery(valid + suffix)
		_, err := ParseTimesheetRead(values, false)
		require.ErrorIs(t, err, ErrTimesheetParams, suffix)
	}
	for _, query := range []string{
		"period=week&from=2026-10-20&to=2026-10-26",
		"period=week&from=2026-10-19&to=2026-10-24",
		"period=month&from=2023-02-01&to=2023-02-29",
		"period=month&from=2024-02-02&to=2024-02-29",
		"period=month&from=2024-02-01&to=2024-03-01",
		"period=week&from=0000-01-03&to=0000-01-09",
		"period=other&from=2026-10-19&to=2026-10-25",
	} {
		values, _ := url.ParseQuery(base + "&" + query)
		_, err := ParseTimesheetRead(values, false)
		require.ErrorIs(t, err, ErrTimesheetParams, query)
	}
	values, _ := url.ParseQuery(valid + "&issue_id=" + uuid.NewString() + "&work_date=2026-10-20&page=2&per_page=50")
	p, err := ParseTimesheetRead(values, true)
	require.NoError(t, err)
	require.Equal(t, 2, p.Page)
	values.Set("work_date", "2026-10-26")
	_, err = ParseTimesheetRead(values, true)
	require.ErrorIs(t, err, ErrTimesheetParams)
	values.Del("reference_workspace_id")
	_, err = ParseTimesheetRead(values, true)
	require.ErrorIs(t, err, ErrTimesheetParams)
}

func TestTimesheetAdministrativeOwnerSelections(t *testing.T) {
	base := "period=week&from=2026-01-05&to=2026-01-11&reference_workspace_id=" + uuid.NewString()
	owner := uuid.NewString()
	for _, mode := range []string{"person", "project", "workspace", "team"} {
		for _, selection := range []string{"", "&owner_user_ids=all", "&owner_user_ids=", "&owner_user_ids=" + owner + "," + owner} {
			values, _ := url.ParseQuery(base + "&mode=" + mode + selection)
			p, err := ParseTimesheetRead(values, false)
			require.NoError(t, err)
			if selection == "&owner_user_ids=" {
				require.NotNil(t, p.OwnerIDs)
				require.Empty(t, p.OwnerIDs)
			}
			if selection == "" || selection == "&owner_user_ids=all" {
				require.Nil(t, p.OwnerIDs)
			}
			if selection == "&owner_user_ids="+owner+","+owner {
				require.Len(t, p.OwnerIDs, 1)
			}
		}
		for _, invalid := range []string{"all," + owner, "none", "bad", owner + ",", uuid.Nil.String()} {
			values, _ := url.ParseQuery(base + "&mode=" + mode + "&owner_user_ids=" + invalid)
			_, err := ParseTimesheetRead(values, false)
			require.ErrorIs(t, err, ErrTimesheetParams)
		}
	}
}
