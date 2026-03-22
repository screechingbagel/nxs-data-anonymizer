FROM golang:1.26-bullseye AS build-env

ARG APP_VERSION

WORKDIR /usr/src/app
ADD ./ /usr/src/app

RUN go build -ldflags "-X github.com/screechingbagel/nxs-data-anonymizer/ctx.version=$APP_VERSION" -o /nxs-data-anonymizer

FROM alpine

RUN apk update --no-cache && apk add --no-cache tar postgresql-client mysql-client libc6-compat s3cmd

COPY --from=build-env /nxs-data-anonymizer /
