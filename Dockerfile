# BUILD STAGE
FROM golang:1.21.4-alpine3.18 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go

## Download migrate
RUN apk add curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz

# RUN STAGE
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrate ./migrate
COPY db/migrations ./migrations
COPY app.env .
COPY start.sh .
COPY wait-for.sh .

EXPOSE 8080