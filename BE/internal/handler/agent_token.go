package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/metaforismo/sprintorio/BE/internal/middleware"
	"github.com/metaforismo/sprintorio/BE/internal/service"
	"github.com/metaforismo/sprintorio/BE/pkg/response"
)

type AgentTokenHandler struct{ svc *service.AgentTokenService }

func NewAgentTokenHandler(svc *service.AgentTokenService) *AgentTokenHandler {
	return &AgentTokenHandler{svc: svc}
}
func (h *AgentTokenHandler) Create(c echo.Context) error {
	var req service.CreateAgentTokenRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Response(), c.Request().Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return response.Error(c, 400, "BAD_REQUEST", "Invalid request body")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return response.Error(c, 400, "BAD_REQUEST", "Invalid request body")
	}
	result, err := h.svc.Create(c.Request().Context(), middleware.GetWorkspace(c).ID, middleware.GetUserID(c), req)
	if errors.Is(err, service.ErrInvalidAgentToken) {
		return response.Error(c, 400, "BAD_REQUEST", err.Error())
	}
	if err != nil {
		return response.InternalError(c)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return response.Success(c, 201, result)
}
func (h *AgentTokenHandler) List(c echo.Context) error {
	tokens, err := h.svc.List(c.Request().Context(), middleware.GetWorkspace(c).ID, middleware.GetUserID(c))
	if err != nil {
		return response.InternalError(c)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return response.Success(c, 200, tokens)
}
func (h *AgentTokenHandler) Revoke(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.Error(c, 400, "BAD_REQUEST", "Invalid token ID")
	}
	found, err := h.svc.Revoke(c.Request().Context(), id, middleware.GetWorkspace(c).ID, middleware.GetUserID(c))
	if err != nil {
		return response.InternalError(c)
	}
	if !found {
		return response.NotFound(c, "Agent token")
	}
	return response.Success(c, 200, map[string]string{"status": "revoked"})
}
