package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"x-ui/logger"
	"x-ui/web/service"
	"x-ui/web/session"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LoginForm struct {
	Username    string `json:"username" form:"username"`
	Password    string `json:"password" form:"password"`
	LoginSecret string `json:"loginSecret" form:"loginSecret"`
}

type IndexController struct {
	BaseController

	settingService service.SettingService
	userService    service.UserService
	tgbot          service.Tgbot
	loginLimiter   *service.RateLimiter
}

func NewIndexController(g *gin.RouterGroup) *IndexController {
	a := &IndexController{loginLimiter: service.NewRateLimiter(10, time.Minute, 10000)}
	a.initRouter(g)
	return a
}

func (a *IndexController) initRouter(g *gin.RouterGroup) {
	g.GET("/", a.index)
	g.POST("/login", a.login)
	g.GET("/logout", a.logout)
	g.POST("/getSecretStatus", a.getSecretStatus)
}

func (a *IndexController) index(c *gin.Context) {
	if session.IsLogin(c) {
		c.Redirect(http.StatusTemporaryRedirect, "panel/")
		return
	}
	html(c, "login.html", "pages.login.title", nil)
}

func (a *IndexController) login(c *gin.Context) {
	var form LoginForm

	if err := c.ShouldBind(&form); err != nil {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.invalidFormData"))
		return
	}
	if form.Username == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.emptyUsername"))
		return
	}
	if form.Password == "" {
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.emptyPassword"))
		return
	}

	if a.loginLimiter == nil {
		a.loginLimiter = service.NewRateLimiter(10, time.Minute, 10000)
	}
	if decision := a.loginLimiter.Allow(loginAttemptKey(getRemoteIp(c), form.Username)); !decision.Allowed {
		retryAfter := int((decision.RetryAfter + time.Second - 1) / time.Second)
		if retryAfter < 1 {
			retryAfter = 1
		}
		c.Header("Retry-After", strconv.Itoa(retryAfter))
		pureJsonMsg(c, http.StatusTooManyRequests, false, "Too many login attempts. Try again later.")
		return
	}

	user := a.userService.CheckUser(form.Username, form.Password, form.LoginSecret)
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	safeUser := template.HTMLEscapeString(form.Username)

	if user == nil {
		logger.Warningf("wrong username: \"%s\", IP: \"%s\"", safeUser, getRemoteIp(c))
		a.tgbot.UserLoginNotify(safeUser, "", getRemoteIp(c), timeStr, 0)
		pureJsonMsg(c, http.StatusOK, false, I18nWeb(c, "pages.login.toasts.wrongUsernameOrPassword"))
		return
	}

	logger.Infof("%s logged in successfully, Ip Address: %s\n", safeUser, getRemoteIp(c))
	a.tgbot.UserLoginNotify(safeUser, ``, getRemoteIp(c), timeStr, 1)

	sessionMaxAge, err := a.settingService.GetSessionMaxAge()
	if err != nil {
		logger.Warning("Unable to get session's max age from DB")
	}

	session.SetMaxAge(c, sessionMaxAge*60)
	session.SetLoginUser(c, user)
	if err := sessions.Default(c).Save(); err != nil {
		logger.Warning("Unable to save session: ", err)
		return
	}

	logger.Infof("%s logged in successfully", safeUser)
	jsonMsg(c, I18nWeb(c, "pages.login.toasts.successLogin"), nil)
}

func loginAttemptKey(remoteIP, username string) string {
	digest := sha256.Sum256([]byte(remoteIP + "\x00" + username))
	return hex.EncodeToString(digest[:])
}

func (a *IndexController) logout(c *gin.Context) {
	user := session.GetLoginUser(c)
	if user != nil {
		logger.Infof("%s logged out successfully", user.Username)
	}
	session.ClearSession(c)
	if err := sessions.Default(c).Save(); err != nil {
		logger.Warning("Unable to save session after clearing:", err)
	}
	c.Redirect(http.StatusTemporaryRedirect, c.GetString("base_path"))
}

func (a *IndexController) getSecretStatus(c *gin.Context) {
	status, err := a.settingService.GetSecretStatus()
	if err == nil {
		jsonObj(c, status, nil)
	}
}
