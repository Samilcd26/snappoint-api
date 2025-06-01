# Development stage
FROM golang:1.24.0-alpine

WORKDIR /app

# Install air and required tools
RUN go install github.com/air-verse/air@latest

# Copy go.mod and go.sum files
COPY go.* ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Expose port
EXPOSE 8080

# Run with air
CMD ["air", "-c", ".air.toml"]
