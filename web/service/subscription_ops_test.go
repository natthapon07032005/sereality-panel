package service

import (
	"errors"
	"testing"
	"time"

	"x-ui/database/model"
)

var errAPIV2Test = errors.New("test error")

func TestRefreshSubscriptionStatusExpiresActiveSubscription(t *testing.T) {
	subscription := model.Subscription{Status: SubscriptionStatusActive, EndsAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	got := RefreshSubscriptionStatus(subscription, time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC))
	if got.Status != SubscriptionStatusExpired {
		t.Fatalf("status = %q, want expired", got.Status)
	}
}

func TestShouldExpireSubscriptionOnlyWhenActiveAndDue(t *testing.T) {
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	if !ShouldExpireSubscription(model.Subscription{Status: SubscriptionStatusActive, EndsAt: now}, now) {
		t.Fatal("active subscription due at now was not marked due")
	}
	if ShouldExpireSubscription(model.Subscription{Status: SubscriptionStatusCancelled, EndsAt: now.Add(-time.Hour)}, now) {
		t.Fatal("cancelled subscription was marked due")
	}
}

func TestToSubscriptionEventOmitsExternalReference(t *testing.T) {
	event := ToSubscriptionEvent(model.Subscription{ID: 8, UserID: 2, PackageID: 3, Status: SubscriptionStatusActive, ExternalReference: "private-payment-reference"})
	if event.ID != 8 || event.Status != SubscriptionStatusActive || event.ExternalReference != "" {
		t.Fatalf("subscription event = %+v", event)
	}
}

func TestExpireDueWithUnavailableDatabaseReturnsError(t *testing.T) {
	if err := expireDue(nil, time.Now().UTC()); err == nil {
		t.Fatal("expireDue(nil) returned nil")
	}
}
