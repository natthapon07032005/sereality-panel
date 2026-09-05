package controller

import (
	"errors"
	"net/http"
	"strconv"

	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

// NodeSyncController exposes a plan-only node synchronization endpoint.
type NodeSyncController struct {
	BaseController
}

// NewNodeSyncController registers the plan endpoint on the supplied router group.
func NewNodeSyncController(g *gin.RouterGroup) *NodeSyncController {
	a := &NodeSyncController{}
	g.POST("/nodes/:nodeId/sync/plan", a.plan)
	return a
}

type nodeSyncPlanRequest struct {
	Local  service.NodeSyncSnapshot `json:"local"`
	Remote service.NodeSyncSnapshot `json:"remote"`
	DryRun bool                     `json:"dryRun"`
}

func (a *NodeSyncController) plan(c *gin.Context) {
	nodeID, err := strconv.Atoi(c.Param("nodeId"))
	if err != nil || nodeID <= 0 {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid node ID")))
		return
	}

	var request nodeSyncPlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid node sync plan")))
		return
	}
	if !request.DryRun {
		c.JSON(http.StatusNotImplemented, service.NewAPIV2Response(nil, service.ErrNodeSyncUnavailable))
		return
	}

	plan, err := service.BuildNodeSyncPlan(service.NodeSyncPlanInput{
		NodeID: nodeID,
		Local:  request.Local,
		Remote: request.Remote,
		DryRun: request.DryRun,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid node sync plan")))
		return
	}

	c.JSON(http.StatusOK, service.NewAPIV2Response(plan, nil))
}
