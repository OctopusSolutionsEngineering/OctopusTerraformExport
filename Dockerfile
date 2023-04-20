# syntax=docker/dockerfile:1

FROM golang:1.20

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY . /app

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /octoterra cmd/octoterra.go

# Run
CMD ["/octoterra"]
