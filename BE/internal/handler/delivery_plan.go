package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/domain"
	"github.com/metaforismo/sprintorio/BE/internal/service"
	"github.com/metaforismo/sprintorio/BE/pkg/response"
)

func (h *ProjectHandler) GetDeliveryPlan(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid project ID")
	}
	ws := c.Get("workspace").(*domain.Workspace)
	result, err := h.projectSvc.GetDeliveryPlan(c.Request().Context(), ws.ID, id)
	if err != nil {
		return deliveryPlanError(c, err)
	}
	return response.Success(c, http.StatusOK, result)
}
func (h *ProjectHandler) UpdateDeliveryPlan(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid project ID")
	}
	var req struct {
		Plan    *domain.DeliveryPlan `json:"plan"`
		Version *int                 `json:"version"`
	}
	// Bound even direct handler invocation; the largest valid plan is below 24 MiB.
	decoder := json.NewDecoder(http.MaxBytesReader(c.Response(), c.Request().Body, 24<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
	}
	if req.Plan == nil || req.Version == nil || *req.Version < 0 {
		return response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "plan and a nonnegative version are required")
	}
	if err := req.Plan.Validate(); err != nil {
		return response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
	ws := c.Get("workspace").(*domain.Workspace)
	result, err := h.projectSvc.UpdateDeliveryPlan(c.Request().Context(), ws.ID, id, *req.Plan, *req.Version)
	if err != nil {
		return deliveryPlanError(c, err)
	}
	return response.Success(c, http.StatusOK, result)
}
func deliveryPlanError(c echo.Context, err error) error {
	if errors.Is(err, service.ErrDeliveryPlanNotFound) {
		return response.NotFound(c, "Project")
	}
	if errors.Is(err, service.ErrDeliveryPlanConflict) {
		return response.Error(c, http.StatusConflict, "DELIVERY_PLAN_CONFLICT", err.Error())
	}
	return response.InternalError(c)
}
