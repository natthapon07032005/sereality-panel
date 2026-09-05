package service

import (
	"errors"
	"time"

	"x-ui/database"
	"x-ui/database/model"
)

const (
	SubscriptionStatusPending   = "pending"
	SubscriptionStatusActive    = "active"
	SubscriptionStatusExpired   = "expired"
	SubscriptionStatusCancelled = "cancelled"
)

func ValidateSubscriptionTransition(from, to string) error {
	if !isSubscriptionStatus(from) || !isSubscriptionStatus(to) {
		return errors.New("unknown subscription status")
	}
	if from == to {
		return nil
	}
	switch from {
	case SubscriptionStatusPending:
		if to == SubscriptionStatusActive || to == SubscriptionStatusCancelled {
			return nil
		}
	case SubscriptionStatusActive:
		if to == SubscriptionStatusExpired || to == SubscriptionStatusCancelled {
			return nil
		}
	}
	return errors.New("invalid subscription status transition")
}

func isSubscriptionStatus(status string) bool {
	switch status {
	case SubscriptionStatusPending, SubscriptionStatusActive, SubscriptionStatusExpired, SubscriptionStatusCancelled:
		return true
	default:
		return false
	}
}

func CalculateSubscriptionEnd(start time.Time, durationDays int) (time.Time, error) {
	if durationDays <= 0 {
		return time.Time{}, errors.New("duration days must be greater than zero")
	}
	return start.AddDate(0, 0, durationDays), nil
}

func ValidateSubscriptionPackageDuration(requestedDays, packageDays int) error {
	if requestedDays <= 0 || packageDays <= 0 || requestedDays != packageDays {
		return errors.New("subscription duration must match the package")
	}
	return nil
}

// CalculateRenewalWindow starts a renewal now unless an active subscription
// still has time remaining, in which case it appends the new entitlement to
// the existing end date. It only calculates a window; it does not activate or
// persist a subscription.
func CalculateRenewalWindow(subscription model.Subscription, now time.Time, durationDays int) (time.Time, time.Time, error) {
	start := now
	if subscription.Status == SubscriptionStatusActive && subscription.EndsAt.After(now) {
		start = subscription.EndsAt
	}
	end, err := CalculateSubscriptionEnd(start, durationDays)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, end, nil
}

// ApplySubscriptionStatus validates and applies a lifecycle transition to an
// in-memory value. Persistence is deliberately left to the caller.
func ApplySubscriptionStatus(subscription model.Subscription, status string) (model.Subscription, error) {
	if err := ValidateSubscriptionTransition(subscription.Status, status); err != nil {
		return subscription, err
	}
	subscription.Status = status
	return subscription, nil
}

type SubscriptionService struct{}

func (s *SubscriptionService) Create(userID, packageID int, start time.Time, durationDays int) (*model.Subscription, error) {
	if userID <= 0 || packageID <= 0 {
		return nil, errors.New("user and package are required")
	}
	var user model.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}
	var pkg model.Package
	if err := database.GetDB().First(&pkg, packageID).Error; err != nil {
		return nil, errors.New("package not found")
	}
	if !pkg.Enabled {
		return nil, errors.New("package is disabled")
	}
	if err := ValidateSubscriptionPackageDuration(durationDays, pkg.DurationDays); err != nil {
		return nil, err
	}
	end, err := CalculateSubscriptionEnd(start, pkg.DurationDays)
	if err != nil {
		return nil, err
	}
	subscription := &model.Subscription{UserID: userID, PackageID: packageID, Status: SubscriptionStatusPending, StartsAt: start, EndsAt: end}
	return subscription, database.GetDB().Create(subscription).Error
}

func (s *SubscriptionService) List() ([]SubscriptionEvent, error) {
	var subscriptions []model.Subscription
	if err := database.GetDB().Order("id desc").Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	events := make([]SubscriptionEvent, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		events = append(events, ToSubscriptionEvent(RefreshSubscriptionStatus(subscription, time.Now().UTC())))
	}
	return events, nil
}

func (s *SubscriptionService) UpdateStatus(id int, status string) error {
	if id <= 0 {
		return errors.New("subscription ID is required")
	}
	var subscription model.Subscription
	if err := database.GetDB().First(&subscription, id).Error; err != nil {
		return err
	}
	updated, err := ApplySubscriptionStatus(subscription, status)
	if err != nil {
		return err
	}
	result := database.GetDB().Model(&model.Subscription{}).
		Where("id = ? AND status = ?", id, subscription.Status).
		Update("status", updated.Status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("subscription changed concurrently")
	}
	return nil
}
