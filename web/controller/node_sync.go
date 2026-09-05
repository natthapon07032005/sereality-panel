package controller

import (
	"errors"
	"net/http"
	"strconv"

	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

// NodeSyncController exposes dry-run and authenticated remote-backed sync planning.
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
	Local    service.NodeSyncSnapshot `json:"local"`
	Remote   service.NodeSyncSnapshot `json:"remote"`
	DryRun   *bool                    `json:"dryRun"`
	APIToken string                   `json:"apiToken"`
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
	if request.DryRun == nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("dryRun must be specified")))
		return
	}

	if !*request.DryRun {
		if !request.Local.Complete {
			c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid node sync plan")))
			return
		}
		remote, err := service.FetchNodeSyncSnapshot(nodeID, request.APIToken)
		if err != nil {
			status := http.StatusBadGateway
			switch {
			case errors.Is(err, service.ErrNodeSyncNodeNotFound), errors.Is(err, service.ErrNodeSyncNodeDisabled), errors.Is(err, service.ErrNodeSyncNodeURLInvalid), errors.Is(err, service.ErrNodeSyncTokenInvalid):
				status = http.StatusBadRequest
			}
			c.JSON(status, service.NewAPIV2Response(nil, err))
			return
		}
		plan, err := service.BuildNodeSyncPlan(service.NodeSyncPlanInput{NodeID: nodeID, Local: request.Local, Remote: remote, DryRun: false})
		if err != nil {
			c.JSON(http.StatusBadGateway, service.NewAPIV2Response(nil, errors.New("invalid node sync plan")))
			return
		}
		plan.RemoteSnapshotFetched = true
		c.JSON(http.StatusOK, service.NewAPIV2Response(plan, nil))
		return
	}

	plan, err := service.BuildNodeSyncPlan(service.NodeSyncPlanInput{
		NodeID: nodeID,
		Local:  request.Local,
		Remote: request.Remote,
		DryRun: *request.DryRun,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, service.NewAPIV2Response(nil, errors.New("invalid node sync plan")))
		return
	}

	c.JSON(http.StatusOK, service.NewAPIV2Response(plan, nil))
}
