package server

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"github.com/nfnt/resize"
	"html/template"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

var pictureRegex = regexp.MustCompile(`/pictures/(?P<session>[0-9a-zA-Z]{6})(?:/(?P<filename>.*\.(?:jpg|png|zip)))?`)

var funcs = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
}

//go:embed sessionlist.html
var sessionListTemplateSource string
var sessionListTemplate = template.Must(template.New("session_list").Funcs(funcs).Parse(sessionListTemplateSource))

type SessionTemplateData struct {
	SessionID string
	Files     []string
}

func (s *Server) handleFileRequests(writer http.ResponseWriter, request *http.Request) {
	matches := pictureRegex.FindStringSubmatch(request.URL.Path)
	if len(matches) != 3 {
		http.NotFound(writer, request)
		return
	}

	session, filename := strings.ToUpper(matches[1]), matches[2]
	if filename == "" {
		s.handleSessionList(writer, request, session)
		return
	} else if strings.HasSuffix(filename, ".zip") {
		s.handleSessionDownload(writer, request, session)
		return
	}
	s.handleDirectAccess(writer, request, session, filename)
}

func (s *Server) handleSessionList(writer http.ResponseWriter, request *http.Request, session string) {
	if !strings.HasSuffix(request.URL.Path, "/") {
		writer.Header().Set("Location", fmt.Sprintf("/pictures/%s/", session))
		writer.WriteHeader(301)
		return
	}

	directoryPath := path.Join(s.dataDirectory, session)
	files, err := ioutil.ReadDir(directoryPath)
	if errors.Is(err, syscall.ENOENT) {
		http.NotFound(writer, request)
		return
	}
	if err != nil {
		log.Printf("dir list error: %s", err.Error())
		http.Error(writer, "Internal Server Error", 500)
		return
	}

	var filenames []string
	for _, file := range files {
		if !file.IsDir() {
			filenames = append(filenames, file.Name())
		}
	}

	err = sessionListTemplate.ExecuteTemplate(writer, "session_list", SessionTemplateData{
		SessionID: session,
		Files:     filenames,
	})
	if err != nil {
		log.Printf("template rendering error: %s", err.Error())
		http.Error(writer, "Internal Server Error", 500)
		return
	}
}

func (s *Server) handleSessionDownload(writer http.ResponseWriter, request *http.Request, session string) {
	directoryPath := path.Join(s.dataDirectory, session)
	files, err := ioutil.ReadDir(directoryPath)
	if errors.Is(err, syscall.ENOENT) {
		http.NotFound(writer, request)
		return
	}
	if err != nil {
		log.Printf("dir list error: %s", err.Error())
		http.Error(writer, "Internal Server Error", 500)
		return
	}

	writer.Header().Set("Content-Type", "application/zip")
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".jpg") && !strings.HasSuffix(file.Name(), ".png") {
			continue
		}

		zipFileWriter, err := zipWriter.Create(file.Name())
		if err != nil {
			log.Printf("zip creation error: %s", err.Error())
			http.Error(writer, "Internal Server Error", 500)
			return
		}

		file, err := os.Open(path.Join(directoryPath, file.Name()))
		if err != nil {
			log.Printf("file open error: %s", err.Error())
			http.Error(writer, "Internal Server Error", 500)
			return
		}

		_, err = io.Copy(zipFileWriter, file)
		if err != nil {
			log.Printf("zip copy error: %s", err.Error())
			http.Error(writer, "Internal Server Error", 500)
			return
		}
	}
}

func (s *Server) handleDirectAccess(writer http.ResponseWriter, request *http.Request, session, filename string) {
	thumbnail, err := strconv.ParseBool(request.URL.Query().Get("thumbnail"))
	if err != nil {
		thumbnail = false
	}

	var mimeType string
	switch {
	case strings.HasSuffix(filename, ".png"):
		mimeType = "image/png"
	case strings.HasSuffix(filename, ".jpg"), strings.HasSuffix(filename, ".jpeg"):
		mimeType = "image/jpeg"
	}

	imagePath := path.Join(s.dataDirectory, session, filename)
	file, err := os.Open(imagePath)
	if errors.Is(err, syscall.ENOENT) {
		http.NotFound(writer, request)
		return
	}
	if err != nil {
		log.Printf("image read error: %s", err.Error())
		http.Error(writer, "Internal Server Error", 500)
		return
	}

	var imgSource io.Reader = file
	if thumbnail {
		imgSource, err = resizeImage(file, s.thumbnailWidth)
		if err != nil {
			log.Printf("resize failed: %s. continuing with full-size image", err.Error())
			imgSource = file
		} else {
			// We always re-encode as JPEG
			mimeType = "image/jpeg"
		}
	}

	writer.Header().Set("Content-Type", mimeType)
	_, err = io.Copy(writer, imgSource)
	if err != nil {
		log.Printf("image send error: %s", err.Error())
		http.Error(writer, "Internal Server Error", 500)
		return
	}
}

func resizeImage(imgSource io.Reader, width uint) (io.Reader, error) {
	img, _, err := image.Decode(imgSource)
	if err != nil {
		return nil, err
	}

	resized := resize.Resize(width, 0, img, resize.Bilinear)

	buffer := &bytes.Buffer{}

	err = jpeg.Encode(buffer, resized, &jpeg.Options{
		Quality: jpeg.DefaultQuality,
	})
	if err != nil {
		return nil, err
	}
	return buffer, nil
}
