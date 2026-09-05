package controller

import (
	"net"
	"net/http"
	"strings"

	"x-ui/config"
	"x-ui/logger"
	"x-ui/web/entity"

	"github.com/gin-gonic/gin"
)

func getRemoteIp(c *gin.Context) string {
	addr := c.Request.RemoteAddr
	ip, _, _ := net.SplitHostPort(addr)
	if ip == "" {
		return addr
	}
	return ip
}

func jsonMsg(c *gin.Context, msg string, err error) {
	jsonMsgObj(c, msg, nil, err)
}

func jsonObj(c *gin.Context, obj interface{}, err error) {
	jsonMsgObj(c, "", obj, err)
}

func jsonMsgObj(c *gin.Context, msg string, obj interface{}, err error) {
	m := entity.Msg{
		Obj: obj,
	}
	if err == nil {
		m.Success = true
		if msg != "" {
			m.Msg = msg + " " + I18nWeb(c, "success")
		}
	} else {
		m.Success = false
		status := I18nWeb(c, "fail")
		if status == "" {
			status = "failed"
		}
		label := strings.TrimSpace(msg)
		if label == "" {
			label = "request"
		}
		m.Msg = label + " " + status
		logger.Warning(label+" "+status+": ", err)
	}
	c.JSON(http.StatusOK, m)
}

func pureJsonMsg(c *gin.Context, statusCode int, success bool, msg string) {
	c.JSON(statusCode, entity.Msg{
		Success: success,
		Msg:     msg,
	})
}

func html(c *gin.Context, name string, title string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}
	data["title"] = title
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.GetHeader("X-Real-IP")
	}
	if host == "" {
		var err error
		host, _, err = net.SplitHostPort(c.Request.Host)
		if err != nil {
			host = c.Request.Host
		}
	}
	data["host"] = host
	data["request_uri"] = c.Request.RequestURI
	data["base_path"] = c.GetString("base_path")
	c.HTML(http.StatusOK, name, getContext(data))
}

func getContext(h gin.H) gin.H {
	a := gin.H{
		"cur_ver": config.GetVersion(),
	}
	for key, value := range h {
		a[key] = value
	}
	return a
}

func isAjax(c *gin.Context) bool {
	return c.GetHeader("X-Requested-With") == "XMLHttpRequest"
}
