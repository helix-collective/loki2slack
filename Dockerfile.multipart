FROM golang:1.17.3-alpine as builder

COPY . /src
RUN cd /src; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o loki2slack

FROM ubuntu:18.04

RUN mkdir /app
# copy the ca-certificate.crt from the build stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/loki2slack /app/loki2slack
ENTRYPOINT [ "/app/loki2slack" ]