package controller

import (
	"errors"
	"strconv"
	"time"

	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type SubscriptionController struct {
	BaseController
	subscriptionService service.SubscriptionService
}

func NewSubscriptionController(g *gin.RouterGroup) *SubscriptionController {
	a := &SubscriptionController{}
	g.GET("/subscriptions", a.list)
	g.POST("/subscriptions", a.create)
	g.POST("/subscriptions/:id/status", a.updateStatus)
	return a
}

type subscriptionCreateRequest struct {
	UserID       int    `json:"userId"`
	PackageID    int    `json:"packageId"`
	StartsAt     string `json:"startsAt"`
	DurationDays int    `json:"durationDays"`
}

type subscriptionStatusRequest struct {
	Status string `json:"status"`
}

func (a *SubscriptionController) list(c *gin.Context) {
	subscriptions, err := a.subscriptionService.List()
	jsonObj(c, subscriptions, err)
}

func (a *SubscriptionController) create(c *gin.Context) {
	var request subscriptionCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		jsonMsg(c, "invalid subscription", err)
		return
	}
	start := time.Now().UTC()
	if request.StartsAt != "" {
		parsed, err := time.Parse(time.RFC3339, request.StartsAt)
		if err != nil {
			jsonMsg(c, "invalid subscription start", err)
			return
		}
		start = parsed
	}
	if request.UserID <= 0 || request.PackageID <= 0 || request.DurationDays <= 0 {
		jsonMsg(c, "invalid subscription", errors.New("user, package and duration are required"))
		return
	}
	subscription, err := a.subscriptionService.Create(request.UserID, request.PackageID, start, request.DurationDays)
	jsonObj(c, subscription, err)
}

func (a *SubscriptionController) updateStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid subscription id", err)
		return
	}
	var request subscriptionStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		jsonMsg(c, "invalid subscription status", err)
		return
	}
	if err := a.subscriptionService.UpdateStatus(id, request.Status); err != nil {
		jsonMsg(c, "subscription status not updated", err)
		return
	}
	jsonMsg(c, "subscription status updated", nil)
}
