package internal

import (
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"strconv"
	"sync"
)

const (
	ContainerLabelGroupName    = "me.zylinski.sleepycontainers.group_name"
	ContainerLabelAccessibleAt = "me.zylinski.sleepycontainers.accessible_at_port"
	ContainerLabelServiceName  = "me.zylinski.sleepycontainers.service_name"
)

type Container struct {
	ID             string
	GroupName      string
	ContainerName  string
	ServiceName    string
	AccessiblePort int
	IsRunning      bool
}

type ContainerGroup struct {
	Name       string
	Containers []*Container
}

func (c *ContainerGroup) IsAllRunning() bool {
	for _, container := range c.Containers {
		if !container.IsRunning {
			return false
		}
	}
	return true
}

type DockerClient struct {
	client             *docker.Client
	startingContainers sync.Map
	stoppingContainers sync.Map
}

func NewDockerClient(client *docker.Client) *DockerClient {
	return &DockerClient{
		client: client,
	}
}

func (c *DockerClient) parseContainer(container *docker.APIContainers) (*Container, error) {
	accessibleAt := ""
	serviceName := ""
	for key, value := range container.Labels {
		if key == ContainerLabelAccessibleAt {
			accessibleAt = value
		} else if key == ContainerLabelServiceName {
			serviceName = value
		}
	}

	port := -1
	if accessibleAt != "" {
		var err error
		port, err = strconv.Atoi(accessibleAt)
		if err != nil {
			return nil, fmt.Errorf("invalid gateway port %s: %w", accessibleAt, err)
		}

		if serviceName == "" {
			return nil, fmt.Errorf("service name is required when accessible_at is set")
		}
	}

	return &Container{
		ID:             container.ID,
		GroupName:      container.Labels[ContainerLabelGroupName],
		ServiceName:    serviceName,
		AccessiblePort: port,
		ContainerName:  container.Names[0],
		IsRunning:      container.State == "running",
	}, nil
}

func (c *DockerClient) GetContainerByServiceName(serviceName string) (*Container, error) {
	containers, err := c.client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		name, ok := container.Labels[ContainerLabelServiceName]
		if !ok || name != serviceName {
			continue
		}

		return c.parseContainer(&container)
	}

	return nil, fmt.Errorf("no containers found with service name")
}

func (c *DockerClient) GetContainerGroupByLabel(label string) (*ContainerGroup, error) {
	containers, err := c.client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		panic(err)
	}

	var matchingContainers []*Container
	for _, container := range containers {
		matches := false
		for key, value := range container.Labels {
			if key == ContainerLabelGroupName && value == label {
				matches = true
				break
			}
		}

		if !matches {
			continue
		}

		parsedContainer, err := c.parseContainer(&container)
		if err != nil {
			return nil, err
		}

		matchingContainers = append(matchingContainers, parsedContainer)
	}

	if len(matchingContainers) == 0 {
		return nil, fmt.Errorf("no containers found with label %s", label)
	}

	return &ContainerGroup{
		Name:       label,
		Containers: matchingContainers,
	}, nil
}

func (c *DockerClient) IsContainerStarting(containerID string) bool {
	_, exists := c.startingContainers.Load(containerID)
	return exists
}

func (c *DockerClient) IsContainerStopping(containerID string) bool {
	_, exists := c.stoppingContainers.Load(containerID)
	return exists
}

func (c *DockerClient) StartContainer(containerID string) error {
	if c.IsContainerStarting(containerID) {
		return fmt.Errorf("container %s is already starting", containerID)
	}

	c.startingContainers.Store(containerID, struct{}{})
	defer c.startingContainers.Delete(containerID)

	return c.client.StartContainer(containerID, nil)
}

func (c *DockerClient) StopContainer(containerID string) error {
	if c.IsContainerStopping(containerID) {
		return fmt.Errorf("container %s is already stopping", containerID)
	}

	c.stoppingContainers.Store(containerID, struct{}{})
	defer c.stoppingContainers.Delete(containerID)

	return c.client.StopContainer(containerID, 5)
}

func (c *DockerClient) IsContainerRunning(containerID string) (bool, error) {
	container, err := c.client.InspectContainerWithOptions(docker.InspectContainerOptions{
		ID: containerID,
	})
	if err != nil {
		return false, err
	}

	return container.State.Running, nil
}

func (c *DockerClient) GetAllUniqueLabels() ([]string, error) {
	containers, err := c.client.ListContainers(docker.ListContainersOptions{All: true})
	if err != nil {
		return nil, err
	}

	labelSet := make(map[string]struct{})
	for _, container := range containers {
		for key, value := range container.Labels {
			if key == ContainerLabelGroupName {
				labelSet[value] = struct{}{}
			}
		}
	}

	var labels []string
	for label := range labelSet {
		labels = append(labels, label)
	}

	return labels, nil
}
