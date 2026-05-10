FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o hndigest .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/hndigest .
COPY sw.js .
COPY static/ static/
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

VOLUME ["/app/output", "/app/data"]

EXPOSE 8000

ENTRYPOINT ["/entrypoint.sh"]
