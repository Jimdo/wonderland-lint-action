FROM golang:1.9.1-alpine

RUN apk --update add ca-certificates

ADD . /go/src/github.com/Jimdo/wonderland-crons
WORKDIR /go/src/github.com/Jimdo/wonderland-crons

RUN set -ex \
    && apk add --update git \
    && go install -v -ldflags "-X main.programVersion=$(git describe --tags || git rev-parse --short HEAD || echo dev)" \
    && apk del --purge git

ENTRYPOINT ["wonderland-crons"]

EXPOSE 8000
