# Stage 1: Build the Go application
FROM golang:1.21-alpine3.19 AS builder

MAINTAINER Vicky Phang <vickyphang11@gmail.com>

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o cephfs_exporter .

# Stage 2: Create a small image with the built binary
FROM alpine

# Install ceph cli
RUN apk update && apk add ceph18-cephadm ceph18-common

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/cephfs_exporter /cephfs_exporter

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["/cephfs_exporter"]
