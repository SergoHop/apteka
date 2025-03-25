package models

import "gorm.io/gorm"

type Medicine struct {
    gorm.Model
    //ID    int    `json:"id" gorm:"primaryKey;autoIncrement"`
    Name        string  `gorm:"not null" json:"name"`
    Description string  `json:"description"`
    Price       float64 `gorm:"not null" json:"price"`
    Quantity    int     `gorm:"not null" json:"quantity"`
}