FROM golang:1.21

RUN go install github.com/jstemmer/go-junit-report@latest

ENV CGO_ENABLED=0

WORKDIR /go/src/github.com/cyverse-de/qms
COPY . .
RUN make

FROM debian:stable-slim

WORKDIR /app

COPY --from=0 /go/src/github.com/cyverse-de/qms/qms /bin/qms
COPY --from=0 /go/src/github.com/cyverse-de/qms/swagger.json swagger.json
COPY --from=0 /go/src/github.com/cyverse-de/qms/migrations migrations

ENTRYPOINT ["qms"]

EXPOSE 8080
