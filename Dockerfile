FROM golang:1.18-alpine As build
WORKDIR /app
ENV CGO_ENABLED=0
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
RUN go mod download
COPY . /app/
RUN go build -buildvcs=false -o=/out/tg-oauth-proxy

FROM alpine:latest
ENV LISTEN_ADDR=0.0.0.0:80
WORKDIR /opt/tg-oauth-proxy/
COPY --from=build /out/tg-oauth-proxy /opt/tg-oauth-proxy/tg-oauth-proxy
COPY --from=build /app/www /opt/tg-oauth-proxy/www
ENTRYPOINT [ "/opt/tg-oauth-proxy/tg-oauth-proxy" ]
