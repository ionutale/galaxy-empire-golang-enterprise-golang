FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /server .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -u 1001 appuser
COPY --from=builder /server /server
EXPOSE 8080
USER appuser
CMD ["/server"]
