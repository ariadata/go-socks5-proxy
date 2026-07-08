FROM golang:1.25-alpine AS builder

# Install required system packages and update certificates
RUN apk update && \
    apk upgrade && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates

# Add Maintainer Info
LABEL maintainer="AriaData <info@ariadata.co>"
LABEL description="SOCKS5 Proxy Server in Go."

# Set the Current Working Directory inside the container
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum main.go ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o socks5-server .

######## Start a new stage from scratch #######
#FROM scratch
FROM gcr.io/distroless/static-debian11

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /build/socks5-server .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose port 1080 to the outside
EXPOSE 1080

# Command to run the executable
CMD ["./socks5-server"]