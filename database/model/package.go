package model

import "time"

// Package describes a reusable access and traffic entitlement.
// A zero traffic or device limit means unlimited.
type Package struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	DurationDays int       `json:"durationDays"`
	TrafficLimit int64     `json:"trafficLimit"`
	DeviceLimit  int       `json:"deviceLimit"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
