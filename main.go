package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	logger := logrus.New()
	service := NewService(db, logger)
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
