package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
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
	ErrInternalError       = fmt.Errorf("internal error")
	RedisAccountKey        = "account:%s"
)

type Service struct {
	db     *gorm.DB
	redis  *redis.Client
	logger *logrus.Logger
}

func NewService(db *gorm.DB, rdb *redis.Client, logger *logrus.Logger) *Service {
	return &Service{db: db, redis: rdb, logger: logger}
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
	var balanceBefore decimal.Decimal
	account := Account{Acc: acc}
	tx := s.db.Begin()
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account).Error; err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "amount": amount.String()}).Error(ErrLockAccount.Error())
		tx.Rollback()
		return ErrAccountNotFound
	}

	balanceBefore = account.Balance
	account.Balance = account.Balance.Add(amount)
	if err := tx.Model(&account).Update("balance", account.Balance).Error; err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "balance": account.Balance.String(), "amount": amount.String()}).Error("failed to update balance")
		tx.Rollback()
		return ErrFailedUpdateBalance
	}
	tx.Commit()
	s.logger.WithFields(logrus.Fields{"acc": acc, "balance_before": balanceBefore.String(), "balance_after": account.Balance.String(), "amount": amount.String()}).Info("deposit success")
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

	balanceBefore := account.Balance
	account.Balance = account.Balance.Sub(amount)
	if err := tx.Model(&account).Update("balance", account.Balance).Error; err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "balance": account.Balance.String(), "amount": amount.String()}).Error(ErrFailedUpdateBalance.Error())
		tx.Rollback()
		return ErrFailedUpdateBalance
	}

	s.logger.WithFields(logrus.Fields{"acc": acc, "balance_before": balanceBefore.String(), "balance_after": account.Balance.String(), "amount": amount.String()}).Info("withdraw success")
	tx.Commit()
	return nil
}

func (s *Service) FastWithdraw(acc string, amount decimal.Decimal) (err error) {
	var balanceBefore decimal.Decimal
	var balanceAfter decimal.Decimal
	ctx := context.Background()
	redisKey := fmt.Sprintf(RedisAccountKey, acc)
	maxRetries := 100
	txf := func(tx *redis.Tx) error {
		balanceStr, internalError := tx.Get(ctx, redisKey).Result()
		if internalError != nil {
			err = ErrLockAccount
			s.logger.WithFields(logrus.Fields{"acc": acc, "error": internalError, "redis_key": redisKey}).Error(err.Error())
			return err
		}

		balance, internalError := decimal.NewFromString(balanceStr)
		if internalError != nil {
			err = ErrInternalError
			s.logger.WithFields(logrus.Fields{"acc": acc, "error": internalError, "balance_string": balanceStr}).Error(err.Error())
			return err
		}
		balanceBefore = balance

		if balance.LessThan(amount) {
			err = ErrInsufficientBalance
			s.logger.WithFields(logrus.Fields{"acc": acc, "balance": balance.String(), "amount": amount.String()}).Error(err.Error())
			return err
		}

		balanceAfter = balance.Sub(amount)
		_, internalError = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			internalError = pipe.Set(ctx, redisKey, balanceAfter.IntPart(), time.Duration(0)).Err()
			if internalError != nil {
				s.logger.WithFields(logrus.Fields{"acc": acc, "error": internalError, "balance_before": balanceBefore.String(), "balance_after": balanceAfter.String(), "amount": amount.String()}).Error(internalError.Error())
			}
			return internalError
		})
		if internalError != nil {
			err = ErrFailedUpdateBalance
			s.logger.WithFields(logrus.Fields{"acc": acc, "error": internalError, "amount": amount.String()}).Error(err.Error())
			return err
		}

		return internalError
	}
	for i := 0; i < maxRetries; i++ {
		err = s.redis.Watch(ctx, txf, redisKey)
		if err == nil {
			break
		}
	}
	if err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "amount": amount.String()}).Error("withdraw failed")
	} else {
		s.logger.WithFields(logrus.Fields{"acc": acc, "balance_before": balanceBefore.String(), "balance_after": balanceAfter.String(), "amount": amount.String()}).Info("withdraw success")
	}
	return
}

func (s *Service) FastGetBalance(acc string) (balance decimal.Decimal, err error) {
	balanceString, err := s.redis.Get(context.Background(), fmt.Sprint(RedisAccountKey, acc)).Result()
	if err != nil || balanceString == "" {
		err = ErrAccountNotFound
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err}).Error(err.Error())
		return
	}

	balance, err = decimal.NewFromString(balanceString)
	if err != nil {
		err = ErrInternalError
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "balance_string": balanceString}).Error(ErrInternalError.Error())
		return
	}
	return
}

func (s *Service) FastDeposit(acc string, amount decimal.Decimal) (err error) {
	var balanceBefore decimal.Decimal
	var balanceAfter decimal.Decimal
	err = s.redis.Watch(context.Background(), func(tx *redis.Tx) error {
		balanceString, err := tx.Get(context.Background(), fmt.Sprint(RedisAccountKey, acc)).Result()
		if err != nil || balanceString == "" {
			err = ErrAccountNotFound
			s.logger.WithFields(logrus.Fields{"acc": acc, "error": err}).Error(err.Error())
			return err
		}

		balance, err := decimal.NewFromString(balanceString)
		if err != nil {
			err = ErrInternalError
			s.logger.WithFields(logrus.Fields{"acc": acc, "balance_string": balanceString}).Error(err.Error())
			return err
		}

		balanceBefore = balance
		balanceAfter = balance.Add(amount)
		err = tx.Set(context.Background(), fmt.Sprint(RedisAccountKey, acc), balanceAfter, time.Duration(0)).Err()
		if err != nil {
			err = ErrFailedUpdateBalance
			s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "amount": amount.String()}).Error(err.Error())
			return err
		}

		return nil
	})

	if err != nil {
		s.logger.WithFields(logrus.Fields{"acc": acc, "error": err, "amount": amount.String()}).Error("deposit failed")
	} else {
		s.logger.WithFields(logrus.Fields{"acc": acc, "balance_before": balanceBefore.String(), "balance_after": balanceAfter.String(), "amount": amount.String()}).Info("deposit success")
	}
	return
}
