FROM golang:1.26

RUN apt-get update && apt-get install -y nodejs npm curl

RUN curl -fsSL https://download.docker.com/linux/static/stable/x86_64/docker-27.1.1.tgz | tar -xz -C /usr/local/bin --strip-components=1 docker/docker

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 3000

CMD ["air"]
