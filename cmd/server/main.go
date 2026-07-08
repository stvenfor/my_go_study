package main

import (
	"log"
	"net/http"

	"github.com/stvenfor/my_go_study/internal/config"
	"github.com/stvenfor/my_go_study/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	app, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("backend 启动于 %s，Supabase: %s", app.Addr(), cfg.Supabase.URL)
	if err := http.ListenAndServe(app.Addr(), app.Handler()); err != nil {
		log.Fatal(err)
	}
}
