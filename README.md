# SleepyContainers

Allows to stop unused containers, and start them again when needed.
Application acts as a reverse proxy, so you can access your containers via the same URL as before.

## How to run

```yaml
services:
  sleepycontainers:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      SLEEPYCONTAINERS_SERVICE_NAME_EXTRACTOR: "query"
      SLEEPYCONTAINERS_PORT: 12345
      SLEEPYCONTAINERS_TIMEOUT: 1h
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    network_mode: "host"
```

Docker sock is mounted to allow the application to manage Docker containers.

Network mode is set to "host" to allow the application to listen on the same port as the containers it manages.

## How to mark containers as sleepy

```yaml
services:
  test-service:
    image: strm/helloworld-http
    labels:
      me.zylinski.sleepycontainers.group_name: "a"
      me.zylinski.sleepycontainers.accessible_at_port: "8080"
      me.zylinski.sleepycontainers.service_name: "test-service"
      me.zylinski.sleepycontainers.priority: "1"
    ports:
      - "8080:80"
  test-service2:
    image: strm/helloworld-http
    labels:
      me.zylinski.sleepycontainers.group_name: "a"
      me.zylinski.sleepycontainers.accessible_at_port: "8080"
      me.zylinski.sleepycontainers.service_name: "test-service2"
      me.zylinski.sleepycontainers.priority: "2"
    ports:
      - "8085:80"
  test-service3:
    image: strm/helloworld-http
    labels:
      me.zylinski.sleepycontainers.group_name: "a"
      me.zylinski.sleepycontainers.priority: "3"
    ports:
      - "8086:80"
```

Labels are used to mark containers as sleepy. You can specify the group name, accessible port, and service name.

Group name is used to group containers together, so you can stop and start them as a group.

Accessible port is used to specify the port on which the container is accessible when it is running.

Service name is used to specify the name of the service, which will be used in the HTTP request to access the container.

Priority is used to determine the order in which containers are started when they are needed. Lower number means higher priority.