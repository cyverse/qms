FROM golang:1.21

RUN go install github.com/jstemmer/go-junit-report@latest

ENV CGO_ENABLED=0

WORKDIR /go/src/github.com/cyverse-de/QMS
COPY . .
RUN make

FROM debian:stable-slim

WORKDIR /app

COPY --from=0 /go/src/github.com/cyverse-de/QMS/QMS /bin/QMS
COPY --from=0 /go/src/github.com/cyverse-de/QMS/swagger.json swagger.json
COPY --from=0 /go/src/github.com/cyverse-de/QMS/migrations migrations

ENTRYPOINT ["QMS"]

EXPOSE 8080
