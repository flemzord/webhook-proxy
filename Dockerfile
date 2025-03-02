FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
ENTRYPOINT ["/webhook-proxy"]
EXPOSE 8080
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY webhook-proxy /webhook-proxy
COPY config.example.yaml /config.yaml