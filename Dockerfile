FROM golang:1.25-alpine AS builder

ARG GIT_BRANCH
ARG GITHUB_SHA
ARG CI

ENV CGO_ENABLED=0

WORKDIR /build
COPY . .

RUN go build -o stash -ldflags "-X main.revision=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%d-%H:%M:%S)" ./app

FROM alpine:3.21

COPY --from=builder /build/stash /srv/stash

RUN apk add --no-cache tzdata

WORKDIR /srv
ENTRYPOINT ["/srv/stash"]
