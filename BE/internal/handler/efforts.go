package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/middleware"
	"github.com/kuayle/kuayle-backend/internal/repository"
	"github.com/kuayle/kuayle-backend/internal/service"
	"github.com/kuayle/kuayle-backend/pkg/response"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

type EffortHandler struct {
	svc    *service.EffortService
	issues repository.IssueRepo
}

func NewEffortHandler(svc *service.EffortService, issues repository.IssueRepo) *EffortHandler {
	return &EffortHandler{svc: svc, issues: issues}
}

func decodeEffortCommand(c echo.Context, req any) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(req); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("expected a single JSON object")
	}
	return nil
}

func (h *EffortHandler) SetEstimate(c echo.Context) error {
	return h.setPlanning(c, true)
}

func (h *EffortHandler) SetRemaining(c echo.Context) error {
	return h.setPlanning(c, false)
}

func (h *EffortHandler) setPlanning(c echo.Context, estimate bool) error {
	if domain.IsToken(c.Request().Context()) || c.Get(middleware.TokenScopesKey) != nil {
		return response.Forbidden(c)
	}
	var estimateReq dto.SetEffortEstimateRequest
	var remainingReq dto.SetEffortRemainingRequest
	var req any = &remainingReq
	if estimate {
		req = &estimateReq
	}
	if err := decodeEffortCommand(c, req); err != nil {
		return response.ValidationError(c, []dto.ErrorDetail{{Field: "body", Message: "invalid Efforts JSON"}})
	}
	ws := c.Get("workspace").(*domain.Workspace)
	ctx := c.Request().Context()
	issue, err := h.issues.GetByIdentifier(ctx, ws.ID, c.Param("identifier"))
	if err != nil {
		return effortResponse(c, nil, err)
	}
	if issue == nil {
		return response.NotFound(c, "Efforts")
	}
	if estimate {
		result, err := h.svc.SetEstimate(ctx, ws.ID, issue.ID, estimateReq)
		return effortResponse(c, result, err)
	}
	result, err := h.svc.SetRemaining(ctx, ws.ID, issue.ID, remainingReq)
	return effortResponse(c, result, err)
}

func (h *EffortHandler) Settings(c echo.Context) error {
	ws := c.Get("workspace").(*domain.Workspace)
	result, err := h.svc.Settings(c.Request().Context(), ws.ID)
	return effortResponse(c, result, err)
}

func (h *EffortHandler) SetSettings(c echo.Context) error {
	role := middleware.GetWorkspaceRole(c)
	if domain.IsToken(c.Request().Context()) || c.Get(middleware.TokenScopesKey) != nil || role != domain.RoleOwner && role != domain.RoleAdmin {
		return response.Forbidden(c)
	}
	var req dto.SetEffortSettingsRequest
	if err := decodeEffortCommand(c, &req); err != nil {
		return response.ValidationError(c, []dto.ErrorDetail{{Field: "body", Message: "invalid settings JSON"}})
	}
	ws := c.Get("workspace").(*domain.Workspace)
	result, err := h.svc.SetSettings(c.Request().Context(), ws.ID, req)
	return effortResponse(c, result, err)
}

func (h *EffortHandler) Summary(c echo.Context) error {
	ws := c.Get("workspace").(*domain.Workspace)
	result, err := h.svc.Summary(c.Request().Context(), ws.ID, c.Param("identifier"))
	return effortResponse(c, result, err)
}

func (h *EffortHandler) Worklogs(c echo.Context) error {
	page := 1
	if values, ok := c.QueryParams()["page"]; ok {
		parsed, err := strconv.Atoi(values[0])
		if err != nil || parsed < 1 || int64(parsed) > domain.EffortMaxSafeInteger/domain.EffortPageSize {
			return response.ValidationError(c, []dto.ErrorDetail{{Field: "page", Message: "must be a positive page within the safe integer range"}})
		}
		page = parsed
	}
	if values, ok := c.QueryParams()["per_page"]; ok && (len(values) != 1 || values[0] != "50") {
		return response.ValidationError(c, []dto.ErrorDetail{{Field: "per_page", Message: "must be 50"}})
	}
	ws := c.Get("workspace").(*domain.Workspace)
	result, err := h.svc.Worklogs(c.Request().Context(), ws.ID, c.Param("identifier"), page)
	return effortResponse(c, result, err)
}

func (h *EffortHandler) WorklogOwners(c echo.Context) error {
	if domain.IsToken(c.Request().Context()) || c.Get(middleware.TokenScopesKey) != nil {
		return response.Forbidden(c)
	}
	ws := c.Get("workspace").(*domain.Workspace)
	result, err := h.svc.WorklogOwners(c.Request().Context(), ws.ID, c.Param("identifier"))
	return effortResponse(c, result, err)
}

func (h *EffortHandler) CreateWorklog(c echo.Context) error { return h.worklogCommand(c, "create") }
func (h *EffortHandler) UpdateWorklog(c echo.Context) error { return h.worklogCommand(c, "update") }
func (h *EffortHandler) DeleteWorklog(c echo.Context) error { return h.worklogCommand(c, "delete") }

func (h *EffortHandler) worklogCommand(c echo.Context, operation string) error {
	if domain.IsToken(c.Request().Context()) || c.Get(middleware.TokenScopesKey) != nil {
		return response.Forbidden(c)
	}
	var create dto.CreateEffortWorklogRequest
	var update dto.UpdateEffortWorklogRequest
	var remove dto.DeleteEffortWorklogRequest
	var req any = &create
	if operation == "update" {
		req = &update
	}
	if operation == "delete" {
		req = &remove
	}
	if err := decodeEffortCommand(c, req); err != nil {
		field := "body"
		var typeError *json.UnmarshalTypeError
		if errors.As(err, &typeError) && typeError.Field != "" {
			field = typeError.Field
		}
		if strings.Contains(err.Error(), "work_date") {
			field = "work_date"
		}
		return response.ValidationError(c, []dto.ErrorDetail{{Field: field, Message: "invalid Efforts JSON"}})
	}
	var worklog uuid.UUID
	if operation != "create" {
		var err error
		worklog, err = uuid.Parse(c.Param("worklogId"))
		if err != nil || worklog == uuid.Nil {
			return response.ValidationError(c, []dto.ErrorDetail{{Field: "worklog_id", Message: "invalid worklog UUID"}})
		}
	}
	ws := c.Get("workspace").(*domain.Workspace)
	ctx := c.Request().Context()
	issue, err := h.issues.GetByIdentifier(ctx, ws.ID, c.Param("identifier"))
	if err != nil {
		return effortResponse(c, nil, err)
	}
	if issue == nil {
		return response.NotFound(c, "Efforts")
	}
	var result *dto.EffortMutationReceipt
	switch operation {
	case "create":
		result, err = h.svc.CreateWorklog(ctx, ws.ID, issue.ID, create)
	case "update":
		result, err = h.svc.UpdateWorklog(ctx, ws.ID, issue.ID, worklog, update)
	case "delete":
		result, err = h.svc.DeleteWorklog(ctx, ws.ID, issue.ID, worklog, remove)
	}
	return effortResponse(c, result, err)
}

func effortResponse(c echo.Context, result any, err error) error {
	if errors.Is(err, domain.ErrEffortForbidden) {
		return response.Forbidden(c)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return response.NotFound(c, "Efforts")
	}
	var commandError *repository.EffortError
	if errors.As(err, &commandError) {
		if commandError.Code == dto.EffortDailyLimit {
			return response.Error(c, http.StatusBadRequest, commandError.Code, commandError.Message)
		}
		if commandError.Code == "VALIDATION_ERROR" {
			return response.ValidationError(c, []dto.ErrorDetail{{Field: commandError.Field, Message: commandError.Message}})
		}
		return response.Error(c, http.StatusConflict, commandError.Code, commandError.Message)
	}
	if err != nil {
		log.WithError(err).Error("Efforts operation failed")
		return response.InternalError(c)
	}
	return response.Success(c, http.StatusOK, result)
}
