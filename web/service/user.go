package service

import (
	"errors"
	"strings"

	"x-ui/database"
	"x-ui/database/model"
	"x-ui/logger"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct{}

// UserSecretView is the minimum user representation needed by the settings
// page. Password hashes are intentionally not part of this response.
type UserSecretView struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	LoginSecret string `json:"loginSecret"`
}

func ToUserSecretView(user model.User) UserSecretView {
	return UserSecretView{ID: user.Id, Username: user.Username, LoginSecret: user.LoginSecret}
}

func (s *UserService) GetFirstUser() (*model.User, error) {
	db := database.GetDB()

	user := &model.User{}
	err := db.Model(model.User{}).
		First(user).
		Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) CheckUser(username string, password string, secret string) *model.User {
	db := database.GetDB()
	if db == nil {
		return nil
	}

	user := &model.User{}
	err := db.Model(model.User{}).
		Where("username = ? and login_secret = ?", username, secret).
		First(user).
		Error
	if err == gorm.ErrRecordNotFound {
		return nil
	} else if err != nil {
		logger.Warning("check user err:", err)
		return nil
	}
	if user.Status == ManagedUserStatusSuspended || user.Status == ManagedUserStatusExpired {
		return nil
	}
	if !passwordMatchesStored(user.Password, password) {
		return nil
	}
	if isLegacyPassword(user.Password) {
		if hashed, hashErr := HashPassword(password); hashErr == nil {
			// Do not overwrite a concurrent password reset that changed the
			// legacy value after this read.
			_ = db.Model(model.User{}).Where("id = ? AND password = ?", user.Id, user.Password).Update("password", hashed).Error
		}
	}
	return user
}

func passwordMatchesStored(stored, password string) bool {
	if isLegacyPassword(stored) {
		return stored == password
	}
	return CheckPasswordHash(password, stored)
}

func isLegacyPassword(stored string) bool {
	return !strings.HasPrefix(stored, "$2a$") &&
		!strings.HasPrefix(stored, "$2b$") &&
		!strings.HasPrefix(stored, "$2x$") &&
		!strings.HasPrefix(stored, "$2y$")
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (s *UserService) UpdateUser(id int, username string, password string) error {
	db := database.GetDB()
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}
	return db.Model(model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"username": username, "password": hashedPassword}).
		Error
}

func (s *UserService) CheckUserPassword(id int, password string) bool {
	if id <= 0 {
		return false
	}
	if database.GetDB() == nil {
		return false
	}
	user := &model.User{}
	if err := database.GetDB().Model(model.User{}).Where("id = ?", id).First(user).Error; err != nil {
		return false
	}
	return passwordMatchesStored(user.Password, password)
}

func (s *UserService) UpdateUserSecret(id int, secret string) error {
	db := database.GetDB()
	return db.Model(model.User{}).
		Where("id = ?", id).
		Update("login_secret", secret).
		Error
}

func (s *UserService) RemoveUserSecret() error {
	db := database.GetDB()
	return db.Model(model.User{}).
		Where("1 = 1").
		Update("login_secret", "").
		Error
}

func (s *UserService) GetUserSecret(id int) *UserSecretView {
	db := database.GetDB()
	user := &model.User{}
	err := db.Model(model.User{}).
		Where("id = ?", id).
		First(user).
		Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	view := ToUserSecretView(*user)
	return &view
}

func (s *UserService) CheckSecretExistence() (bool, error) {
	db := database.GetDB()

	var count int64
	err := db.Model(model.User{}).
		Where("login_secret IS NOT NULL").
		Count(&count).
		Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *UserService) UpdateFirstUser(username string, password string) error {
	if username == "" {
		return errors.New("username can not be empty")
	} else if password == "" {
		return errors.New("password can not be empty")
	}
	db := database.GetDB()
	user := &model.User{}
	err := db.Model(model.User{}).First(user).Error
	if database.IsNotFound(err) {
		user.Username = username
		user.Password, err = HashPassword(password)
		if err != nil {
			return err
		}
		return db.Model(model.User{}).Create(user).Error
	} else if err != nil {
		return err
	}
	user.Username = username
	user.Password, err = HashPassword(password)
	if err != nil {
		return err
	}
	return db.Save(user).Error
}
