package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var staticRoot = os.Getenv("STATIC_ROOT")

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
	v1.Get("copy", func(c *fiber.Ctx) error {
		src := c.Query("src")
		dest := c.Query("dest")
		src = "docker://" + src
		dest = "docker://" + dest
		go func() {
			log.Println("Start Copy", "src", src, "dest", dest)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := runCopy(ctx, src, dest); err != nil {
				log.Printf("[ERROR] failed to run copy: %+v", err)
			}
		}()
		return c.SendString("Copying, please wait")
	})
	return app
}
