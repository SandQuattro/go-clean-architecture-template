## ! FOR LOCAL TESTS ONLY !
# Start from a small, secure base image
FROM golang:1.23.2-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download the Go module dependencies
RUN apk add --no-cache git && \
    go mod download

# Copy the source code into the container
COPY . .

# Build the Go binary
RUN VERSION=$(git describe --tags --always --dirty) && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags "-X clean-arch-template/version.Version=$VERSION" \
    -a -installsuffix cgo -o app ./cmd/template/main.go

# Create a minimal production image
FROM alpine:latest

# It's essential to regularly update the packages within the image to include security patches
# Reduce image size
RUN apk update && \
    apk upgrade && \
    apk add jq && \
    apk add --no-cache ca-certificates && \
    rm -rf /var/cache/apk/* && \
    rm -rf /tmp/*

# Avoid running code as a root user
RUN adduser -D appuser
USER appuser

# Set the working directory inside the container
WORKDIR /app

# Copy only the necessary files from the builder stage
COPY --from=builder /app/app .

RUN mkdir config && mkdir migrations
COPY --from=builder /app/config/config.json ./config/
COPY --from=builder /app/migrations/* ./migrations/

# Set any environment variables required by the application

# Expose the port that the application listens on
EXPOSE 8000

# Run the binary when the container starts
CMD ["./app"]
