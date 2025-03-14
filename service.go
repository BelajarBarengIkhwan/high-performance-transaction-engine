package main

import (
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewService(db *gorm.DB, logger *logrus.Logger) *Service {
	return &Service{db: db, logger: logger}
}

func (s *Service) GetBalance(acc string) (decimal.Decimal, error) {
	return decimal.NewFromInt(0), nil
}

func (s *Service) Deposit(acc string, amount decimal.Decimal) error {
	return nil
}

func (s *Service) Withdraw(acc string, amount decimal.Decimal) error {
	return nil
}

func (s *Service) Transfer(from string, to string, amount decimal.Decimal) error {
	return nil
}
