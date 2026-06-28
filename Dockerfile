FROM golang:1.26.4-alpine3.24 AS base
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o scoreattack

EXPOSE 8080

ENTRYPOINT ["./scoreattack"]