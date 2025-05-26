package internal

import (
	"net/http"
	"strings"
)

type ServiceNameExtractor interface {
	Extract(r *http.Request) (string, error)
}

type QueryServiceNameExtractor struct {
	Param string
}

func (e *QueryServiceNameExtractor) Extract(r *http.Request) (string, error) {
	serviceName := r.URL.Query().Get(e.Param)
	serviceName = strings.TrimSuffix(serviceName, "/")
	if serviceName == "" {
		return "", nil
	}
	return serviceName, nil
}

type HeaderServiceNameExtractor struct {
	Header string
}

func (e *HeaderServiceNameExtractor) Extract(r *http.Request) (string, error) {
	serviceName := r.Header.Get(e.Header)
	if serviceName == "" {
		return "", nil
	}
	return serviceName, nil
}

type PathServiceNameExtractor struct {
}

func (e *PathServiceNameExtractor) Extract(r *http.Request) (string, error) {
	serviceName := strings.TrimPrefix(r.URL.Path, "/")
	serviceName = strings.TrimSuffix(serviceName, "/")
	if serviceName == "" {
		return "", nil
	}
	return serviceName, nil
}

type SubdomainServiceNameExtractor struct {
}

func (e *SubdomainServiceNameExtractor) Extract(r *http.Request) (string, error) {
	host := r.Host
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return "", nil
	}
	serviceName := parts[0]
	serviceName = strings.TrimSuffix(serviceName, "/")
	if serviceName == "" {
		return "", nil
	}
	return serviceName, nil
}
