package main

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Account struct {
	Acc     string `gorm:"primaryKey"`
	Balance decimal.Decimal
}

func InitDatabase() (db *gorm.DB) {
	dsn := "host=localhost user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	return
}

func InitRedis() (rdb *redis.Client) {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		panic(err)
	}
	return
}
