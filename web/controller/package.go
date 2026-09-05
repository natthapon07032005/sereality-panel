package controller

import (
	"strconv"

	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type PackageController struct {
	BaseController
	packageService service.PackageService
}

func NewPackageController(g *gin.RouterGroup) *PackageController {
	a := &PackageController{}
	g.GET("/packages", a.list)
	g.POST("/packages", a.create)
	g.PUT("/packages/:id", a.update)
	g.DELETE("/packages/:id", a.delete)
	return a
}

func (a *PackageController) list(c *gin.Context) {
	packages, err := a.packageService.List()
	jsonObj(c, packages, err)
}

func (a *PackageController) create(c *gin.Context) {
	var input service.PackageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		jsonMsg(c, "invalid package", err)
		return
	}
	pkg, err := a.packageService.Create(input)
	jsonObj(c, pkg, err)
}

func (a *PackageController) update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid package id", err)
		return
	}
	var input service.PackageInput
	if err = c.ShouldBindJSON(&input); err != nil {
		jsonMsg(c, "invalid package", err)
		return
	}
	err = a.packageService.Update(id, input)
	jsonMsg(c, "package updated", err)
}

func (a *PackageController) delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		jsonMsg(c, "invalid package id", err)
		return
	}
	err = a.packageService.Delete(id)
	jsonMsg(c, "package deleted", err)
}
