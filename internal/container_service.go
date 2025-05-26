package internal

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/url"
	"sync"
	"time"
)

type ServiceStatus int

const (
	ServiceStatusLoading      ServiceStatus = iota
	ServiceStatusShuttingDown ServiceStatus = iota
	ServiceStatusRunning      ServiceStatus = iota
)

type ServiceContainer struct {
	Container *Container
	Status    ServiceStatus
}

func (serviceContainer *ServiceContainer) AccessURL() (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://127.0.0.1:%d", serviceContainer.Container.AccessiblePort))
}

type ContainerService struct {
	DockerClient      *DockerClient
	InactivityTimeout time.Duration

	accessTimeMap sync.Map // map[string]time.Time
	shuttingDown  sync.Map // map[string]bool
}

func NewContainerService(dockerClient *DockerClient, inactivityTimeout time.Duration) (*ContainerService, error) {
	service := &ContainerService{
		DockerClient:      dockerClient,
		InactivityTimeout: inactivityTimeout,
	}

	allLabels, err := service.DockerClient.GetAllUniqueLabels()
	if err != nil {
		logrus.WithField("err", err).Error("Error getting all labels")
		return nil, fmt.Errorf("error getting all labels: %w", err)
	}

	for _, label := range allLabels {
		service.accessTimeMap.Store(label, time.Now())
	}

	return service, nil
}

func (s *ContainerService) GetServiceContainer(serviceName string) (*ServiceContainer, error) {
	container, err := s.DockerClient.GetContainerByServiceName(serviceName)
	if err != nil {
		return nil, fmt.Errorf("container for service %s not found: %w", serviceName, err)
	}

	if container.AccessiblePort == -1 {
		return nil, fmt.Errorf("service %s not accessible at any port", serviceName)
	}

	containerGroup, err := s.DockerClient.GetContainerGroupByLabel(container.GroupName)
	if err != nil {
		return nil, fmt.Errorf("container group %s not found: %w", container.GroupName, err)
	}

	for _, container := range containerGroup.Containers {
		if s.DockerClient.IsContainerStopping(container.ID) {
			return &ServiceContainer{
				Container: container,
				Status:    ServiceStatusShuttingDown,
			}, nil
		}
	}

	if !containerGroup.IsAllRunning() {
		go s.startContainers(containerGroup)
		return &ServiceContainer{
			Container: container,
			Status:    ServiceStatusLoading,
		}, nil
	}

	return &ServiceContainer{
		Container: container,
		Status:    ServiceStatusRunning,
	}, nil
}

func (s *ContainerService) NotifyAccess(groupName string) {
	s.accessTimeMap.Store(groupName, time.Now())
}

func (s *ContainerService) InactiveContainersCleanerLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.accessTimeMap.Range(func(key, value interface{}) bool {
				lastAccessTime := value.(time.Time)
				if time.Since(lastAccessTime) > s.InactivityTimeout {
					label := key.(string)
					containerGroup, err := s.DockerClient.GetContainerGroupByLabel(label)
					if err != nil {
						return true
					}
					s.stopContainers(containerGroup)
					s.accessTimeMap.Delete(key)
				}
				return true
			})
		}
	}
}

func (s *ContainerService) startContainers(group *ContainerGroup) {
	for _, container := range group.GetContainersByHighestPriority() {
		if !container.IsRunning && !s.DockerClient.IsContainerStarting(container.ID) && !s.DockerClient.IsContainerStopping(container.ID) {
			logrus.WithField("name", container.ContainerName).WithField("group", container.GroupName).Infof("Starting container")
			err := s.DockerClient.StartContainer(container.ID)
			if err != nil {
				logrus.WithField("name", container.ContainerName).WithField("group", container.GroupName).WithField("err", err).Errorf("Error starting container")
			}
			s.accessTimeMap.Store(group.Name, time.Now())
		}
	}
}

func (s *ContainerService) stopContainers(group *ContainerGroup) {
	for _, container := range group.GetContainersByLowestPriority() {
		if container.IsRunning && !s.DockerClient.IsContainerStarting(container.ID) && !s.DockerClient.IsContainerStopping(container.ID) {
			s.shuttingDown.Store(group.Name, true)
			logrus.WithField("name", container.ContainerName).WithField("group", container.GroupName).Infof("Stopping containers due to inactivity")
			err := s.DockerClient.StopContainer(container.ID)
			s.shuttingDown.Delete(group.Name)
			if err != nil {
				logrus.WithField("name", container.ContainerName).WithField("group", container.GroupName).WithField("err", err).Errorf("Error stoping container")
			}
		}
	}
}
