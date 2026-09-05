package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type APIV2Controller struct {
	BaseController
	packageService      service.PackageService
	userService         service.UserManagementService
	nodeService         service.NodeService
	subscriptionService service.SubscriptionService
}

func NewAPIV2Controller(g *gin.RouterGroup) *APIV2Controller {
	a := &APIV2Controller{}
	g.GET("/health", a.health)
	g.GET("/packages", a.packages)
	g.POST("/packages", a.createPackage)
	g.PUT("/packages/:id", a.updatePackage)
	g.DELETE("/packages/:id", a.deletePackage)
	g.GET("/users", a.users)
	g.POST("/users/:id/status", a.updateUserStatus)
	g.POST("/users/:id/package", a.assignUserPackage)
	g.GET("/nodes", a.nodes)
	g.POST("/nodes", a.createNode)
	g.PUT("/nodes/:id", a.updateNode)
	g.DELETE("/nodes/:id", a.deleteNode)
	g.GET("/subscriptions", a.subscriptions)
	g.POST("/subscriptions", a.createSubscription)
	g.POST("/subscriptions/:id/status", a.updateSubscriptionStatus)
	return a
}

func (a *APIV2Controller) packages(c *gin.Context) {
	packages, err := a.packageService.List()
	c.JSON(http.StatusOK, service.NewAPIV2Response(packages, err))
}

func (a *APIV2Controller) nodes(c *gin.Context) {
	nodes, err := a.nodeService.List()
	c.JSON(http.StatusOK, service.NewAPIV2Response(nodes, err))
}

func (a *APIV2Controller) users(c *gin.Context) {
	users, err := a.userService.List()
	c.JSON(http.StatusOK, service.NewAPIV2Response(users, err))
}

func (a *APIV2Controller) updateUserStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid user ID")))
		return
	}
	var request userStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, err))
		return
	}
	err = a.userService.UpdateStatus(id, request.Status)
	c.JSON(http.StatusOK, service.NewAPIV2Response(nil, err))
}

func (a *APIV2Controller) assignUserPackage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid user ID")))
		return
	}
	var request userPackageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, err))
		return
	}
	err = a.userService.AssignPackage(id, request.PackageID)
	c.JSON(http.StatusOK, service.NewAPIV2Response(nil, err))
}

func (a *APIV2Controller) createPackage(c *gin.Context) {
	var input service.PackageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, err))
		return
	}
	pkg, err := a.packageService.Create(input)
	c.JSON(http.StatusOK, service.NewAPIV2Response(pkg, err))
}

func (a *APIV2Controller) updatePackage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid package ID")))
		return
	}
	var input service.PackageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, err))
		return
	}
	err = a.packageService.Update(id, input)
	c.JSON(http.StatusOK, service.NewAPIV2Response(nil, err))
}

func (a *APIV2Controller) deletePackage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid package ID")))
		return
	}
	err = a.packageService.Delete(id)
	c.JSON(http.StatusOK, service.NewAPIV2Response(nil, err))
}

func (a *APIV2Controller) createNode(c *gin.Context) {
	var input service.NodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, err))
		return
	}
	node, err := a.nodeService.Create(input)
	c.JSON(http.StatusOK, service.NewAPIV2Response(node, err))
}

func (a *APIV2Controller) updateNode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid node ID")))
		return
	}
	var input service.NodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, err))
		return
	}
	err = a.nodeService.Update(id, input)
	c.JSON(http.StatusOK, service.NewAPIV2Response(nil, err))
}

func (a *APIV2Controller) deleteNode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid node ID")))
		return
	}
	err = a.nodeService.Delete(id)
	c.JSON(http.StatusOK, service.NewAPIV2Response(nil, err))
}

func (a *APIV2Controller) subscriptions(c *gin.Context) {
	subscriptions, err := a.subscriptionService.List()
	c.JSON(http.StatusOK, service.NewAPIV2Response(subscriptions, err))
}

type apiV2SubscriptionRequest struct {
	UserID       int    `json:"userId"`
	PackageID    int    `json:"packageId"`
	StartsAt     string `json:"startsAt"`
	DurationDays int    `json:"durationDays"`
}

func (a *APIV2Controller) createSubscription(c *gin.Context) {
	var request apiV2SubscriptionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, err))
		return
	}
	start := time.Now().UTC()
	if request.StartsAt != "" {
		parsed, err := time.Parse(time.RFC3339, request.StartsAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid subscription start")))
			return
		}
		start = parsed
	}
	subscription, err := a.subscriptionService.Create(request.UserID, request.PackageID, start, request.DurationDays)
	c.JSON(http.StatusOK, service.NewAPIV2Response(subscription, err))
}

func (a *APIV2Controller) updateSubscriptionStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid subscription ID")))
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, err))
		return
	}
	err = a.subscriptionService.UpdateStatus(id, request.Status)
	c.JSON(http.StatusOK, service.NewAPIV2Response(nil, err))
}

func (a *APIV2Controller) health(c *gin.Context) {
	c.JSON(http.StatusOK, service.NewAPIV2Response(map[string]string{"version": "v2", "status": "ok"}, nil))
}
