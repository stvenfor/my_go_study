package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/stvenfor/my_go_study/internal/config"
	"github.com/stvenfor/my_go_study/internal/handler"
	appmiddleware "github.com/stvenfor/my_go_study/internal/middleware"
	"github.com/stvenfor/my_go_study/internal/repository"
	"github.com/stvenfor/my_go_study/internal/supabase"
)

type App struct {
	cfg    config.Config
	router chi.Router
}

func New(cfg config.Config) (*App, error) {
	sbClient, err := supabase.New(cfg.Supabase)
	if err != nil {
		return nil, err
	}

	profileRepo := repository.NewProfileRepository(sbClient)
	transactionRepo := repository.NewTransactionRepository(sbClient)

	profileHandler := handler.NewProfileHandler(profileRepo)
	transactionHandler := handler.NewTransactionHandler(transactionRepo)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", handler.HealthHandler{}.ServeHTTP)

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(authRoutes chi.Router) {
			authRoutes.Use(appmiddleware.Auth(cfg))
			authRoutes.Get("/me/profile", profileHandler.GetMe)
			authRoutes.Patch("/me/profile", profileHandler.UpdateMe)
		})

		api.Route("/transactions", func(tr chi.Router) {
			tr.Get("/", transactionHandler.List)
			tr.Get("/{id}", transactionHandler.Get)
			tr.Post("/", transactionHandler.Create)
			tr.Put("/{id}", transactionHandler.Update)
			tr.Delete("/{id}", transactionHandler.Delete)
		})
	})

	return &App{cfg: cfg, router: r}, nil
}

func (a *App) Handler() http.Handler {
	return a.router
}

func (a *App) Addr() string {
	return a.cfg.Server.Addr
}
