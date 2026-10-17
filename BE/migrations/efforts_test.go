package migrations_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEffortMigration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	admin, err := sql.Open("pgx", url)
	require.NoError(t, err)
	name := "kuayle_effort_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(`CREATE DATABASE "` + name + `"`)
	if postgresCode(err) == "42501" {
		_ = admin.Close()
		t.Skip("isolated migration database requires CREATEDB")
	}
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), `DROP DATABASE "`+name+`" WITH (FORCE)`)
		_ = admin.Close()
	})
	testURL := databaseURLWithName(t, url, name)
	m, err := migrate.New("file://.", testURL)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = m.Close() })
	require.NoError(t, m.Up(), "fresh database")
	db, err := sql.Open("pgx", testURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	requireMigrationVersion(t, m, 47)
	require.NoError(t, m.Steps(-1), "empty Efforts rollback")
	requireMigrationVersion(t, m, 46)
	fixture := insertBaseMigrationFixture(t, db)
	legacyToken := uuid.New()
	_, err = db.Exec(`INSERT INTO personal_access_tokens(id,user_id,name,token_hash,token_prefix,scopes)
	VALUES($1,$2,'Before Efforts',repeat('a',64),'test',ARRAY['issues:read'])`, legacyToken, fixture.userA)
	require.NoError(t, err)
	require.NoError(t, m.Steps(1), "upgrade a populated existing database")
	var unchangedScopes bool
	require.NoError(t, db.QueryRow(`SELECT scopes=ARRAY['issues:read'] FROM personal_access_tokens WHERE id=$1`, legacyToken).Scan(&unchangedScopes))
	require.True(t, unchangedScopes, "existing tokens never acquire Efforts scopes")
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM issues`).Scan(&count))
	require.Equal(t, 2, count)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM pg_trigger WHERE tgname='privacy_statement_shared' AND tgrelid IN
 ('workspace_effort_settings'::regclass,'issue_efforts'::regclass,'effort_worklogs'::regclass,'effort_events'::regclass,'effort_operations'::regclass)`).Scan(&count))
	require.Equal(t, 5, count)
	_, err = db.Exec(`INSERT INTO workspace_effort_settings(workspace_id) VALUES($1)`, fixture.workspaceA)
	require.NoError(t, err)
	var enabled bool
	var zone string
	require.NoError(t, db.QueryRow(`SELECT enabled,timezone FROM workspace_effort_settings WHERE workspace_id=$1`, fixture.workspaceA).Scan(&enabled, &zone))
	require.True(t, enabled)
	require.Equal(t, "Europe/Rome", zone)
	for _, zone := range []string{"Mars/Olympus", "+02:00", "CET", "posix/Europe/Rome"} {
		_, err = db.Exec(`UPDATE workspace_effort_settings SET timezone=$1 WHERE workspace_id=$2`, zone, fixture.workspaceA)
		require.Error(t, err, zone)
	}
	_, err = db.Exec(`INSERT INTO issue_efforts(workspace_id,issue_id) VALUES($1,$2)`, fixture.workspaceA, fixture.issueA)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO issue_efforts(workspace_id,issue_id) VALUES($1,$2)`, fixture.workspaceA, fixture.issueB)
	require.Error(t, err, "cross-workspace issue reference")
	for _, value := range []int64{-1, 2147483648} {
		_, err = db.Exec(`UPDATE issue_efforts SET estimate_minutes=$1 WHERE issue_id=$2`, value, fixture.issueA)
		require.Error(t, err)
	}
	_, err = db.Exec(`UPDATE issue_efforts SET remaining_minutes=0 WHERE issue_id=$1`, fixture.issueA)
	require.NoError(t, err)
	var estimate, remaining sql.NullInt64
	require.NoError(t, db.QueryRow(`SELECT estimate_minutes,remaining_minutes FROM issue_efforts WHERE issue_id=$1`, fixture.issueA).Scan(&estimate, &remaining))
	require.False(t, estimate.Valid)
	require.True(t, remaining.Valid)
	require.Zero(t, remaining.Int64)

	owner := uuid.New()
	_, err = db.Exec(`INSERT INTO users(id,email,name,password_hash) VALUES($1,$2,'Historical person','hash')`, owner, owner.String()+"@example.test")
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'member')`, fixture.workspaceA, owner)
	require.NoError(t, err)
	insert := `INSERT INTO effort_worklogs(id,workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes,description) VALUES($1,$2,$3,$4,$4,'2026-10-16',$5,$6)`
	worklog := uuid.New()
	_, err = db.Exec(insert, worklog, fixture.workspaceA, fixture.issueA, owner, 1, strings.Repeat("界", 2000))
	require.NoError(t, err)
	for _, duration := range []int{0, -1, 1441} {
		_, err = db.Exec(insert, uuid.New(), fixture.workspaceA, fixture.issueA, owner, duration, nil)
		require.Error(t, err)
	}
	_, err = db.Exec(insert, uuid.New(), fixture.workspaceA, fixture.issueA, owner, 1, strings.Repeat("界", 2001))
	require.Error(t, err)
	_, err = db.Exec(insert, uuid.New(), fixture.workspaceA, fixture.issueB, owner, 1, nil)
	require.Error(t, err)
	for _, query := range []string{
		`UPDATE effort_worklogs SET owner_user_id=gen_random_uuid() WHERE id=$1`,
		`UPDATE effort_worklogs SET issue_id=gen_random_uuid() WHERE id=$1`,
		`UPDATE effort_worklogs SET workspace_id=gen_random_uuid() WHERE id=$1`,
		`UPDATE effort_worklogs SET created_by_user_id=gen_random_uuid() WHERE id=$1`,
	} {
		_, err = db.Exec(query, worklog)
		require.ErrorContains(t, err, "immutable")
	}
	_, err = db.Exec(`DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`, fixture.workspaceA, owner)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM users WHERE id=$1`, owner)
	require.NoError(t, err)
	var retained uuid.UUID
	require.NoError(t, db.QueryRow(`SELECT owner_user_id FROM effort_worklogs WHERE id=$1`, worklog).Scan(&retained))
	require.Equal(t, owner, retained)

	_, err = db.Exec(`INSERT INTO effort_events(workspace_id,issue_id,event_type,actor_kind,system_source,cause,after_snapshot,revision)
 VALUES($1,$2,'remaining_set','system','github','closure','{"planning":{"estimate_minutes":null,"remaining_minutes":0}}',1)`, fixture.workspaceA, fixture.issueA)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO effort_events(workspace_id,issue_id,event_type,actor_kind,cause,after_snapshot,revision)
 VALUES($1,$2,'remaining_set','system','closure','{}',1)`, fixture.workspaceA, fixture.issueA)
	require.Error(t, err, "system actor requires an origin")
	request := uuid.New()
	operation := `INSERT INTO effort_operations(workspace_id,actor_user_id,request_id,request_hash,issue_id,resource_id,revision) VALUES($1,$2,$3,decode(repeat('ab',32),'hex'),$4,$4,1)`
	_, err = db.Exec(operation, fixture.workspaceA, owner, request, fixture.issueA)
	require.NoError(t, err)
	_, err = db.Exec(operation, fixture.workspaceA, owner, request, fixture.issueA)
	require.Error(t, err, "retry receipt key is unique")

	down, err := os.ReadFile("000047_efforts.down.sql")
	require.NoError(t, err)
	tx, err := db.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(string(down))
	require.ErrorContains(t, err, "Cannot remove Efforts")
	require.NoError(t, tx.Rollback())

	child := uuid.New()
	_, err = db.Exec(`INSERT INTO issues(id,workspace_id,team_id,number,identifier_text,title,creator_id,parent_id,status_id)
	 SELECT $1,$2,$3,2,'MIGA-2','Child',$4,$5,status_id FROM issues WHERE id=$5`, child, fixture.workspaceA, fixture.teamA, fixture.userA, fixture.issueA)
	require.NoError(t, err)
	_, err = db.Exec(insert, uuid.New(), fixture.workspaceA, child, owner, 10, nil)
	require.NoError(t, err)
	_, err = db.Exec(`DELETE FROM issues WHERE id=$1`, fixture.issueA)
	require.NoError(t, err)
	for _, table := range []string{"issue_efforts", "effort_worklogs", "effort_events"} {
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE issue_id=$1`, fixture.issueA).Scan(&count))
		require.Zero(t, count, table)
	}
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM effort_worklogs WHERE issue_id=$1`, child).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM effort_operations WHERE request_id=$1`, request).Scan(&count))
	require.Equal(t, 1, count, "receipt survives issue deletion")
	_, err = db.Exec(`DELETE FROM workspaces WHERE id=$1`, fixture.workspaceA)
	require.NoError(t, err)
	for _, table := range []string{"workspace_effort_settings", "issue_efforts", "effort_worklogs", "effort_events", "effort_operations"} {
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM `+table).Scan(&count))
		require.Zero(t, count, table)
	}
	require.NoError(t, m.Steps(-1))
	requireMigrationVersion(t, m, 46)
}
