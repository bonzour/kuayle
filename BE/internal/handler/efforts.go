package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/kuayle/kuayle-backend/internal/domain"
	"github.com/kuayle/kuayle-backend/internal/dto"
	"github.com/kuayle/kuayle-backend/internal/service"
	"github.com/kuayle/kuayle-backend/pkg/response"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

type EffortHandler struct{ svc *service.EffortService }

func NewEffortHandler(svc *service.EffortService) *EffortHandler { return &EffortHandler{svc: svc} }

func (h *EffortHandler) Settings(c echo.Context) error {
	ws := c.Get("workspace").(*domain.Workspace)
	result, err := h.svc.Settings(c.Request().Context(), ws.ID)
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

func effortResponse(c echo.Context, result any, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return response.NotFound(c, "Efforts")
	}
	if err != nil {
		log.WithError(err).Error("Efforts read failed")
		return response.InternalError(c)
	}
	return response.Success(c, http.StatusOK, result)
}
