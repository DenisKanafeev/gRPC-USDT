FROM golang:alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Выносим миграции в корень проекта
RUN mkdir -p /app/migrations && cp -r internal/storage/migrations/* /app/migrations/

WORKDIR /app/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

EXPOSE 50051 2112

CMD ["./main"]
