package main

import (
	"flag"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
)

func init() {
	flag.StringVar(&ConfigFile, "file", "", "Path to config file")
	if ConfigFile == "" {
		log.Fatal("Config file not specified, use default")
		f, _ := os.Create("config.ini")
		f.WriteString(DefaultConfig)
	}
}

func main() {
	route := chi.NewMux()
	config := loadIni()
	route.Handle("/", handlers.LoggingHandler(os.Stdout, config.NewWebReverseProxy()))
	route.Handle("/metrics", promhttp.Handler())
	log.Infof("Starting server on %s", config.HttpAddress)
	// Start the server
	switch {
	case config.IsPrivate == true && config.UseTLS.Use == true:
		if err := http.ListenAndServeTLS(config.HttpAddress, config.UseTLS.CertFile, config.UseTLS.KeyFile, route); err != nil {
			log.Fatalf("Failed to start terraform registry proxy\nERR:%v", err)
		}
	default:
		if err := http.ListenAndServe(config.HttpAddress, route); err != nil {
			log.Fatalf("Failed to start terraform registry proxy\nERR:%v", err)
		}
	}
	recover()
}
