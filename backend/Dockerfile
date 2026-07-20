FROM golang:1.26

RUN apt-get update && apt-get install -y nodejs npm

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 3000

CMD ["air"]
