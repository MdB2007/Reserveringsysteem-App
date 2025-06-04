# Dockerfile
FROM golang:1.24.3-alpine
WORKDIR /app
RUN apk add --no-cache git
COPY . .
RUN go build -o main .
EXPOSE 80
CMD ["./main"]