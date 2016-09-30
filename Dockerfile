FROM golang:alpine

RUN apk --update add ca-certificates

ADD . /go/src/github.com/Jimdo/wonderland-crons
WORKDIR /go/src/github.com/Jimdo/wonderland-crons

RUN go install -v

ENTRYPOINT ["wonderland-crons"]

EXPOSE 8000
