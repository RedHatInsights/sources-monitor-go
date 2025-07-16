FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS build

WORKDIR /build

RUN microdnf install --assumeyes go

ARG GOARCH=amd64
ARG GOOS=linux

COPY . .

RUN go mod download \
    && go build -o sources-monitor-go . \
    && strip sources-monitor-go

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

COPY --from=build /build/sources-monitor-go /sources-monitor-go
COPY licenses/LICENSE /licenses/LICENSE

ENTRYPOINT ["/sources-monitor-go"]
