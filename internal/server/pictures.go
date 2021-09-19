package server

import (
	"archive/zip"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
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

	session, filename := matches[1], matches[2]
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

	writer.Header().Set("Content-Type", "image/jpeg")
	_, err = io.Copy(writer, file)
	if err != nil {
		log.Printf("image send error: %s", err.Error())
		http.Error(writer, "Internal Server Error", 500)
		return
	}
}
