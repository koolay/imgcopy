package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cristalhq/aconfig"
	"github.com/cristalhq/aconfig/aconfigtoml"
	cli "github.com/urfave/cli/v2"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type Config struct {
	Port       int `default:"8080"`
	HwRegistry struct {
		User     string `default:"root" `
		Password string `default:"dev"`
	}
}

var (
	cfg     Config
	verbose bool
	port    int
)

func main() {
	loader := aconfig.LoaderFor(&cfg, aconfig.Config{
		SkipDefaults: false,
		SkipFiles:    true,
		SkipEnv:      true,
		SkipFlags:    true,
		EnvPrefix:    "IMGCOPY",
		Files:        []string{"config.toml"},
		FileDecoders: map[string]aconfig.FileDecoder{
			".toml": aconfigtoml.New(),
		},
	})

	if err := loader.Load(); err != nil {
		log.Fatalf("failed to load config, error: %v", err)
	}

	app := &cli.App{
		Name:  "imgcopy",
		Usage: "copy image to",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "verbose",
				Usage:       "print debug info",
				Aliases:     []string{"v"},
				Value:       false,
				Destination: &verbose,
			},
			&cli.IntFlag{
				Name:        "port",
				Usage:       "http listen port",
				Value:       8080,
				Destination: &port,
			},
		},
		Action: func(*cli.Context) error {
			os.Setenv("PORT", fmt.Sprintf("%d", port))
			// Create fiber app
			app := fiber.New(fiber.Config{
				Prefork: false, // go run app.go -prod
			})

			// Middleware
			app.Use(recover.New())
			app.Use(logger.New())
			app.Get("/", func(c *fiber.Ctx) error {
				return c.SendString("Hello world!")
			})

			// Create a /api/v1 endpoint
			// v1 := app.Group("/api/v1")

			log.Println("start to listen on", port)
			go func() {
				if err := app.Listen(fmt.Sprintf("0.0.0.0:%d", port)); err != nil {
					panic(err)
				}
			}()

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
			sig := <-quit
			log.Printf("Shutdown with %v\n", sig)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Panic(err)
	}
}
