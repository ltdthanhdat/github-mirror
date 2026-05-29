# Use the official Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.26.3 AS builder

# Create and set the working directory inside the container.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container to leverage cached dependencies.
# To avoid redownloading dependencies on every code change, we first copy the go.mod and go.sum.
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire application source code.
COPY . .

# Build the application.
# The output binary will be named 'mirror' and placed in /app.
# We disable CGO to produce a static binary.
# We also set the build tags to exclude any development-only code if needed.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o mirror ./cmd/mirror

# Use a small Alpine image so the worker has git and CA certificates at runtime.
FROM alpine:3.22

RUN apk add --no-cache ca-certificates git

# Copy the binary and runtime templates from the builder stage.
COPY --from=builder /app/mirror /app/mirror
COPY --from=builder /app/internal/ui/templates /app/internal/ui/templates

# Expose the port the application runs on.
# We'll use 8080 as the default, but it can be overridden by environment variable.
EXPOSE 8080

# Command to run the executable.
ENTRYPOINT ["/app/mirror"]
