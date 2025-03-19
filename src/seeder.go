package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	ACCOUNT_PREFIX = "ACCOUNT-%d"
)

func Seeding(N int, balance decimal.Decimal, db *gorm.DB, rdb *redis.Client, logger *logrus.Logger) {
	logger.WithFields(logrus.Fields{"count": N, "balance": balance.String()}).Info("generate account seeds")
	accounts := []*Account{}
	for i := range N {
		account := fmt.Sprintf(ACCOUNT_PREFIX, i)
		accounts = append(accounts, &Account{Acc: account, Balance: balance})
	}

	logger.Info("migrate database")
	db.AutoMigrate(&Account{})
	logger.Info("delete all accounts")
	db.Where("1=1").Delete(&Account{})
	logger.WithFields(logrus.Fields{"count": len(accounts)}).Info("insert accounts")
	err := db.Create(accounts).Error
	if err != nil {
		panic(err)
	}

	logger.WithFields(logrus.Fields{"count": len(accounts)}).Info("cache accounts")
	pipe := rdb.Pipeline()
	for _, acc := range accounts {
		pipe.Set(context.Background(), fmt.Sprintf(RedisAccountKey, acc.Acc), balance.IntPart(), time.Duration(0))
	}

	_, err = pipe.Exec(context.Background())
	if err != nil {
		panic(err)
	}

	logger.WithFields(logrus.Fields{"count": len(accounts)}).Info("seeding success")
}
