package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	db := InitDatabase()
	rdb := InitRedis()
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
