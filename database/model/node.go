package model

import "time"

type Node struct {
	ID               int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name             string    `json:"name"`
	BaseURL          string    `json:"baseUrl"`
	APITokenHash     string    `json:"-"`
	Enabled          bool      `json:"enabled"`
	LastHealthStatus string    `json:"lastHealthStatus"`
	LastHealthAt     time.Time `json:"lastHealthAt"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
