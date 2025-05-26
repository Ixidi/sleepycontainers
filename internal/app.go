package internal

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"time"
)

func init() {
	if os.Getenv("SLEEPYCONTAINERS_DEBUG") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	logrus.SetOutput(os.Stdout)
}

type SleepyContainers struct {
}

func (s *SleepyContainers) Run() {
	port := os.Getenv("SLEEPYCONTAINERS_PORT")
	if port == "" {
		logrus.Fatal("SLEEPYCONTAINERS_PORT environment variable is not set")
	}
	portInt, err := strconv.Atoi(port)
	if err != nil {
		logrus.Fatalf("SLEEPYCONTAINERS_PORT environment variable is not a valid integer: %v", err)
	}

	timeOut := os.Getenv("SLEEPYCONTAINERS_TIMEOUT")
	if timeOut == "" {
		logrus.Fatal("SLEEPYCONTAINERS_TIMEOUT environment variable is not set")
	}

	timeOutDuration, err := time.ParseDuration(timeOut)
	if err != nil {
		logrus.Fatalf("SLEEPYCONTAINERS_TIMEOUT environment variable is not a valid duration: %v", err)
	}

	rawDockerClient, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		logrus.Fatal("error while creating Docker client: %v", err)
	}

	if err := rawDockerClient.Ping(); err != nil {
		logrus.Fatal("error while connecting to Docker, this most likely means that Docker daemon is not running or you did not mount host Docker socket as a volume: %v", err)
	}

	var serviceNameExtractor ServiceNameExtractor
	switch os.Getenv("SLEEPYCONTAINERS_SERVICE_NAME_EXTRACTOR") {
	case "query":
		serviceNameExtractor = &QueryServiceNameExtractor{Param: "sleepy_container"}
	case "header":
		serviceNameExtractor = &HeaderServiceNameExtractor{Header: "X-SleepyContainers-Target"}
	case "path":
		serviceNameExtractor = &PathServiceNameExtractor{}
	case "subdomain":
		serviceNameExtractor = &SubdomainServiceNameExtractor{}
	default:
		logrus.WithField("error", "SLEEPYCONTAINERS_SERVICE_NAME_EXTRACTOR invalid or not set").Fatal("Error loading environment variable")
	}

	dockerClient := NewDockerClient(rawDockerClient)
	containerService, err := NewContainerService(dockerClient, timeOutDuration)
	if err != nil {
		logrus.WithField("error", err).Fatal("Error creating container service")
	}

	go containerService.InactiveContainersCleanerLoop()

	proxy := NewProxy(containerService)

	server := HttpServer{
		Port:                 portInt,
		Proxy:                proxy,
		ServiceNameExtractor: serviceNameExtractor,
	}

	logrus.Fatal(server.Start())
}
