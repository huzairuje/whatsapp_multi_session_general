package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"whatsapp_multi_session_general/boot"
	"whatsapp_multi_session_general/config"
	"whatsapp_multi_session_general/primitive"

	"github.com/gin-gonic/gin"
	"github.com/gookit/event"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	//parse flag
	flag.StringVar(&config.Env, "env", "local", "A config name that used by server")
	flag.Parse()

	// Create a new Gin router
	router := gin.Default()

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not Matching of Any Routes"})
	})

	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Method Not Allowed"})
	})

	//initiate
	appSetupHandler := boot.Setup(router)

	port := fmt.Sprintf(":%v", config.Conf.Port)
	if port == "" {
		port = fmt.Sprintf(":%v", 1234)
	}

	log.Printf("Server running on port %s", port)
	serve := &http.Server{
		Addr:    port,
		Handler: appSetupHandler,
	}

	// Start server
	go func() {
		if err := serve.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 1 second.
	quit := make(chan os.Signal)
	// kill (no param) default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	event.MustFire(primitive.ShutDownEvent, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := serve.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	select {
	case <-ctx.Done():
		log.Println("timeout of 1 seconds.")
	}
	log.Println("Server exiting")
}
