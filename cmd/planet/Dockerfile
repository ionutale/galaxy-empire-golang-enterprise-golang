FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /planet .

FROM alpine:3.19
COPY --from=builder /planet /planet
EXPOSE 8082
CMD ["/planet"]
