package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

var (
	NUM_ACCOUNT     = 100
	INITIAL_BALANCE = decimal.NewFromInt(1_000_000)
)

func main() {
	logger := logrus.New()
	db := InitDatabase()
	rdb := InitRedis()
	Seeding(NUM_ACCOUNT, INITIAL_BALANCE, db, rdb)
	service := NewService(db, rdb, logger)
	api := fiber.New()
	api.Get("/balance/:acc", func(c *fiber.Ctx) error {
		acc := c.Params("acc")
		balance, err := service.GetBalance(acc)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(fiber.Map{"remark": "inquiry balance failed"})
		}
		return c.JSON(balance)
	})
}
