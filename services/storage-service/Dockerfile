# Stage 1: Build the application
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the application
# CGO_ENABLED=0 for a static binary, GOOS=linux for Linux compatibility
# -ldflags "-X main.Version=$(cat VERSION)" to inject version if you have a VERSION file
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /storage-service ./cmd/main.go

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /storage-service .

# Copy the configuration file
# Ensure the configs directory and config.yaml exist in your project root relative to Dockerfile
COPY configs/ ./configs/

# Expose the port the application runs on (should match config.yaml and main.go default)
EXPOSE 8082

# Command to run the application
CMD ["./storage-service"] 