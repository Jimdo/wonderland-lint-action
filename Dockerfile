FROM golang:alpine

RUN apk --update add ca-certificates

ADD . /go/src/github.com/Jimdo/wonderland-cron
WORKDIR /go/src/github.com/Jimdo/wonderland-cron

RUN go install -v

ENTRYPOINT ["wonderland-cron"]

EXPOSE 8000
