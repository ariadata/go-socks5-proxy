FROM golang:1.26-alpine AS builder

RUN apk update && \
    apk upgrade && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates

LABEL maintainer="AriaData <info@ariadata.co>"

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o socks5-server .

FROM scratch

COPY --from=builder /build/socks5-server /socks5-server

EXPOSE 1080

CMD ["/socks5-server"]
