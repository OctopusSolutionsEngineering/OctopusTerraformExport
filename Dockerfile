# syntax=docker/dockerfile:1

FROM golang:1.24 AS build

ARG Version=development

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY . /app

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'main.Version=${Version}'" -o /octoterra cmd/cli/octoterra.go

# Create the execution image
FROM alpine:latest

COPY --from=build /octoterra /octoterra

# Run
ENTRYPOINT ["/octoterra"]
