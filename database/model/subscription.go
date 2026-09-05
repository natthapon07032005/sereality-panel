package model

import "time"

type Subscription struct {
	ID                int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID            int       `json:"userId"`
	PackageID         int       `json:"packageId"`
	Status            string    `json:"status"`
	StartsAt          time.Time `json:"startsAt"`
	EndsAt            time.Time `json:"endsAt"`
	ExternalReference string    `json:"externalReference"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	User              *User     `json:"-" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Package           *Package  `json:"-" gorm:"foreignKey:PackageID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}
