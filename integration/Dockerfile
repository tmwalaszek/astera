FROM golang:latest

WORKDIR /app

ENV GOPROXY http://192.168.50.251:8080

RUN git clone https://github.com/hashicorp/terraform
WORKDIR /app/terraform

ENTRYPOINT ["sh", "-c", "date; GOPROXY=http://192.168.50.251:8080 go mod tidy; go get -u ./...; date"]

