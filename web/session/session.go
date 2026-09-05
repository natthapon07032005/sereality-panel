package session

import (
	"encoding/gob"
	"net/http"

	"x-ui/database"
	"x-ui/database/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	loginUserKey = "LOGIN_USER"
	defaultPath  = "/"
)

func init() {
	gob.Register(model.User{})
}

func SetLoginUser(c *gin.Context, user *model.User) {
	if user == nil {
		return
	}
	// Never serialize the password hash or login secret into the client-side cookie.
	safeUser := *user
	safeUser.Password = ""
	safeUser.LoginSecret = ""
	s := sessions.Default(c)
	s.Set(loginUserKey, safeUser)
}

func SetMaxAge(c *gin.Context, maxAge int) {
	s := sessions.Default(c)
	s.Options(sessions.Options{
		Path:     defaultPath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func GetLoginUser(c *gin.Context) *model.User {
	s := sessions.Default(c)
	obj := s.Get(loginUserKey)
	if obj == nil {
		return nil
	}
	user, ok := obj.(model.User)
	if !ok {

		s.Delete(loginUserKey)
		return nil
	}
	return &user
}

func IsLogin(c *gin.Context) bool {
	user := GetLoginUser(c)
	if user == nil || !isActiveUserStatus(user.Status) {
		return false
	}

	// Re-read status so a suspension or expiry invalidates existing sessions.
	db := database.GetDB()
	if db == nil || user.Id <= 0 {
		return false
	}
	current := &model.User{}
	if err := db.Select("status").First(current, user.Id).Error; err != nil {
		return false
	}
	return isActiveUserStatus(current.Status)
}

func isActiveUserStatus(status string) bool {
	return status == "" || status == "active"
}

func ClearSession(c *gin.Context) {
	s := sessions.Default(c)
	s.Clear()
	s.Options(sessions.Options{
		Path:     defaultPath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}
