package controller

import (
	"strconv"

	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type NodeController struct {
	BaseController
	nodeService service.NodeService
}

func NewNodeController(g *gin.RouterGroup) *NodeController {
	a := &NodeController{}
	g.GET("/nodes", a.list)
	g.POST("/nodes", a.create)
	g.PUT("/nodes/:id", a.update)
	g.DELETE("/nodes/:id", a.delete)
	return a
}

func (a *NodeController) list(c *gin.Context) {
	nodes, err := a.nodeService.List()
	jsonObj(c, nodes, err)
}

func (a *NodeController) create(c *gin.Context) {
	var input service.NodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		jsonMsg(c, "invalid node", err)
		return
	}
	node, err := a.nodeService.Create(input)
	jsonObj(c, node, err)
}

func (a *NodeController) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid node id", err)
		return
	}
	var input service.NodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		jsonMsg(c, "invalid node", err)
		return
	}
	jsonMsg(c, "node updated", a.nodeService.Update(id, input))
}

func (a *NodeController) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid node id", err)
		return
	}
	jsonMsg(c, "node deleted", a.nodeService.Delete(id))
}
