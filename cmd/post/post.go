package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/VTGare/softserve-homework/internal/config"
	"github.com/VTGare/softserve-homework/internal/database"
	"github.com/VTGare/softserve-homework/internal/middlewares"
	"github.com/VTGare/softserve-homework/pkg/post"
	"github.com/VTGare/softserve-homework/pkg/post/endpoints"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	defer logger.Sync()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	sugar := logger.Sugar()

	//Load app's configuration from config.json file in working directory
	cfg, err := config.New("config.json")
	if err != nil {
		fmt.Println("Failed to load config. Error: ", err)
		os.Exit(1)
	}

	db, err := database.New(cfg.Redis.Host, cfg.Redis.Port)
	if err != nil {
		fmt.Println("Failed to connect to Redis. Error: ", err)
		os.Exit(1)
	}
	defer db.Close()

	postService := post.NewService(db, sugar)

	ep := endpoints.NewEndpointSet(postService)

	srv := createServer(cfg, ep, sugar)
	go func() {
		sugar.Infof("Listening on %v", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			sugar.Infof("Error: %v", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srv.Shutdown(ctx)

	sugar.Info("shutting down")
	os.Exit(0)
}

func createServer(cfg *config.Config, ep *endpoints.Set, logger *zap.SugaredLogger) *http.Server {
	r := mux.NewRouter()

	r.Use(middlewares.Logger(logger), middlewares.Recover(logger))

	r.Methods("GET").Path("/api/posts/{id}").HandlerFunc(ep.GetEndpoint)
	r.Methods("DELETE").Path("/api/posts/{id}").HandlerFunc(ep.DeleteEndpoint)
	r.Methods("GET").Path("/api/posts").HandlerFunc(ep.SearchEndpoint)
	r.Methods("POST").Path("/api/posts").HandlerFunc(ep.AddEndpoint)
	r.Methods("GET").Path("/api/count").HandlerFunc(ep.CountEndpoint)

	return &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}
}
