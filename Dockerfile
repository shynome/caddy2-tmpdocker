FROM golang:1.13-alpine as Build
COPY . /caddy2-tmpdocker
WORKDIR /caddy2-tmpdocker/cmd/caddy
RUN go build -mod=vendor -o /usr/local/bin/caddy2

FROM alpine
COPY --from=Build /usr/local/bin/caddy2 /usr/local/bin/caddy2
ENTRYPOINT [ "/usr/local/bin/caddy2" ]
