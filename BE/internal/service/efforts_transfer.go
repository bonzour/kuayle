package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
)

func isEffortTransferTable(table string) bool {
	switch table {
	case "workspace_effort_settings", "issue_efforts", "effort_worklogs", "effort_events":
		return true
	}
	return false
}

func effortArchiveSnapshots(row dto.WorkspaceTransferRow) ([]domain.EffortSnapshot, error) {
	var snapshots []domain.EffortSnapshot
	for _, field := range []string{"before_snapshot", "after_snapshot"} {
		if row[field] == nil {
			continue
		}
		raw, err := json.Marshal(row[field])
		if err != nil {
			return nil, err
		}
		var snapshot domain.EffortSnapshot
		if err = json.Unmarshal(raw, &snapshot); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func collectEffortArchiveUsers(data dto.WorkspaceTransferData, ids map[uuid.UUID]bool) {
	for _, row := range data.Tables["effort_events"] {
		snapshots, _ := effortArchiveSnapshots(row)
		for _, snapshot := range snapshots {
			if w := snapshot.Worklog; w != nil {
				ids[w.OwnerUserID] = true
				ids[w.CreatedByUserID] = true
			}
		}
	}
}

// Historical UUIDs are remapped without manufacturing accounts or memberships.
// Allocate deleted worklog IDs as well, so every snapshot refers to the same copy.
func prepareEffortIdentities(parsed *parsedWorkspaceArchive, mapping map[string]string, seen map[string]string) error {
	for _, row := range parsed.data.UnavailableUsers {
		id := transferString(row["id"])
		if mapping[id] != "" {
			return invalidArchive("historical user ID collides with another identity", nil)
		}
		mapping[id] = uuid.NewString()
	}
	for _, row := range parsed.data.Tables["effort_events"] {
		snapshots, err := effortArchiveSnapshots(row)
		if err != nil {
			return invalidArchive("invalid Efforts snapshot", err)
		}
		for _, snapshot := range snapshots {
			if w := snapshot.Worklog; w != nil {
				id := w.ID.String()
				if mapping[id] != "" && seen[id] != "effort_worklogs" {
					return invalidArchive("worklog snapshot ID collides with another identity", nil)
				}
				if mapping[id] == "" {
					mapping[id] = uuid.NewString()
					seen[id] = "effort_worklogs"
				}
			}
		}
	}
	return nil
}

// Only typed references are remapped; free text, even a literal UUID, is preserved.
func remapEffortRow(source dto.WorkspaceTransferRow, mapping map[string]string) dto.WorkspaceTransferRow {
	row := cloneTransferRow(source)
	for _, key := range []string{"id", "workspace_id", "issue_id", "owner_user_id", "created_by_user_id", "actor_user_id"} {
		if id := transferString(row[key]); id != "" {
			row[key] = mapping[id]
		}
	}
	for _, field := range []string{"before_snapshot", "after_snapshot"} {
		if row[field] == nil {
			continue
		}
		raw, _ := json.Marshal(row[field])
		var snapshot domain.EffortSnapshot
		_ = json.Unmarshal(raw, &snapshot)
		if w := snapshot.Worklog; w != nil {
			w.ID = uuid.MustParse(mapping[w.ID.String()])
			w.OwnerUserID = uuid.MustParse(mapping[w.OwnerUserID.String()])
			w.CreatedByUserID = uuid.MustParse(mapping[w.CreatedByUserID.String()])
		}
		raw, _ = json.Marshal(snapshot)
		var value any
		_ = dto.DecodeTransferJSON(raw, &value)
		row[field] = value
	}
	return row
}

func effortArchiveInteger(value any, min, max int64) bool {
	n, err := strconv.ParseInt(transferString(value), 10, 64)
	return err == nil && n >= min && n <= max
}

func validateEffortArchive(parsed *parsedWorkspaceArchive) error {
	data := parsed.data
	bad := func(message string) error { return invalidArchive("Efforts: "+message, nil) }
	if parsed.manifest.Version == 1 {
		if len(data.UnavailableUsers) > 0 {
			return bad("legacy archive contains historical identities")
		}
		for table, rows := range data.Tables {
			if isEffortTransferTable(table) && len(rows) > 0 {
				return bad("legacy archive contains Efforts data")
			}
		}
	}
	if parsed.manifest.Version >= 2 && parsed.manifest.Counts["unavailable_users"] != len(data.UnavailableUsers) {
		return bad("historical user count mismatch")
	}
	users := map[string]bool{}
	for _, row := range data.Users {
		users[transferString(row["id"])] = true
	}
	unavailable := map[string]bool{}
	for _, row := range data.UnavailableUsers {
		id := transferString(row["id"])
		value, err := uuid.Parse(id)
		if err != nil || value == uuid.Nil || value.String() != id || users[id] || unavailable[id] {
			return bad("invalid historical user")
		}
		unavailable[id] = true
		users[id] = true
	}
	// Historical references may only appear in Efforts. Never turn them into access.
	for table, rows := range data.Tables {
		if !isEffortTransferTable(table) {
			for _, row := range rows {
				for field, value := range row {
					if strings.HasSuffix(field, "_id") && unavailable[transferString(value)] {
						return bad("historical identity grants access or appears outside Efforts")
					}
				}
			}
		}
	}
	issues := map[string]bool{}
	for _, row := range data.Tables["issues"] {
		issues[transferString(row["id"])] = true
	}
	settings := data.Tables["workspace_effort_settings"]
	if len(settings) > 1 {
		return bad("multiple workspace configurations")
	}
	for _, row := range settings {
		if _, ok := row["enabled"].(bool); !ok {
			return bad("enabled must be a boolean")
		}
		zone := transferString(row["timezone"])
		_, err := time.LoadLocation(zone)
		if err != nil || zone != "UTC" && (!strings.Contains(zone, "/") || strings.HasPrefix(zone, "posix/") || strings.HasPrefix(zone, "right/")) {
			return bad("invalid workspace timezone")
		}
		if !effortArchiveInteger(row["revision"], 0, domain.EffortMaxSafeInteger) {
			return bad("invalid settings revision")
		}
	}
	revisions := map[string]int64{}
	for _, row := range data.Tables["issue_efforts"] {
		id := transferString(row["issue_id"])
		if _, exists := revisions[id]; exists {
			return bad("duplicate issue planning")
		}
		if !effortArchiveInteger(row["revision"], 0, domain.EffortMaxSafeInteger) {
			return bad("invalid issue revision")
		}
		revisions[id] = transferInt64(row["revision"])
		for _, key := range []string{"estimate_minutes", "remaining_minutes"} {
			if row[key] != nil && !effortArchiveInteger(row[key], 0, domain.EffortMaxMinutes) {
				return bad("invalid " + key)
			}
		}
	}
	identities := map[uuid.UUID]domain.EffortWorklogSnapshot{}
	validWorklog := func(w *domain.EffortWorklogSnapshot) bool {
		if old, ok := identities[w.ID]; ok && (old.OwnerUserID != w.OwnerUserID || old.CreatedByUserID != w.CreatedByUserID || !old.CreatedAt.Equal(w.CreatedAt)) {
			return false
		}
		identities[w.ID] = *w
		_, err := domain.ParseEffortDate(string(w.WorkDate))
		return err == nil && w.ID != uuid.Nil && users[w.OwnerUserID.String()] && users[w.CreatedByUserID.String()] && w.DurationMinutes >= 1 && w.DurationMinutes <= 1440 && (w.Description == nil || utf8.RuneCountInString(*w.Description) <= 2000) && !w.CreatedAt.IsZero() && !w.UpdatedAt.IsZero()
	}
	worklogIssues := map[string]string{}
	for _, row := range data.Tables["effort_worklogs"] {
		raw, _ := json.Marshal(row)
		var w domain.EffortWorklogSnapshot
		if err := json.Unmarshal(raw, &w); err != nil || !validWorklog(&w) {
			return bad("invalid worklog")
		}
		if worklogIssues[w.ID.String()] != "" {
			return bad("duplicate worklog")
		}
		worklogIssues[w.ID.String()] = transferString(row["issue_id"])
	}
	for table, rows := range data.Tables {
		if isEffortTransferTable(table) {
			for _, row := range rows {
				for _, field := range []string{"id", "workspace_id", "issue_id", "owner_user_id", "created_by_user_id", "actor_user_id"} {
					if row[field] == nil {
						continue
					}
					id := transferString(row[field])
					parsed, err := uuid.Parse(id)
					if err != nil || parsed == uuid.Nil || parsed.String() != id {
						return bad("invalid UUID in " + table + "." + field)
					}
				}
				if transferString(row["workspace_id"]) != parsed.manifest.SourceWorkspaceID {
					return bad("cross-workspace row")
				}
				if table != "workspace_effort_settings" && !issues[transferString(row["issue_id"])] {
					return bad("missing issue")
				}
			}
		}
	}
	for _, row := range data.Tables["effort_events"] {
		issue := transferString(row["issue_id"])
		kind := transferString(row["event_type"])
		cause := transferString(row["cause"])
		if cause != "manual" && cause != "worklog" && cause != "closure" && cause != "reactivation" {
			return bad("invalid event cause")
		}
		if !effortArchiveInteger(row["snapshot_version"], 1, 1) || !effortArchiveInteger(row["revision"], 1, revisions[issue]) {
			return bad("invalid event revision or snapshot version")
		}
		switch row["actor_kind"] {
		case "user":
			if !users[transferString(row["actor_user_id"])] || row["system_source"] != nil {
				return bad("invalid event actor")
			}
		case "system":
			if row["actor_user_id"] != nil || strings.TrimSpace(transferString(row["system_source"])) == "" {
				return bad("invalid system actor")
			}
		default:
			return bad("invalid actor kind")
		}
		planning := kind == "estimate_set" || kind == "remaining_set"
		if !planning && kind != "worklog_created" && kind != "worklog_updated" && kind != "worklog_deleted" {
			return bad("invalid event type")
		}
		snapshots, err := effortArchiveSnapshots(row)
		if err != nil || len(snapshots) == 0 {
			return bad("invalid event snapshots")
		}
		if kind == "worklog_created" && (row["before_snapshot"] != nil || row["after_snapshot"] == nil) || kind == "worklog_deleted" && (row["after_snapshot"] != nil || row["before_snapshot"] == nil) || (planning || kind == "worklog_updated") && len(snapshots) != 2 {
			return bad("incomplete event snapshots")
		}
		var previous *domain.EffortWorklogSnapshot
		for _, snapshot := range snapshots {
			if planning {
				p := snapshot.Planning
				if p == nil || snapshot.Worklog != nil {
					return bad("invalid planning snapshot")
				}
				for _, n := range []*int64{p.EstimateMinutes, p.RemainingMinutes} {
					if n != nil && (*n < 0 || *n > domain.EffortMaxMinutes) {
						return bad("invalid planning snapshot minutes")
					}
				}
			} else {
				w := snapshot.Worklog
				if w == nil || snapshot.Planning != nil || !validWorklog(w) {
					return bad("invalid worklog snapshot")
				}
				id := w.ID.String()
				if prior := worklogIssues[id]; prior != "" && prior != issue {
					return bad("snapshot references another issue")
				}
				worklogIssues[id] = issue
				if previous != nil && (previous.ID != w.ID || previous.OwnerUserID != w.OwnerUserID || previous.CreatedByUserID != w.CreatedByUserID || !previous.CreatedAt.Equal(w.CreatedAt)) {
					return bad("snapshot changes immutable attribution")
				}
				previous = w
			}
		}
	}
	return validateEffortArchiveBudget(data, nil)
}

func validateEffortArchiveBudget(data dto.WorkspaceTransferData, mapping map[string]string) error {
	totals := map[string]int64{}
	for _, row := range data.Tables["effort_worklogs"] {
		owner := transferString(row["owner_user_id"])
		if mapping != nil {
			owner = mapping[owner]
		}
		key := owner + "/" + transferString(row["work_date"])
		totals[key] += transferInt64(row["duration_minutes"])
		if totals[key] > 1440 {
			return invalidArchive(fmt.Sprintf("Efforts: daily work exceeds 1440 minutes on %s", row["work_date"]), nil)
		}
	}
	return nil
}
