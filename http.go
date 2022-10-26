package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var (
	staticRoot = os.Getenv("STATIC_ROOT")
	loginKey   = os.Getenv("LOGIN_KEY")
)

func setupHttpApp() *fiber.App {
	// Create fiber app
	app := fiber.New(fiber.Config{
		Prefork: false, // go run app.go -prod
	})
	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Get("/sse", handleSse)

	app.Static("/", staticRoot)

	// Create a /api endpoint
	v1 := app.Group("/api")
	v1.Post("copy", func(c *fiber.Ctx) error {
		var postData map[string]string
		if err := c.BodyParser(&postData); err != nil {
			return fmt.Errorf("invalid json body: %v", err)
		}
		inputKey := postData["loginKey"]
		if inputKey != loginKey {
			return sendFailure(c, 401, 401, "invalid login key")
		}

		go actionCopy(postData["src"], postData["dest"])
		return sendSuccess(c, nil)
	})
	return app
}

func sendSuccess(c *fiber.Ctx, data interface{}) error {
	return c.JSON(map[string]interface{}{"errcode": 0, "data": data})
}

func sendFailure(c *fiber.Ctx, statusCode, errcode int, errmsg string) error {
	return c.Status(statusCode).JSON(map[string]interface{}{"errcode": errcode, "errmsg": errmsg})
}

func actionCopy(src, dest string) {
	src = "docker://" + src
	dest = "docker://" + dest
	log.Println("Start Copy", "src", src, "dest", dest)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := runCopy(ctx, src, dest); err != nil {
		logEvent(err.Error())
		log.Printf("[ERROR] failed to run copy: %+v", err)
	}
}
