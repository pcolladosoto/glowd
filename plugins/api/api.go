package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	glowdTypes "github.com/scitags/flowd-go/types"
)

var (
	Defaults = map[string]interface{}{
		"bindAddress": "127.0.0.1",
		"bindPort":    7777,
	}
)

type ApiPlugin struct {
	server *echo.Echo

	BindAddress string `json:"bindAddress"`
	BindPort    int    `json:"bindPort"`
}

func (p *ApiPlugin) String() string {
	return "api"
}

func (p *ApiPlugin) Init() error {
	slog.Debug("initialising the api plugin")
	p.server = echo.New()

	// Configure the methods for each path
	p.server.GET("/", handleRoot)
	p.server.GET("/dummy/start", handleDummyStartFlow)
	p.server.GET("/dummy/end", handleDummyEndFlow)

	// Prevent the banner from showing up in the log
	p.server.HideBanner = true
	p.server.HidePort = true

	return nil
}

func (p *ApiPlugin) Run(done <-chan struct{}, outChan chan<- glowdTypes.FlowID) {
	slog.Debug("running the api plugin")

	// Configure the middleware for extending the context of the
	// different handlers. We defer that to this point so that we
	// have a handle of the channel we are to push events into.
	p.server.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(&extendedContext{c, p.server.Routes(), outChan})
		}
	})

	go func() {
		if err := p.server.Start(fmt.Sprintf("%s:%d", p.BindAddress, p.BindPort)); err != http.ErrServerClosed {
			slog.Error("couldn't start the API server", "err", err)
		}
	}()

	// Simply wait until we're done
	<-done
	slog.Debug("cleanly exiting the api plugin")
}

func (p *ApiPlugin) Cleanup() error {
	slog.Debug("cleaning up the api plugin")
	if err := p.server.Shutdown(context.TODO()); err != nil {
		return fmt.Errorf("error shutting down the API server: %w", err)
	}
	return nil
}
