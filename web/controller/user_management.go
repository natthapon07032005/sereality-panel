package controller

import (
	"strconv"

	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type UserManagementController struct {
	BaseController
	userService service.UserManagementService
}

func NewUserManagementController(g *gin.RouterGroup) *UserManagementController {
	a := &UserManagementController{}
	g.GET("/users", a.list)
	g.POST("/users/:id/status", a.updateStatus)
	g.POST("/users/:id/package", a.assignPackage)
	return a
}

type userStatusRequest struct { Status string `json:"status"` }
type userPackageRequest struct { PackageID int `json:"packageId"` }

func (a *UserManagementController) updateStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil { jsonMsg(c, "invalid user id", err); return }
	var request userStatusRequest
	if err = c.ShouldBindJSON(&request); err != nil { jsonMsg(c, "invalid user status", err); return }
	err = a.userService.UpdateStatus(id, request.Status)
	jsonMsg(c, "user status updated", err)
}

func (a *UserManagementController) assignPackage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil { jsonMsg(c, "invalid user id", err); return }
	var request userPackageRequest
	if err = c.ShouldBindJSON(&request); err != nil { jsonMsg(c, "invalid package assignment", err); return }
	err = a.userService.AssignPackage(id, request.PackageID)
	jsonMsg(c, "user package updated", err)
}

func (a *UserManagementController) list(c *gin.Context) {
	users, err := a.userService.List()
	jsonObj(c, users, err)
}
