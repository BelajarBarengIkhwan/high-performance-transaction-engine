package main

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrLockAccount         = fmt.Errorf("failed to lock account")
	ErrAccountNotFound     = fmt.Errorf("account not found")
	ErrInsufficientBalance = fmt.Errorf("insufficient balance")
	ErrFailedUpdateBalance = fmt.Errorf("failed to update balance")
)

type Service struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewService(db *gorm.DB, logger *logrus.Logger) *Service {
	return &Service{db: db, logger: logger}
}

func (s *Service) GetBalance(acc string) (decimal.Decimal, error) {
	account := Account{Acc: acc}
	if err := s.db.First(&account).Error; err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err}).Error(ErrAccountNotFound.Error())
		return decimal.Zero, ErrAccountNotFound
	}
	return account.Balance, nil
}

func (s *Service) Deposit(acc string, amount decimal.Decimal) error {
	account := Account{Acc: acc}
	tx := s.db.Begin()
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account).Error; err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "amount": amount.String()}).Error(ErrLockAccount.Error())
		tx.Rollback()
		return ErrAccountNotFound
	}

	account.Balance = account.Balance.Add(amount)
	if err := tx.Model(&account).Update("balance", account.Balance).Error; err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "balance": account.Balance.String(), "amount": amount.String()}).Error("failed to update balance")
		tx.Rollback()
		return ErrFailedUpdateBalance
	}
	tx.Commit()
	return nil
}

func (s *Service) Withdraw(acc string, amount decimal.Decimal) error {
	account := Account{Acc: acc}
	tx := s.db.Begin()
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account).Error; err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "amount": amount.String()}).Error(ErrLockAccount.Error())
		tx.Rollback()
		return ErrAccountNotFound
	}

	if account.Balance.LessThan(amount) {
		s.logger.WithFields(logrus.Fields{"acc": acc, "balance": account.Balance.String(), "amount": amount.String()}).Error(ErrInsufficientBalance.Error())
		tx.Rollback()
		return ErrInsufficientBalance
	}

	account.Balance = account.Balance.Sub(amount)
	if err := tx.Model(&account).Update("balance", account.Balance).Error; err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "balance": account.Balance.String(), "amount": amount.String()}).Error(ErrFailedUpdateBalance.Error())
		tx.Rollback()
		return ErrFailedUpdateBalance
	}
	tx.Commit()
	return nil
}
