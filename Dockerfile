FROM registry.access.redhat.com/ubi9/go-toolset:latest as build

USER root

WORKDIR /build

ARG GOARCH=amd64
ARG GOOS=linux

COPY . .

RUN go mod download \
    && go build -o sources-monitor-go . \
    && strip sources-monitor-go

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

COPY --from=build /build/sources-monitor-go /sources-monitor-go
COPY licenses/LICENSE /licenses/LICENSE

RUN chmod +x /sources-monitor-go

USER 1001

ENTRYPOINT ["/sources-monitor-go"]
