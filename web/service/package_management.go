package service

import (
	"errors"
	"strings"

	"x-ui/database"
	"x-ui/database/model"
)

type PackageInput struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	DurationDays int    `json:"durationDays"`
	TrafficLimit int64  `json:"trafficLimit"`
	DeviceLimit  int    `json:"deviceLimit"`
	Enabled      bool   `json:"enabled"`
}

func ValidatePackageInput(input PackageInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("package name is required")
	}
	if input.DurationDays <= 0 {
		return errors.New("duration days must be greater than zero")
	}
	if input.TrafficLimit < 0 || input.DeviceLimit < 0 {
		return errors.New("limits cannot be negative")
	}
	return nil
}

func PackageDeletionAllowed(userReferences, subscriptionReferences int64) bool {
	return userReferences == 0 && subscriptionReferences == 0
}

type PackageService struct{}

func (s *PackageService) List() ([]model.Package, error) {
	var packages []model.Package
	if err := database.GetDB().Order("id desc").Find(&packages).Error; err != nil {
		return nil, err
	}
	return packages, nil
}

func (s *PackageService) Create(input PackageInput) (*model.Package, error) {
	if err := ValidatePackageInput(input); err != nil {
		return nil, err
	}
	pkg := &model.Package{Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description), DurationDays: input.DurationDays, TrafficLimit: input.TrafficLimit, DeviceLimit: input.DeviceLimit, Enabled: input.Enabled}
	return pkg, database.GetDB().Create(pkg).Error
}

func (s *PackageService) Update(id int, input PackageInput) error {
	if id <= 0 {
		return errors.New("package ID is required")
	}
	if err := ValidatePackageInput(input); err != nil {
		return err
	}
	result := database.GetDB().Model(&model.Package{}).Where("id = ?", id).Updates(map[string]interface{}{"name": strings.TrimSpace(input.Name), "description": strings.TrimSpace(input.Description), "duration_days": input.DurationDays, "traffic_limit": input.TrafficLimit, "device_limit": input.DeviceLimit, "enabled": input.Enabled})
	if result.Error != nil {
		return result.Error
	}
	return requireRowsAffected("package", result.RowsAffected)
}

func (s *PackageService) Delete(id int) error {
	if id <= 0 {
		return errors.New("package ID is required")
	}
	// Keep the reference check and delete in one conditional write so a
	// concurrent assignment/subscription insert cannot pass a stale pre-check.
	result := database.GetDB().Where("id = ? AND NOT EXISTS (SELECT 1 FROM users WHERE users.package_id = packages.id) AND NOT EXISTS (SELECT 1 FROM subscriptions WHERE subscriptions.package_id = packages.id)", id).Delete(&model.Package{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var exists int64
		if err := database.GetDB().Model(&model.Package{}).Where("id = ?", id).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return errors.New("package not found")
		}
		return errors.New("package is referenced by users or subscriptions")
	}
	return requireRowsAffected("package", result.RowsAffected)
}
