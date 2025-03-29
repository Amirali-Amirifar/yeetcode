FROM golang:1.24

LABEL authors="amirali"

WORKDIR /app

COPY go.mod  ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/backend ./cmd

EXPOSE 8080


ENTRYPOINT ["./backend"]