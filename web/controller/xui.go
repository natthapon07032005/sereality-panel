package controller

import (
	"github.com/gin-gonic/gin"
)

type XUIController struct {
	BaseController

	inboundController        *InboundController
	settingController        *SettingController
	xraySettingController    *XraySettingController
	packageController        *PackageController
	userManagementController *UserManagementController
	nodeController           *NodeController
	nodeSyncController       *NodeSyncController
	subscriptionController   *SubscriptionController
	apiV2Controller          *APIV2Controller
}

func NewXUIController(g *gin.RouterGroup) *XUIController {
	a := &XUIController{}
	a.initRouter(g)
	return a
}

func (a *XUIController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/panel")
	g.Use(a.checkLogin)

	g.GET("/", a.index)
	g.GET("/inbounds", a.inbounds)
	g.GET("/settings", a.settings)
	g.GET("/xray", a.xraySettings)
	g.GET("/package-manager", a.packageManager)
	g.GET("/user-manager", a.userManager)
	g.GET("/subscription-manager", a.subscriptionManager)
	g.GET("/node-manager", a.nodeManager)

	a.inboundController = NewInboundController(g)
	a.settingController = NewSettingController(g)
	a.xraySettingController = NewXraySettingController(g)
	a.packageController = NewPackageController(g)
	a.userManagementController = NewUserManagementController(g)
	a.nodeController = NewNodeController(g)
	a.nodeSyncController = NewNodeSyncController(g.Group("/api/v2"))
	a.subscriptionController = NewSubscriptionController(g)
	a.apiV2Controller = NewAPIV2Controller(g.Group("/api/v2"))
}

func (a *XUIController) index(c *gin.Context) {
	html(c, "index.html", "pages.index.title", nil)
}

func (a *XUIController) inbounds(c *gin.Context) {
	html(c, "inbounds.html", "pages.inbounds.title", nil)
}

func (a *XUIController) settings(c *gin.Context) {
	html(c, "settings.html", "pages.settings.title", nil)
}

func (a *XUIController) xraySettings(c *gin.Context) {
	html(c, "xray.html", "pages.xray.title", nil)
}

func (a *XUIController) packageManager(c *gin.Context) {
	html(c, "packages.html", "Sereality Packages", nil)
}

func (a *XUIController) userManager(c *gin.Context) {
	html(c, "users.html", "Sereality Users", nil)
}

func (a *XUIController) subscriptionManager(c *gin.Context) {
	html(c, "subscriptions.html", "Sereality Subscriptions", nil)
}

func (a *XUIController) nodeManager(c *gin.Context) {
	html(c, "nodes.html", "Sereality Nodes", nil)
}
