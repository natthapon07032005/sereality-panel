package service

import (
	"testing"
	"time"

	"x-ui/database/model"
)

func TestSubscriptionTransitionRejectsInvalidLifecycleMoves(t *testing.T) {
	invalid := [][2]string{
		{"unknown", "unknown"},
		{SubscriptionStatusExpired, SubscriptionStatusActive},
		{SubscriptionStatusCancelled, SubscriptionStatusActive},
		{SubscriptionStatusActive, SubscriptionStatusPending},
	}
	for _, transition := range invalid {
		if err := ValidateSubscriptionTransition(transition[0], transition[1]); err == nil {
			t.Fatalf("transition %q -> %q was accepted", transition[0], transition[1])
		}
	}
}

func TestCalculateSubscriptionEndUsesPackageDuration(t *testing.T) {
	start := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	got, err := CalculateSubscriptionEnd(start, 30)
	if err != nil {
		t.Fatalf("CalculateSubscriptionEnd() error = %v", err)
	}
	want := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("subscription end = %s, want %s", got, want)
	}
}

func TestCalculateRenewalWindowExtendsAnActiveSubscription(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	existingEnd := now.Add(10 * 24 * time.Hour)
	start, end, err := CalculateRenewalWindow(model.Subscription{Status: SubscriptionStatusActive, EndsAt: existingEnd}, now, 30)
	if err != nil {
		t.Fatalf("CalculateRenewalWindow() error = %v", err)
	}
	if !start.Equal(existingEnd) || !end.Equal(existingEnd.AddDate(0, 0, 30)) {
		t.Fatalf("renewal window = %s - %s, want %s - %s", start, end, existingEnd, existingEnd.AddDate(0, 0, 30))
	}
}

func TestApplySubscriptionStatusUsesLifecycleRules(t *testing.T) {
	subscription, err := ApplySubscriptionStatus(model.Subscription{Status: SubscriptionStatusPending}, SubscriptionStatusActive)
	if err != nil {
		t.Fatalf("ApplySubscriptionStatus() error = %v", err)
	}
	if subscription.Status != SubscriptionStatusActive {
		t.Fatalf("status = %q, want active", subscription.Status)
	}
	if _, err := ApplySubscriptionStatus(subscription, SubscriptionStatusPending); err == nil {
		t.Fatal("active subscription was allowed to move back to pending")
	}
}

func TestValidateSubscriptionPackageDurationRejectsClientOverride(t *testing.T) {
	if err := ValidateSubscriptionPackageDuration(30, 30); err != nil {
		t.Fatalf("matching package duration rejected: %v", err)
	}
	if err := ValidateSubscriptionPackageDuration(7, 30); err == nil {
		t.Fatal("client duration override was accepted")
	}
}
