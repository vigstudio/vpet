package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shourenso/vpet/internal/game"
	"github.com/shourenso/vpet/internal/httpapi"
	"github.com/shourenso/vpet/internal/store"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	databaseURL := required("DATABASE_URL")
	publicURL := required("PUBLIC_URL")
	secretText := required("TRANSFER_SECRET")
	secret, err := hex.DecodeString(secretText)
	if err != nil || len(secret) < 32 {
		log.Fatal("TRANSFER_SECRET must be at least 32 random bytes encoded as hex")
	}
	db, err := store.Open(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = db.MigrateAll(ctx); err != nil {
		log.Fatal(err)
	}
	if err = db.RecoverMarketJournal(ctx); err != nil {
		log.Fatalf("recover imported market journal: %v", err)
	}
	engine := game.NewEngine(db)
	runtimeCtx, stopRuntime := context.WithCancel(context.Background())
	defer stopRuntime()
	go engine.Run(runtimeCtx)
	secure := envBool("COOKIE_SECURE", strings.HasPrefix(publicURL, "https://"))
	api, err := httpapi.New(db, engine, publicURL, secret, secure)
	if err != nil {
		log.Fatal(err)
	}
	root, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	static := spaHandler(http.FS(root))
	server := &http.Server{
		Addr:              env("LISTEN_ADDR", ":8080"),
		Handler:           api.Handler(static),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// WriteTimeout stays disabled because notification SSE responses are long-lived.
		IdleTimeout: 60 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("vpet listening on %s", server.Addr)
		serverErr <- server.ListenAndServe()
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case err = <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-stop:
		stopRuntime()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()
		if err = server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}
}

// spaHandler serves the embedded web app. The bundle keeps the name main.js across releases,
// so a proxy caching by URL would hand players the previous build for as long as its own
// default TTL (Cloudflare used four hours). "no-cache" means revalidate, not "do not store":
// paired with a content ETag the browser still gets a 304 for an unchanged bundle.
//
// Embedded files report a zero modification time, so net/http sends no Last-Modified and has
// no validator of its own; the ETag below is what keeps revalidation cheap.
func spaHandler(files http.FileSystem) http.Handler {
	fileServer := http.FileServer(files)
	tags := map[string]string{}
	var mu sync.RWMutex

	etag := func(name string) string {
		mu.RLock()
		tag, ok := tags[name]
		mu.RUnlock()
		if ok {
			return tag
		}
		f, err := files.Open(name)
		if err != nil {
			return ""
		}
		defer f.Close()
		sum := sha256.New()
		if _, err = io.Copy(sum, f); err != nil {
			return ""
		}
		tag = fmt.Sprintf("%q", hex.EncodeToString(sum.Sum(nil))[:32])
		mu.Lock()
		tags[name] = tag
		mu.Unlock()
		return tag
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := "index.html"
		if r.URL.Path != "/" {
			if f, err := files.Open(strings.TrimPrefix(r.URL.Path, "/")); err == nil {
				_ = f.Close()
				name = strings.TrimPrefix(r.URL.Path, "/")
			} else {
				r.URL.Path = "/"
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		if tag := etag(name); tag != "" {
			w.Header().Set("ETag", tag)
		}
		fileServer.ServeHTTP(w, r)
	})
}

func required(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return fallback
	}
}
