FROM registry.access.redhat.com/ubi8/ubi:8.4 as build

RUN mkdir /build
WORKDIR /build

RUN dnf -y --disableplugin=subscription-manager install go

COPY go.mod .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o sources-monitor-go . && strip sources-monitor-go

FROM scratch
COPY --from=build /build/sources-monitor-go /sources-monitor-go
ENTRYPOINT ["/sources-monitor-go"]
