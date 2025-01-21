FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go get -u github.com/gorilla/websocket

RUN go get -u github.com/google/uuid

RUN go get -u github.com/shamaton/msgpack/v2

RUN go build -o main ./main.go

EXPOSE 8888

CMD ["./main"]