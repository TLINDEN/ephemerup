# syntax=docker/dockerfile:1 -*-python-*-
FROM golang:1.24-alpine as builder

RUN apk update
RUN apk upgrade
RUN apk add --no-cache git make bash binutils

RUN git --version

WORKDIR /work

COPY go.mod .
COPY . .
RUN go mod download
RUN make && strip ephemerupd

FROM alpine:3.22
LABEL maintainer="Uploads Author <info@daemon.de>"

RUN install -o 1001 -g 1001 -d /data

WORKDIR /app
COPY --from=builder /work/ephemerupd /app/ephemerupd

ENV EPHEMERUPD_LISTEN=:8080
ENV EPHEMERUPD_STORAGEDIR=/data
ENV EPHEMERUPD_DBFILE=/data/bbolt.db
ENV EPHEMERUPD_DEBUG=1

USER 1001:1001
EXPOSE 8080
VOLUME /data
CMD ["/app/ephemerupd"]
