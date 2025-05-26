package internal

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"
)

type WrongServiceStatusError struct {
	ServiceName string
	GroupName   string
	Status      ServiceStatus
}

type ReverseProxy struct {
	GroupName string
	Backend   *httputil.ReverseProxy
}

func (w WrongServiceStatusError) Error() string {
	return fmt.Sprintf("service %s is in status %d", w.ServiceName, w.Status)
}

type Proxy struct {
	ContainerService *ContainerService

	reverseProxiesMap sync.Map // map[string]*ReverseProxy
}

func NewProxy(containerService *ContainerService) *Proxy {
	return &Proxy{
		ContainerService:  containerService,
		reverseProxiesMap: sync.Map{},
	}
}

func (p *Proxy) Handle(serviceName string, w http.ResponseWriter, r *http.Request) error {
	reverseProxy, err := p.getReverseProxy(serviceName)
	if err != nil {
		return err
	}

	p.ContainerService.NotifyAccess(reverseProxy.GroupName)
	reverseProxy.Backend.ServeHTTP(w, r)
	return nil
}

func (p *Proxy) getReverseProxy(serviceName string) (*ReverseProxy, error) {
	reverseProxyAny, ok := p.reverseProxiesMap.Load(serviceName)
	reverseProxy, ok := reverseProxyAny.(*ReverseProxy)

	if ok {

		return reverseProxy, nil
	}

	reverseProxy, err := p.createReverseProxy(serviceName)
	if err != nil {
		return nil, err
	}

	p.reverseProxiesMap.Store(serviceName, reverseProxy)

	return reverseProxy, nil
}

func (p *Proxy) createReverseProxy(serviceName string) (*ReverseProxy, error) {
	container, err := p.ContainerService.GetServiceContainer(serviceName)
	if err != nil {
		return nil, err
	}

	if container.Status != ServiceStatusRunning {
		return nil, WrongServiceStatusError{
			ServiceName: serviceName,
			GroupName:   container.Container.GroupName,
			Status:      container.Status,
		}
	}

	accessURL, err := container.AccessURL()
	if err != nil {
		return nil, fmt.Errorf("error getting access URL for service %s: %w", serviceName, err)
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(accessURL)
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		p.reverseProxiesMap.Delete(serviceName)
		http.Redirect(w, r, r.RequestURI, http.StatusFound)
	}
	return &ReverseProxy{
		GroupName: container.Container.GroupName,
		Backend:   reverseProxy,
	}, nil
}
