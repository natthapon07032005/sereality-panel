package service

import (
	"errors"
	"time"

	"x-ui/database"
	"x-ui/database/model"

	"gorm.io/gorm"
)

func ShouldExpireSubscription(subscription model.Subscription, now time.Time) bool {
	return subscription.Status == SubscriptionStatusActive && !subscription.EndsAt.IsZero() && !now.Before(subscription.EndsAt)
}

func RefreshSubscriptionStatus(subscription model.Subscription, now time.Time) model.Subscription {
	if subscription.Status == SubscriptionStatusActive && !subscription.EndsAt.IsZero() && !now.Before(subscription.EndsAt) {
		subscription.Status = SubscriptionStatusExpired
	}
	return subscription
}

type SubscriptionEvent struct {
	ID                int       `json:"id"`
	UserID            int       `json:"userId"`
	PackageID         int       `json:"packageId"`
	Status            string    `json:"status"`
	StartsAt          time.Time `json:"startsAt"`
	EndsAt            time.Time `json:"endsAt"`
	ExternalReference string    `json:"-"`
}

func ToSubscriptionEvent(subscription model.Subscription) SubscriptionEvent {
	return SubscriptionEvent{ID: subscription.ID, UserID: subscription.UserID, PackageID: subscription.PackageID, Status: subscription.Status, StartsAt: subscription.StartsAt, EndsAt: subscription.EndsAt}
}

func (s *SubscriptionService) ExpireDue(now time.Time) error {
	return expireDue(database.GetDB(), now)
}

func expireDue(db *gorm.DB, now time.Time) error {
	if db == nil {
		return errors.New("database unavailable")
	}
	return db.Model(&model.Subscription{}).
		Where("status = ? AND ends_at IS NOT NULL AND ends_at != ? AND ends_at <= ?", SubscriptionStatusActive, time.Time{}, now).
		Update("status", SubscriptionStatusExpired).Error
}
