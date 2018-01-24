FROM golang:alpine

RUN apk add --update ca-certificates

ADD . /go/src/github.com/Jimdo/wonderland-ecs-metadata
WORKDIR /go/src/github.com/Jimdo/wonderland-ecs-metadata

RUN set -ex \
 && apk add --update git \
 && go install -ldflags "-X github.com/Jimdo/wonderland-ecs-metadata/config/config.ProgramVersion=$(git describe --tags || git rev-parse --short HEAD || echo dev)" ./...\
 && apk del --purge git

EXPOSE 9000
