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

type WithdrawRequest struct {
	Acc    string          `json:"acc"`
	Amount decimal.Decimal `json:"amount"`
}

func main() {
	logger := logrus.New()
	db := InitDatabase()
	rdb := InitRedis()
	Seeding(NUM_ACCOUNT, INITIAL_BALANCE, db, rdb, logger)
	service := NewService(db, rdb, logger)
	api := fiber.New()
	api.Post("/withdraw", func(c *fiber.Ctx) error {
		var req WithdrawRequest

		if c.Get("Content-Type") != "application/json" {
			c.Status(http.StatusBadRequest)
			return c.JSON(fiber.Map{"remark": "invalid request header"})
		}

		if err := c.BodyParser(&req); err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(fiber.Map{"remark": "invalid request"})
		}

		err := service.Withdraw(req.Acc, req.Amount)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(fiber.Map{"remark": "inquiry balance failed"})
		}
		return c.SendStatus(http.StatusNoContent)
	})
	api.Post("/fast-withdraw", func(c *fiber.Ctx) error {
		var req WithdrawRequest

		if c.Get("Content-Type") != "application/json" {
			c.Status(http.StatusBadRequest)
			return c.JSON(fiber.Map{"remark": "invalid request header"})
		}

		if err := c.BodyParser(&req); err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(fiber.Map{"remark": "invalid request"})
		}

		err := service.FastWithdraw(req.Acc, req.Amount)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(fiber.Map{"remark": "inquiry balance failed"})
		}
		return c.SendStatus(http.StatusNoContent)
	})
	api.Get("/balance/:acc", func(c *fiber.Ctx) error {
		acc := c.Params("acc")
		balance, err := service.GetBalance(acc)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(fiber.Map{"remark": "inquiry balance failed"})
		}
		return c.JSON(fiber.Map{"balance": balance.String()})
	})
	api.Get("/fast-balance/:acc", func(c *fiber.Ctx) error {
		acc := c.Params("acc")
		balance, err := service.FastGetBalance(acc)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return c.JSON(fiber.Map{"remark": "inquiry balance failed"})
		}
		return c.JSON(fiber.Map{"balance": balance.String()})
	})
	api.Listen(":8080")
}
