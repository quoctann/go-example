# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

WORKDIR /app
# Copy go mod and sum files
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download
# Copy source code
COPY . .
# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /root/
# Copy the binary from the builder stage
COPY --from=builder /app/main .
# Copy the config file (optional, but good for default)
COPY --from=builder /app/config/config.yaml .

# Expose port
EXPOSE 8080

# Command to run the application
CMD ["./main"]