FROM registry.access.redhat.com/ubi8/ubi-minimal:latest as build
WORKDIR /build

RUN microdnf install go

COPY go.mod .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o sources-monitor-go . && strip sources-monitor-go

# We actually don't need a distro (theres no shell, but we wouldn't be able to do anything anyway)
FROM gcr.io/distroless/static:nonroot
COPY --from=build /build/sources-monitor-go /sources-monitor-go

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-monitor-go"]
