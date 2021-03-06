FROM golang:1.17.3-alpine as builder

FROM ubuntu:18.04

RUN mkdir /app
# copy the ca-certificate.crt from the build stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY loki2slack /app/loki2slack
ENTRYPOINT [ "/app/loki2slack" ]