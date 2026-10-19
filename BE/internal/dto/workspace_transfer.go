package dto

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"
)

const WorkspaceExportFormat = "kuayle.workspace"
const WorkspaceExportVersion = 2

type WorkspaceExportManifest struct {
	Format                  string         `json:"format"`
	Version                 int            `json:"version"`
	ExportedAt              time.Time      `json:"exported_at"`
	SourceSchemaVersion     uint           `json:"source_schema_version"`
	SourceWorkspaceID       string         `json:"source_workspace_id"`
	SourceWorkspaceName     string         `json:"source_workspace_name"`
	SourceWorkspaceSlug     string         `json:"source_workspace_slug"`
	Counts                  map[string]int `json:"counts"`
	Omitted                 []string       `json:"omitted"`
	Warnings                []string       `json:"warnings"`
	RequiresReconfiguration []string       `json:"requires_reconfiguration"`
}

type WorkspaceTransferRow map[string]any

type WorkspaceTransferData struct {
	UnavailableUsers []WorkspaceTransferRow            `json:"unavailable_users,omitempty"`
	Workspace        WorkspaceTransferRow              `json:"workspace"`
	Users            []WorkspaceTransferRow            `json:"users"`
	Tables           map[string][]WorkspaceTransferRow `json:"tables"`
}

type WorkspaceImportPreview struct {
	Manifest     WorkspaceExportManifest `json:"manifest"`
	Name         string                  `json:"name"`
	Slug         string                  `json:"slug"`
	MissingUsers []string                `json:"missing_users"`
}

type WorkspaceImportResult struct {
	ID                      string         `json:"id"`
	Name                    string         `json:"name"`
	Slug                    string         `json:"slug"`
	Counts                  map[string]int `json:"counts"`
	Warnings                []string       `json:"warnings"`
	RequiresReconfiguration []string       `json:"requires_reconfiguration"`
}

type WorkspaceTransferArchive struct {
	Manifest WorkspaceExportManifest
	Data     WorkspaceTransferData
	Assets   map[string][]byte
}

func DecodeTransferJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("workspace transfer JSON contains multiple values")
		}
		return err
	}
	return nil
}
