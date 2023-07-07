package main

import (
	"context"
	"errors"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	glog "github.com/labstack/gommon/log"
	"github.com/spf13/viper"

	"github.com/allv/proxy/app"
	"github.com/allv/proxy/metrics"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("read config file: %v", err)
	}

	endpoint := viper.GetString("rpc-endpoint")
	u, err := url.Parse(endpoint)
	if err != nil {
		log.Fatalf("invalid rpc-endpoint value (%s): %v", endpoint, err)
	}

	// app
	proxy := app.New(endpoint)
	wrapped := metrics.New().Wrap(proxy)

	// web
	e := echo.New()

	logger, ok := e.Logger.(*glog.Logger)
	if ok {
		logger.SetLevel(glog.INFO)
		logger.SetHeader(`{"time":"${time_rfc3339_nano}","level":"${level}","prefix":"${prefix}""}`)
	}

	e.Use(echoprometheus.NewMiddleware("allv"))
	e.GET("/metrics", echoprometheus.NewHandler())

	// timeout if the request takes more than 5 seconds to avoid goroutine pile-up
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 5 * time.Second,
	}))

	e.GET("/live", func(c echo.Context) error {
		// request balance for an existing non empty wallet
		b, err := proxy.GetBalance(c.Request().Context(), "0xE5D015A6D172000Cd497F4Fb625Aa48d1c2f7875")
		if err != nil || b.Cmp(big.NewInt(0)) == 0 {
			return c.NoContent(http.StatusServiceUnavailable)
		}
		return c.NoContent(http.StatusOK)
	})

	e.GET("/ready", func(c echo.Context) error {
		if _, err := net.DialTimeout("tcp", u.Host+":80", 1*time.Second); err != nil {
			return c.NoContent(http.StatusServiceUnavailable)
		}
		return c.NoContent(http.StatusOK)
	})

	e.GET("/eth/balance/:addr", func(c echo.Context) error {
		b, err := wrapped.GetBalance(c.Request().Context(), c.Param("addr"))
		switch {
		case errors.Is(err, app.ErrInvalidAddress):
			return c.NoContent(http.StatusBadRequest)
		default:
			if err != nil {
				logger.Error("get balance", err)
				c.NoContent(http.StatusInternalServerError)
			}
		}
		return c.JSONPretty(http.StatusOK, struct {
			Balance string `json:"balance"`
		}{Balance: b.String()}, "  ")
	})

	go func() {
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("server shut down: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit
	logger.Info("received interrupt signal...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Fatal(err)
	}
}
