package controller

import (
	"net/http"

	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

// OperationsController provides route-free response writers for the safe
// operational DTOs. Route and middleware registration remain the caller's
// responsibility so these primitives do not change production behavior.
type OperationsController struct{}

func NewOperationsController() *OperationsController {
	return &OperationsController{}
}

func (a *OperationsController) WriteRateLimitDecision(c *gin.Context, decision service.RateLimitDecision) {
	c.JSON(http.StatusOK, decision)
}

func (a *OperationsController) WriteAuditEvent(c *gin.Context, event service.AuditEvent) {
	c.JSON(http.StatusOK, event)
}

func (a *OperationsController) WriteBackupStatus(c *gin.Context, status service.BackupStatus) {
	c.JSON(http.StatusOK, status)
}
