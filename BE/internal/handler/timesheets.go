package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/service"
	"github.com/kuayle/kuayle-backend/pkg/response"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

type TimesheetHandler struct{ svc *service.TimesheetService }

func NewTimesheetHandler(svc *service.TimesheetService) *TimesheetHandler {
	return &TimesheetHandler{svc: svc}
}

func (h *TimesheetHandler) Issues(c echo.Context) error {
	params, err := dto.ParseTimesheetIssues(c.QueryParams())
	if err != nil {
		return timesheetResponse(c, nil, err)
	}
	result, err := h.svc.Issues(c.Request().Context(), params)
	return timesheetResponse(c, result, err)
}

func (h *TimesheetHandler) Options(c echo.Context) error {
	params, err := dto.ParseTimesheetOptions(c.QueryParams())
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "Invalid Timesheet parameters")
	}
	result, err := h.svc.Options(c.Request().Context(), params)
	return timesheetResponse(c, result, err)
}

func (h *TimesheetHandler) Grid(c echo.Context) error {
	params, err := dto.ParseTimesheetRead(c.QueryParams(), false)
	if err != nil {
		return timesheetResponse(c, nil, err)
	}
	result, err := h.svc.Grid(c.Request().Context(), params)
	return timesheetResponse(c, result, err)
}

func (h *TimesheetHandler) Worklogs(c echo.Context) error {
	params, err := dto.ParseTimesheetRead(c.QueryParams(), true)
	if err != nil {
		return timesheetResponse(c, nil, err)
	}
	result, err := h.svc.Worklogs(c.Request().Context(), params)
	return timesheetResponse(c, result, err)
}

func timesheetResponse(c echo.Context, result any, err error) error {
	switch {
	case errors.Is(err, domain.ErrEffortForbidden):
		return response.Forbidden(c)
	case errors.Is(err, sql.ErrNoRows):
		return response.NotFound(c, "Timesheet selection")
	case errors.Is(err, dto.ErrTimesheetParams):
		return response.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "Invalid Timesheet parameters")
	case err != nil:
		log.WithError(err).Error("failed to load Timesheet")
		return response.InternalError(c)
	default:
		return response.Success(c, http.StatusOK, result)
	}
}
