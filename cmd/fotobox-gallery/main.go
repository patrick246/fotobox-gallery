package main

import (
	"flag"
	"github.com/patrick246/fotobox-gallery/internal/server"
	"log"
	"os"
)

func main() {
	var dataDir string
	var thumbnailWidth uint

	flag.StringVar(&dataDir, "data-dir", "", "The data directory to render photos from")
	flag.UintVar(&thumbnailWidth, "thumbnail.width", 400, "Thumbnail width for session preview")
	flag.Parse()

	if dataDir == "" {
		dataDir = os.Getenv("FOTOBOX_DATA_DIR")
	}
	if dataDir == "" {
		dataDir = "/data"
	}

	srv := server.NewServer(8080, dataDir, thumbnailWidth)
	log.Println("listening on :8080")
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
