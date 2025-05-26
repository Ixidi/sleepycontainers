package internal

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	ProxyTargetHeader = "X-SleepyContainers-Target"
)

type HttpServer struct {
	Port                 int
	Proxy                *Proxy
	ServiceNameExtractor ServiceNameExtractor

	server        *http.Server
	accessTimeMap sync.Map
	templates     *Templates
}

func (p *HttpServer) Start() error {
	templates, err := LoadTemplates()
	if err != nil {
		logrus.WithField("err", err).Error("Error loading templates")
		return err
	}

	p.templates = templates

	p.server = &http.Server{
		Addr:    ":" + strconv.Itoa(p.Port),
		Handler: http.HandlerFunc(p.handleRequest),
	}

	logrus.Infof("Starting proxy server at http://127.0.0.1:%d", p.Port)
	return p.server.ListenAndServe()
}

func (p *HttpServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		logrus.WithFields(logrus.Fields{
			"method":   r.Method,
			"url":      r.URL.String(),
			"duration": duration,
			"remote":   r.RemoteAddr,
		}).Debug("Handled request")
	}()

	serviceName, err := p.ServiceNameExtractor.Extract(r)
	if serviceName == "" {
		p.handleError(w, fmt.Errorf("missing service name label in request"))
		return
	}

	err = p.Proxy.Handle(serviceName, w, r)
	if err == nil {
		return
	}

	var wrongServiceStatusError WrongServiceStatusError
	if errors.As(err, &wrongServiceStatusError) {
		switch wrongServiceStatusError.Status {
		case ServiceStatusLoading:
			err := p.templates.WriteLoadingTemplate(w, wrongServiceStatusError.GroupName)
			if err != nil {
				logrus.WithField("err", err).Error("Error writing loading template")
				http.Error(w, "Service is loading...", http.StatusOK)
			}
		case ServiceStatusShuttingDown:
			err := p.templates.WriteShutdownTemplate(w, wrongServiceStatusError.GroupName)
			if err != nil {
				logrus.WithField("err", err).Error("Error writing shutdown template")
				http.Error(w, "Service is being shut down...", http.StatusOK)
			}
		default:
			p.handleError(w, fmt.Errorf("service %s is in status %d", wrongServiceStatusError.ServiceName, wrongServiceStatusError.Status))
		}
		return
	}

	p.handleError(w, fmt.Errorf("error handling request for service %s: %w", serviceName, err))
}

func (p *HttpServer) handleError(w http.ResponseWriter, err error) {
	logrus.WithField("err", err).Warn("Error handling request")
	if p.templates.WriteProblemTemplate(w, err) != nil {
		logrus.WithField("err", err).Error("Error writing problem template")
		http.Error(w, "Unexpected error has occurred. Please contact site administrator.", http.StatusInternalServerError)
	}
}
