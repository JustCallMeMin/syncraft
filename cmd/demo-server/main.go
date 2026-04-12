package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/JustCallMeMin/syncraft/internal/demo/browser"
)

func main() {
	listenAddr := flag.String("listen", ":8080", "HTTP listen address")
	dataDir := flag.String("data-dir", defaultDataDir(), "Persistence directory for demo state")
	flag.Parse()

	server, err := browser.NewServer(*dataDir)
	if err != nil {
		log.Fatalf("create browser demo server: %v", err)
	}
	handler, err := server.Handler()
	if err != nil {
		log.Fatalf("build browser demo handler: %v", err)
	}

	log.Printf("syncraft demo server listening on http://localhost%s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, handler); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}

func defaultDataDir() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(".", ".syncraft-demo")
	}
	return filepath.Join(cacheDir, "syncraft-demo")
}
