package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	app := cli.NewApp()
	app.Name = "hookup"
	app.Usage = "Start Webhook Server"

	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "port",
			Value: 9090,
		},
		cli.StringFlag{
			Name:  "handlers",
			Usage: "Path to dir with webhook handlers scripts",
			Value: "/etc/hookup.d/",
		},
	}

	app.Action = func(c *cli.Context) {
		StartWebhookServer(c.Int("port"), func(source string, eventType string, payload string) {
			for _, cmd := range findHandlerCmds(c.String("handlers")) {
				go execHandler(cmd, source, eventType, payload)
			}
		})
	}

	app.Run(os.Args)
}

func StartWebhookServer(port int, handler func(source string, eventType string, payload string)) {
	ec := echo.New()
	ec.Use(middleware.Logger())
	ec.Use(middleware.Recover())

	ec.Post("/github/events", func(c *echo.Context) error {
		defer c.Request().Body.Close()

		eventType := c.Request().Header.Get("X-GitHub-Event")
		payload, err := ioutil.ReadAll(c.Request().Body)

		log.Printf("Receive github event '%s': \n%s\n", eventType, string(payload))

		handler("github", eventType, string(payload))

		return err
	})

	log.Printf("Starting web hook server on :%v\n", port)

	ec.Run(":" + strconv.Itoa(port))
}

func findHandlerCmds(handlersDir string) []string {
	handlers := make([]string, 0)

	err := filepath.Walk(handlersDir, func(path string, f os.FileInfo, err error) error {
		if err == nil && !f.IsDir() {
			handlers = append(handlers, path)
		}
		return err
	})

	if err != nil || len(handlers) == 0 {
		log.Printf("Handlers not found: %v\n", err)
	}

	return handlers
}

func execHandler(handler string, source string, eventType string, payload string) {
	log.Printf("Run %s", handler)

	out, err := exec.Command("/bin/bash", handler, "--source", source, "--event", eventType, "--payload", payload).CombinedOutput()

	if err != nil {
		log.Printf("Error: %s\n\n", err)
	} else {
		log.Printf("Out: \n%s\n\n", string(out))
	}
}