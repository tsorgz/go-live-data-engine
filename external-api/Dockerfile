FROM golang:1.21-alpine

WORKDIR /app
COPY go.mod .
COPY . .

RUN go mod download
RUN go build -o main .

EXPOSE 8081

CMD ["./main"]