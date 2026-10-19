package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/pkg/validate"
)

type effortCommand struct {
	operation   string
	request     dto.EffortMutationRequest
	issue       uuid.UUID
	worklog     uuid.UUID
	owner       uuid.UUID
	estimate    *int64
	remaining   *int64
	date        domain.EffortDate
	duration    int64
	description *string
	settings    *dto.SetEffortSettingsRequest
}

func (r *EffortRepository) SetEstimate(ctx context.Context, ws, issue uuid.UUID, req dto.SetEffortEstimateRequest) (*EffortMutationResult, error) {
	if err := validateEffortRequest(req); err != nil {
		return nil, err
	}
	return r.mutate(ctx, ws, effortCommand{operation: "estimate_set", request: req.EffortMutationRequest, issue: issue, estimate: req.EstimateMinutes.Value})
}
func (r *EffortRepository) SetRemaining(ctx context.Context, ws, issue uuid.UUID, req dto.SetEffortRemainingRequest) (*EffortMutationResult, error) {
	if err := validateEffortRequest(req); err != nil {
		return nil, err
	}
	return r.mutate(ctx, ws, effortCommand{operation: "remaining_set", request: req.EffortMutationRequest, issue: issue, remaining: req.RemainingMinutes})
}
func (r *EffortRepository) CreateWorklog(ctx context.Context, ws, issue uuid.UUID, req dto.CreateEffortWorklogRequest) (*EffortMutationResult, error) {
	if err := validateEffortRequest(req); err != nil {
		return nil, err
	}
	owner, err := uuid.Parse(req.OwnerUserID)
	if err != nil || owner == uuid.Nil {
		return nil, effortError("VALIDATION_ERROR", "owner_user_id", "invalid owner")
	}
	return r.mutate(ctx, ws, effortCommand{operation: "worklog_created", request: req.EffortMutationRequest, issue: issue, owner: owner, date: req.WorkDate, duration: req.DurationMinutes, description: req.Description})
}
func (r *EffortRepository) UpdateWorklog(ctx context.Context, ws, issue, worklog uuid.UUID, req dto.UpdateEffortWorklogRequest) (*EffortMutationResult, error) {
	if err := validateEffortRequest(req); err != nil {
		return nil, err
	}
	return r.mutate(ctx, ws, effortCommand{operation: "worklog_updated", request: req.EffortMutationRequest, issue: issue, worklog: worklog, date: req.WorkDate, duration: req.DurationMinutes, description: req.Description})
}
func (r *EffortRepository) DeleteWorklog(ctx context.Context, ws, issue, worklog uuid.UUID, req dto.DeleteEffortWorklogRequest) (*EffortMutationResult, error) {
	if err := validateEffortRequest(req); err != nil {
		return nil, err
	}
	return r.mutate(ctx, ws, effortCommand{operation: "worklog_deleted", request: req.EffortMutationRequest, issue: issue, worklog: worklog})
}
func (r *EffortRepository) SetSettings(ctx context.Context, ws uuid.UUID, req dto.SetEffortSettingsRequest) (*EffortMutationResult, error) {
	if err := validateEffortRequest(req); err != nil {
		return nil, err
	}
	if domain.IsToken(ctx) {
		return nil, domain.ErrEffortForbidden
	}
	return r.mutate(ctx, ws, effortCommand{operation: "settings_set", request: req.EffortMutationRequest, settings: &req})
}

func validateEffortRequest(req any) error {
	if err := validate.Struct(req); err != nil {
		fields := map[string]string{"RequestID": "request_id", "ExpectedRevision": "expected_revision", "OwnerUserID": "owner_user_id", "WorkDate": "work_date", "DurationMinutes": "duration_minutes", "Description": "description", "RemainingMinutes": "remaining_minutes", "Enabled": "enabled", "Timezone": "timezone", "Value": "estimate_minutes", "Set": "estimate_minutes"}
		field := ""
		if details := validate.FormatErrors(err); len(details) > 0 {
			field = fields[details[0]["field"]]
		}
		return effortError("VALIDATION_ERROR", field, "invalid Efforts request")
	}
	return nil
}

func (c effortCommand) fingerprint(ws uuid.UUID) ([]byte, error) {
	// json.Marshal sorts map keys. UUID destinations are resolved and normalized;
	// descriptions use one null representation for omission and explicit null.
	value := map[string]any{"operation": c.operation, "workspace_id": ws, "expected_revision": *c.request.ExpectedRevision}
	if c.settings != nil {
		value["enabled"] = *c.settings.Enabled
		value["timezone"] = c.settings.Timezone
	} else {
		value["issue_id"] = c.issue
	}
	switch c.operation {
	case "estimate_set":
		value["estimate_minutes"] = c.estimate
	case "remaining_set":
		value["remaining_minutes"] = c.remaining
	case "worklog_created":
		value["owner_user_id"] = c.owner
		fallthrough
	case "worklog_updated":
		value["work_date"] = c.date
		value["duration_minutes"] = c.duration
		value["description"] = c.description
	}
	if c.worklog != uuid.Nil {
		value["worklog_id"] = c.worklog
	}
	data, err := json.Marshal(value)
	hash := sha256.Sum256(data)
	return hash[:], err
}

type effortWorklogRow = domain.EffortWorklogSnapshot

func worklogTx(ctx context.Context, tx *sqlx.Tx, ws, issue, id uuid.UUID, lock bool) (*effortWorklogRow, error) {
	var row effortWorklogRow
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	err := tx.GetContext(ctx, &row, `SELECT id,owner_user_id,created_by_user_id,work_date,duration_minutes,description,created_at,updated_at
 FROM effort_worklogs WHERE workspace_id=$1 AND issue_id=$2 AND id=$3`+suffix, ws, issue, id)
	return &row, err
}

func (r *EffortRepository) mutate(ctx context.Context, ws uuid.UUID, c effortCommand) (*EffortMutationResult, error) {
	access, ok := domain.AccessFromContext(ctx)
	if !ok || access.UserID == uuid.Nil || access.WorkspaceID != uuid.Nil && access.WorkspaceID != ws {
		return nil, domain.ErrEffortForbidden
	}
	if err := domain.CheckEffortToken(ctx); err != nil {
		return nil, err
	}
	requestID, err := uuid.Parse(c.request.RequestID)
	if err != nil || requestID == uuid.Nil {
		return nil, effortError("VALIDATION_ERROR", "request_id", "invalid request_id")
	}
	if c.settings == nil && c.issue == uuid.Nil {
		return nil, sql.ErrNoRows
	}
	hash, err := c.fingerprint(ws)
	if err != nil {
		return nil, err
	}
	tx, err := beginEffortTx(ctx, r.db, ws, c.settings != nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,471))`, ws.String()+"/"+access.UserID.String()+"/"+requestID.String()); err != nil {
		return nil, err
	}
	var receipt struct {
		Hash     []byte     `db:"request_hash"`
		Resource uuid.UUID  `db:"resource_id"`
		Issue    *uuid.UUID `db:"issue_id"`
		Revision int64      `db:"revision"`
	}
	err = tx.GetContext(ctx, &receipt, `SELECT request_hash,resource_id,issue_id,revision FROM effort_operations WHERE workspace_id=$1 AND actor_user_id=$2 AND request_id=$3`, ws, access.UserID, requestID)
	replay := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	// For a reused key, authorize the requested issue before revealing a conflict.
	var issue domain.Issue
	teams := []uuid.UUID{}
	if c.settings == nil {
		if err = tx.GetContext(ctx, &issue, `SELECT * FROM issues WHERE workspace_id=$1 AND id=$2`, ws, c.issue); err != nil {
			return nil, err
		}
		teams = append(teams, issue.TeamID)
	}
	users := []uuid.UUID{access.UserID}
	if c.operation == "worklog_created" {
		users = append(users, c.owner)
	}
	if err = lockEffortMembers(ctx, tx, ws, teams, users); err != nil {
		return nil, err
	}
	role, err := effortRole(ctx, tx, ws, access.UserID)
	if err != nil {
		return nil, err
	}
	admin := role == "owner" || role == "admin"
	if role == "guest" || !admin && role != "member" {
		return nil, domain.ErrEffortForbidden
	}
	if c.settings != nil {
		if !admin || domain.IsToken(ctx) {
			return nil, domain.ErrEffortForbidden
		}
	} else {
		issues, e := lockEffortIssues(ctx, tx, ws, []uuid.UUID{c.issue}, nil, nil)
		if e != nil {
			return nil, e
		}
		if len(issues) != 1 {
			return nil, sql.ErrNoRows
		}
		issue = issues[0]
		if err = requireVisible(ctx, tx, effortWorkspaceVisible(ctx, "'"+ws.String()+"'::uuid")+" AND "+teamVisible(ctx, uuidSQL(issue.TeamID))); err != nil {
			return nil, err
		}
	}
	settings, err := effortSettingsTx(ctx, tx, ws)
	if err != nil {
		return nil, err
	}
	var before domain.EffortPlanningSnapshot
	var currentRevision int64
	if c.settings == nil {
		if !replay && !settings.Enabled {
			return nil, effortError(dto.EffortDisabled, "", "Efforts is disabled")
		}
		var e error
		before, currentRevision, e = effortPlanningTx(ctx, tx, ws, c.issue)
		if e != nil {
			return nil, e
		}
		if !replay && currentRevision != *c.request.ExpectedRevision {
			return nil, effortError(dto.EffortRevisionConflict, "expected_revision", "Efforts changed")
		}
	}
	var old *effortWorklogRow
	if c.worklog != uuid.Nil {
		old, err = worklogTx(ctx, tx, ws, c.issue, c.worklog, false)
		if errors.Is(err, sql.ErrNoRows) && replay && bytes.Equal(hash, receipt.Hash) {
			// Receipts deliberately have no personal data. Retained event snapshots
			// supply ownership even if a worklog was deleted after this command.
			var snapshot []byte
			err = tx.GetContext(ctx, &snapshot, `SELECT COALESCE(after_snapshot,before_snapshot) FROM effort_events WHERE workspace_id=$1 AND issue_id=$2
    AND event_type=$5 AND revision=$3 AND COALESCE(after_snapshot,before_snapshot)->'worklog'->>'id'=$4`, ws, c.issue, receipt.Revision, c.worklog.String(), c.operation)
			if err == nil {
				var s domain.EffortSnapshot
				err = json.Unmarshal(snapshot, &s)
				old = s.Worklog
				if old == nil && err == nil {
					err = sql.ErrNoRows
				}
			}
		}
		if err != nil {
			return nil, err
		}
		if !admin && old.OwnerUserID != access.UserID {
			return nil, domain.ErrEffortForbidden
		}
	}
	if c.operation == "estimate_set" || c.operation == "remaining_set" {
		var assigned bool
		if err = tx.GetContext(ctx, &assigned, `SELECT COALESCE(assignee_id=$2,FALSE) OR EXISTS(SELECT 1 FROM issue_assignees WHERE issue_id=$1 AND user_id=$2) FROM issues WHERE id=$1`, c.issue, access.UserID); err != nil {
			return nil, err
		}
		if !admin && !assigned {
			return nil, domain.ErrEffortForbidden
		}
	}
	if c.operation == "worklog_created" {
		if !admin && c.owner != access.UserID {
			return nil, domain.ErrEffortForbidden
		}
		ownerRole, e := effortRole(ctx, tx, ws, c.owner)
		if e != nil {
			return nil, domain.ErrEffortForbidden
		}
		if ownerRole != "owner" && ownerRole != "admin" && ownerRole != "member" {
			return nil, domain.ErrEffortForbidden
		}
		ownerCtx := domain.WithAccess(ctx, domain.Access{UserID: c.owner, WorkspaceID: ws})
		if err = requireVisible(ownerCtx, tx, teamVisible(ownerCtx, uuidSQL(issue.TeamID))); err != nil {
			return nil, domain.ErrEffortForbidden
		}
	}
	if replay {
		if !bytes.Equal(hash, receipt.Hash) {
			return nil, effortError(dto.EffortRequestConflict, "request_id", "request_id was already used for another command")
		}
		return &EffortMutationResult{Receipt: dto.EffortMutationReceipt{RequestID: requestID.String(), ResourceID: receipt.Resource.String(), Revision: receipt.Revision}, Replay: true}, nil
	}
	var resource uuid.UUID
	var revision int64
	actor := domain.EffortActor{Kind: "user", UserID: &access.UserID}
	if c.settings != nil {
		resource = ws
		revision = settings.Revision + 1
		if settings.Revision != *c.request.ExpectedRevision {
			return nil, effortError(dto.EffortRevisionConflict, "expected_revision", "Efforts settings changed")
		}
		if err = r.settingsCommand(ctx, tx, ws, c, settings, revision); err != nil {
			return nil, err
		}
	} else {
		revision = currentRevision + 1
		resource = c.issue
		after := before
		terminal, e := effortTerminal(ctx, tx, &issue)
		if e != nil {
			return nil, e
		}
		switch c.operation {
		case "estimate_set":
			after.EstimateMinutes = c.estimate
			if c.estimate != nil && after.RemainingMinutes == nil {
				var spent int64
				if err = tx.GetContext(ctx, &spent, `SELECT COALESCE(SUM(duration_minutes),0) FROM effort_worklogs WHERE workspace_id=$1 AND issue_id=$2`, ws, c.issue); err != nil {
					return nil, err
				}
				remaining := max(*c.estimate-spent, 0)
				after.RemainingMinutes = &remaining
			}
		case "remaining_set":
			if terminal {
				return nil, effortError("VALIDATION_ERROR", "remaining_minutes", "remaining effort cannot be edited on closed issues")
			}
			after.RemainingMinutes = c.remaining
		default:
			resource, err = r.worklogCommand(ctx, tx, ws, c, old, actor, revision, settings.Timezone)
			if err != nil {
				return nil, err
			}
			if c.operation == "worklog_created" && after.RemainingMinutes != nil {
				remaining := max(*after.RemainingMinutes-c.duration, 0)
				after.RemainingMinutes = &remaining
			}
		}
		if terminal {
			zero := int64(0)
			after.RemainingMinutes = &zero
		}
		if err = saveEffortPlanning(ctx, tx, ws, c.issue, after, revision); err != nil {
			return nil, err
		}
		if c.operation == "estimate_set" {
			if err = writeEffortEvent(ctx, tx, ws, c.issue, actor, "estimate_set", "manual", planningSnapshot(before), planningSnapshot(after), revision); err != nil {
				return nil, err
			}
		}
		if c.operation == "remaining_set" || !sameMinutes(before.RemainingMinutes, after.RemainingMinutes) {
			cause := "manual"
			if c.operation == "worklog_created" {
				cause = "worklog"
			}
			if err = writeEffortEvent(ctx, tx, ws, c.issue, actor, "remaining_set", cause, planningSnapshot(before), planningSnapshot(after), revision); err != nil {
				return nil, err
			}
		}
	}
	if revision > domain.EffortMaxSafeInteger {
		return nil, fmt.Errorf("Efforts revision overflow")
	}
	var issueID *uuid.UUID
	if c.settings == nil {
		issueID = &c.issue
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO effort_operations(workspace_id,actor_user_id,request_id,request_hash,issue_id,resource_id,revision) VALUES($1,$2,$3,$4,$5,$6,$7)`, ws, access.UserID, requestID, hash, issueID, resource, revision)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &EffortMutationResult{Receipt: dto.EffortMutationReceipt{RequestID: requestID.String(), ResourceID: resource.String(), Revision: revision}}, nil
}

func (r *EffortRepository) worklogCommand(ctx context.Context, tx *sqlx.Tx, ws uuid.UUID, c effortCommand, old *effortWorklogRow, actor domain.EffortActor, revision int64, timezone string) (uuid.UUID, error) {
	owner := c.owner
	dates := []domain.EffortDate{c.date}
	if old != nil {
		owner = old.OwnerUserID
		dates = append(dates, old.WorkDate)
	}
	if c.operation == "worklog_deleted" {
		dates = []domain.EffortDate{old.WorkDate}
	}
	if err := lockEffortBudgets(ctx, tx, ws, owner, dates...); err != nil {
		return uuid.Nil, err
	}
	if c.operation != "worklog_deleted" {
		if _, err := domain.ParseEffortDate(string(c.date)); err != nil {
			return uuid.Nil, effortError("VALIDATION_ERROR", "work_date", err.Error())
		}
		if c.duration < 1 || c.duration > 1440 {
			return uuid.Nil, effortError("VALIDATION_ERROR", "duration_minutes", "duration must be between 1 and 1440")
		}
		if c.description != nil && utf8.RuneCountInString(*c.description) > 2000 {
			return uuid.Nil, effortError("VALIDATION_ERROR", "description", "description exceeds 2000 characters")
		}
		var future bool
		if err := tx.GetContext(ctx, &future, `SELECT $1::date>(clock_timestamp() AT TIME ZONE $2)::date`, c.date, timezone); err != nil {
			return uuid.Nil, err
		}
		if future {
			return uuid.Nil, effortError("VALIDATION_ERROR", "work_date", "work_date cannot be in the future")
		}
		var total int64
		if err := tx.GetContext(ctx, &total, `SELECT COALESCE(SUM(duration_minutes),0) FROM effort_worklogs WHERE workspace_id=$1 AND owner_user_id=$2 AND work_date=$3 AND id<>$4`, ws, owner, c.date, c.worklog); err != nil {
			return uuid.Nil, err
		}
		if total+c.duration > 1440 {
			return uuid.Nil, effortError(dto.EffortDailyLimit, "duration_minutes", "daily effort limit exceeded")
		}
	}
	var before, after *domain.EffortSnapshot
	resource := c.worklog
	if old != nil {
		locked, err := worklogTx(ctx, tx, ws, c.issue, c.worklog, true)
		if err != nil {
			return uuid.Nil, err
		}
		old = locked
		before = &domain.EffortSnapshot{Worklog: old}
	}
	switch c.operation {
	case "worklog_created":
		resource = uuid.New()
		_, err := tx.ExecContext(ctx, `INSERT INTO effort_worklogs(id,workspace_id,issue_id,owner_user_id,created_by_user_id,work_date,duration_minutes,description) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, resource, ws, c.issue, owner, actor.UserID, c.date, c.duration, c.description)
		if err != nil {
			return uuid.Nil, err
		}
	case "worklog_updated":
		if _, err := tx.ExecContext(ctx, `UPDATE effort_worklogs SET work_date=$1,duration_minutes=$2,description=$3,updated_at=clock_timestamp() WHERE id=$4 AND workspace_id=$5 AND issue_id=$6`, c.date, c.duration, c.description, resource, ws, c.issue); err != nil {
			return uuid.Nil, err
		}
	case "worklog_deleted":
		if _, err := tx.ExecContext(ctx, `DELETE FROM effort_worklogs WHERE id=$1 AND workspace_id=$2 AND issue_id=$3`, resource, ws, c.issue); err != nil {
			return uuid.Nil, err
		}
	}
	if c.operation != "worklog_deleted" {
		row, err := worklogTx(ctx, tx, ws, c.issue, resource, false)
		if err != nil {
			return uuid.Nil, err
		}
		after = &domain.EffortSnapshot{Worklog: row}
	}
	if err := writeEffortEvent(ctx, tx, ws, c.issue, actor, c.operation, "manual", before, after, revision); err != nil {
		return uuid.Nil, err
	}
	return resource, nil
}

func (r *EffortRepository) settingsCommand(ctx context.Context, tx *sqlx.Tx, ws uuid.UUID, c effortCommand, previous domain.EffortSettings, revision int64) error {
	var valid bool
	if err := tx.GetContext(ctx, &valid, `SELECT EXISTS(SELECT 1 FROM pg_timezone_names WHERE name=$1) AND ($1='UTC' OR position('/' IN $1)>0) AND $1 NOT LIKE 'posix/%' AND $1 NOT LIKE 'right/%'`, c.settings.Timezone); err != nil {
		return err
	}
	if !valid {
		return effortError("VALIDATION_ERROR", "timezone", "invalid IANA timezone")
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO workspace_effort_settings(workspace_id,enabled,timezone,revision) VALUES($1,$2,$3,$4)
 ON CONFLICT(workspace_id) DO UPDATE SET enabled=EXCLUDED.enabled,timezone=EXCLUDED.timezone,revision=EXCLUDED.revision,updated_at=clock_timestamp()`, ws, *c.settings.Enabled, c.settings.Timezone, revision)
	if err != nil {
		return err
	}
	if !previous.Enabled && *c.settings.Enabled {
		var ids []uuid.UUID
		if err = tx.SelectContext(ctx, &ids, `SELECT issue_id FROM issue_efforts WHERE workspace_id=$1 ORDER BY issue_id`, ws); err != nil {
			return err
		}
		issues, e := lockEffortIssues(ctx, tx, ws, ids, nil, nil)
		if e != nil {
			return e
		}
		for i := range issues {
			if err = closeEffortTx(ctx, tx, &issues[i], "reactivation"); err != nil {
				return err
			}
		}
	}
	return nil
}
