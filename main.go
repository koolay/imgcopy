package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	cli "github.com/urfave/cli/v2"
)

type Config struct {
	DockerAuthName   string
	DockerAuthToken  string
	DockerConfigFile string
}

var (
	cfg     Config
	verbose bool
	port    int = 8080
)

func runCopy(ctx context.Context, src, dest string) error {
	cmd := NewCommand(func(line string) {
		logEvent(fmt.Sprintf("%s %s", time.Now().Format("2006-01-02T15:04:05.000"), line))
	}, func(line string) {
		logEvent(fmt.Sprintf("%s %s", time.Now().Format("2006-01-02T15:04:05.000"), line))
	})
	args := []string{
		"--debug",
		"--insecure-policy",
		"copy",
		"--dest-authfile",
		cfg.DockerConfigFile,
		"--all",
		"--src-tls-verify=false",
		src,
		dest,
	}
	fmt.Printf("command args: %v", args)
	return cmd.Run(ctx, "skopeo", args, nil)
}

func loadConfig() {
	cfg.DockerAuthName = os.Getenv("DOCKER_AUTH_NAME")
	cfg.DockerAuthToken = os.Getenv("DOCKER_AUTH_TOKEN")
	cfg.DockerConfigFile = os.Getenv("DOCKER_CONFIG_FILE")
	if cfg.DockerConfigFile == "" {
		cfg.DockerConfigFile = ".docker.json"
	}

	portEnv := os.Getenv("PORT")
	if portEnv != "" {
		var err error
		port, err = strconv.Atoi(portEnv)
		if err != nil {
			log.Fatalf("invalid port: %v", portEnv)
		}
	}
}

func initDockerToken(tokenFile, authName, token string) error {
	log.Println("writie dockerconfig file:", tokenFile)
	if err := os.WriteFile(tokenFile, []byte(`{
	"auths": {
		"`+authName+`": {
			"auth": "`+token+`"
		}
	}
}`), 0755); err != nil {
		return fmt.Errorf("failed to write docker config file, file: %s, error: %w", cfg.DockerConfigFile, err)
	}

	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)
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
			loadConfig()
			if _, err := os.Stat(cfg.DockerConfigFile); os.IsNotExist(err) {
				if cfg.DockerAuthName != "" && cfg.DockerAuthToken != "" {
					if err := initDockerToken(cfg.DockerConfigFile, cfg.DockerAuthName, cfg.DockerAuthToken); err != nil {
						return err
					}
				}
			}

			app := setupHttpApp()

			// go func() {
			// 	for {
			// 		broker.Messages <- "time is:" + time.Now().String()
			// 		time.Sleep(time.Second)
			// 	}
			// }()
			go broker.listen()

			go func() {
				log.Println("start to listen on", port)
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
