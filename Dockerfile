################################
# STEP 1 build executable binary
################################
FROM registry.access.redhat.com/hi/go:latest-fips-builder@sha256:2944ca2126a611f87c3ff7a28d567bdd051cafb69a6a1f35e73c6f1f8bf72b9f AS builder

USER 0

WORKDIR /workspace

# Cache deps before copying source so that we do not need to re-download for every build
COPY go.mod go.sum .

# Fetch dependencies hermetically by sourcing the local prefetch cache
RUN if [ -f /cachi2/cachi2.env ]; then . /cachi2/cachi2.env; fi && \
    go mod download

# Now copy the rest of the files for build
COPY *.go .

# Build the binary using the local cached modules
RUN if [ -f /cachi2/cachi2.env ]; then . /cachi2/cachi2.env; fi && \
    go build -ldflags "-w -s" -o sources-monitor-go .

############################
# STEP 2 build a small image
############################
FROM registry.access.redhat.com/hi/go:latest-fips@sha256:b793b36bc6129d842d28f051681deb7907aae8b9b40e894fcd62e0fb2c1eaad5

WORKDIR /

COPY --from=builder /workspace/sources-monitor-go /sources-monitor-go

COPY licenses/LICENSE /licenses/LICENSE

USER 1001

ENTRYPOINT ["/sources-monitor-go"]
