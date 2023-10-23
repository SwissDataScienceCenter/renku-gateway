package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	config, err := getConfig()
	if err != nil {
		log.Fatalln(err)
	}
	server, err := NewLoginServer(&config)
	if err != nil {
		log.Fatalln(err)
	}

	// Start server
	go func() {
		if err := server.echo.Start(fmt.Sprintf(":%d", config.Server.Port)); err != nil && err != http.ErrServerClosed {
			server.echo.Logger.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.echo.Shutdown(ctx); err != nil {
		server.echo.Logger.Fatal(err)
	}
}
