# Frontend build stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./
RUN npm ci

# Copy frontend source and build
COPY web/ ./
RUN npm run build

# Backend build stage
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=backend-builder /app/server /app/server

# Copy frontend build from frontend-builder
COPY --from=frontend-builder /app/web/dist /app/web/dist

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

CMD ["/app/server"]
