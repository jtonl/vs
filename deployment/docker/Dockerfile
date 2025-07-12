FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY vs.go .
COPY go.mod .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vs vs.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget
WORKDIR /root/

COPY --from=builder /app/vs .

RUN addgroup -g 1001 -S appuser && \
    adduser -S -D -H -u 1001 -h /root -s /sbin/nologin -G appuser appuser

USER appuser

EXPOSE 32767

CMD ["./vs"]