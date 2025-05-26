FROM golang:1.23

EXPOSE 8080

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY templates ./templates

RUN go build -o sleepycontainers sleepycontainers/cmd/sleepycontainers

ENTRYPOINT ["./sleepycontainers"]