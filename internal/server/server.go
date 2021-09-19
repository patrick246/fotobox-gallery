package server

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var httpRequestDurationHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name: "http_request_duration_seconds",
	Help: "Endpoint response time histogram",
}, []string{"path", "code", "method"})

type Server struct {
	server        http.Server
	dataDirectory string
}

func init() {
	prometheus.MustRegister(httpRequestDurationHistogram)
}

func NewServer(port uint, dataDirectory string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/ready", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
	})
	mux.Handle("/metrics", promhttp.Handler())

	s := &Server{
		server: http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		dataDirectory: dataDirectory,
	}

	mux.Handle("/pictures/", promhttp.InstrumentHandlerDuration(httpRequestDurationHistogram.MustCurryWith(prometheus.Labels{
		"path": "/pictures/*",
	}), http.HandlerFunc(s.handleFileRequests)))

	return s
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}
