package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/egafa/yandexdiplom/config"
	"github.com/egafa/yandexdiplom/internal/handler"
	"github.com/egafa/yandexdiplom/storage"
	"github.com/egafa/yandexdiplom/zipcompess"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.LoadConfig()

	log.Println("Запуск Сервера", cfg.AddrServer)
	log.Println("DatabaseDSN ", *&cfg.DatabaseDSN)
	log.Println("AccuralAddress ", *&cfg.AccuralAddress)

	repo, err := storage.NewRepo(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer repo.Close()

	ctx, cancel := context.WithCancel(context.Background())

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	//r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(zipcompess.DecompressHandle)
	r.Use(zipcompess.GzipHandle)
	//r.Use(handler.UserIdentity)

	r.Route("/api/user/register", func(r chi.Router) {
		r.Post("/", handler.RegisterUser(&repo))
	})

	r.Route("/api/user/login", func(r chi.Router) {
		r.Post("/", handler.LoginUser(&repo))
	})

	r.Route("/api/user/orders", func(r chi.Router) {
		r.Use(handler.UserIdentity)
		r.Post("/", handler.LoadOrder(&repo))
		r.Get("/", handler.GetOrders(&repo))
	})

	r.Route("/api/user/balance", func(r chi.Router) {
		r.Use(handler.UserIdentity)
		r.Get("/", handler.GetBalance(&repo))
		r.Post("/withdraw", handler.LoadWithdraw(&repo))
		r.Get("/withdrawals", handler.GetListWithdraws(&repo))
	})

	srv := &http.Server{
		Handler: r,
		Addr:    cfg.AddrServer,
	}

	idleConnsClosed := make(chan struct{})

	go sendReq(ctx, cfg, &repo)

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		log.Print("HTTP server Shutdown")
		close(idleConnsClosed)
		cancel()
	}()

	log.Print("Запуск сервера HTTP")

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed

	log.Print("HTTP server close")

}
