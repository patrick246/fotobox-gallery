package main

import (
	"flag"
	"github.com/patrick246/fotobox-gallery/internal/server"
	"log"
	"os"
)

var dataDir string

func main() {
	flag.StringVar(&dataDir, "data-dir", "", "The data directory to render photos from")
	flag.Parse()

	if dataDir == "" {
		dataDir = os.Getenv("FOTOBOX_DATA_DIR")
	}
	if dataDir == "" {
		dataDir = "/data"
	}

	srv := server.NewServer(8080, dataDir)
	log.Println("listening on :8080")
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
