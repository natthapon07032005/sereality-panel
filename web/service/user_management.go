package service

import (
	"errors"

	"x-ui/database"
	"x-ui/database/model"
)

const (
	ManagedUserStatusActive    = "active"
	ManagedUserStatusSuspended = "suspended"
	ManagedUserStatusExpired   = "expired"
)

func ValidateManagedUserStatus(status string) error {
	switch status {
	case ManagedUserStatusActive, ManagedUserStatusSuspended, ManagedUserStatusExpired:
		return nil
	default:
		return errors.New("unknown user status")
	}
}

// ManagedUser is the safe administrative representation of a panel user.
type ManagedUser struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	PackageID   int    `json:"packageId"`
	Status      string `json:"status"`
	Password    string `json:"-"`
	LoginSecret string `json:"-"`
}

func ToManagedUser(user model.User) ManagedUser {
	status := user.Status
	if status == "" {
		status = ManagedUserStatusActive
	}
	return ManagedUser{ID: user.Id, Username: user.Username, PackageID: user.PackageID, Status: status}
}

type UserManagementService struct{}

func (s *UserManagementService) List() ([]ManagedUser, error) {
	var users []model.User
	if err := database.GetDB().Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	result := make([]ManagedUser, 0, len(users))
	for _, user := range users {
		result = append(result, ToManagedUser(user))
	}
	return result, nil
}

func (s *UserManagementService) UpdateStatus(id int, status string) error {
	if id <= 0 {
		return errors.New("user ID is required")
	}
	if err := ValidateManagedUserStatus(status); err != nil {
		return err
	}
	result := database.GetDB().Model(&model.User{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (s *UserManagementService) AssignPackage(id, packageID int) error {
	if id <= 0 || packageID < 0 {
		return errors.New("invalid user or package id")
	}
	if packageID > 0 {
		var pkg model.Package
		if err := database.GetDB().First(&pkg, packageID).Error; err != nil {
			return errors.New("package not found")
		}
	}
	query := database.GetDB().Model(&model.User{}).Where("id = ?", id)
	if packageID > 0 {
		// Validate the package as part of the write, so a concurrent package
		// deletion cannot turn a successful assignment into an orphan.
		query = query.Where("EXISTS (SELECT 1 FROM packages WHERE packages.id = ?)", packageID)
	}
	result := query.Update("package_id", packageID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var userExists int64
		if err := database.GetDB().Model(&model.User{}).Where("id = ?", id).Count(&userExists).Error; err != nil {
			return err
		}
		if userExists == 0 {
			return errors.New("user not found")
		}
		return errors.New("package not found")
	}
	return nil
}
