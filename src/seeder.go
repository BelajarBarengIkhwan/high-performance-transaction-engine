package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ACCOUNT_PREFIX = "ACCOUNT-%d"
)

func Seeding(N int, balance decimal.Decimal, db *gorm.DB, rdb *redis.Client) {
	accounts := []*Account{}
	for i := range N {
		account := fmt.Sprintf(ACCOUNT_PREFIX, i)
		accounts = append(accounts, &Account{Acc: account, Balance: balance})
	}

	db.AutoMigrate(&Account{})
	db.Where("1=1").Delete(&Account{})
	err := db.Create(accounts).Error
	if err != nil {
		panic(err)
	}

	pipe := rdb.Pipeline()
	for _, acc := range accounts {
		pipe.Set(context.Background(), fmt.Sprintf(RedisAccountKey, acc.Acc), balance, time.Duration(0))
	}

	_, err = pipe.Exec(context.Background())
	if err != nil {
		panic(err)
	}
}
