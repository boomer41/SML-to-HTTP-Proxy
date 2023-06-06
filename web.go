package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

type webExporter struct {
	processImage *processImageManager
	logger       logger

	server *http.Server
}

func newWebExporter(processImage *processImageManager, log logger) *webExporter {
	return &webExporter{
		processImage: processImage,
		logger:       log,
	}
}

func (i *webExporter) serve(cfg *webConfig) error {
	if i.server != nil {
		return errors.New("server already started")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/processImage", i.getProcessImage)

	var handler http.Handler = mux

	if !cfg.DisableRequestLog {
		handler = &webLoggerHandler{
			logger: i.logger.newSubLogger("request"),
			next:   handler,
		}
	}

	i.server = &http.Server{
		Addr:    cfg.Address,
		Handler: handler,
	}

	return i.server.ListenAndServe()
}

func (i *webExporter) getProcessImage(resp http.ResponseWriter, _ *http.Request) {
	image := i.processImage.get()
	marshal, err := json.Marshal(image)

	if err != nil {
		i.logger.Printf("failed to serialize process image: %v", err)
		resp.WriteHeader(500)
		return
	}

	resp.WriteHeader(200)
	_, _ = resp.Write(marshal)
}

type webLoggerHandler struct {
	logger logger
	next   http.Handler
}

func (w *webLoggerHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	w.next.ServeHTTP(writer, request)
	w.logger.Printf("[%s] %s %s", request.RemoteAddr, request.Method, request.URL)
}
