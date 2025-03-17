package main

import "github.com/shopspring/decimal"

type Account struct {
	Acc     string `gorm:"primaryKey"`
	Balance decimal.Decimal
}
