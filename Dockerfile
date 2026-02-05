# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for dependencies if needed
# RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build static binary
# CGO_ENABLED=0 ensures static linking
RUN CGO_ENABLED=0 GOOS=linux go build -o /atlas ./cmd/atlas

# Final Stage
FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=builder /atlas /atlas

# Expose default port
EXPOSE 8080

# Define volumes for persistence
VOLUME ["/data", "/config"]

# Set default env vars
ENV ATLAS_PORT=8080
ENV ATLAS_DATA_DIR=/data
ENV ATLAS_CONFIG_DIR=/config

ENTRYPOINT ["/atlas"]
CMD ["server"]
