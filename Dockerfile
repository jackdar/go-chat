# Stage 1: Builder
FROM golang:1.24 AS builder

WORKDIR /app

# COPY go.mod go.sum ./
# RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server/main.go

# Stage 2: Runner
FROM alpine:latest

# Install CA certificates for HTTPS communication if needed
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main .

# Expose the port your application listens on
EXPOSE 8080

# Command to run the application
CMD ["./main"]
