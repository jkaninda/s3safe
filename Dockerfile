FROM golang:1.24.3 AS build
WORKDIR /app
ARG appVersion=""
# Copy the source code.
COPY . .
# Installs Go dependencies
RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'github.com/jkaninda/s3safe/utils.Version=${appVersion}'" -o /app/s3safe


FROM alpine:3.22.0
ENV TZ=UTC
ARG appVersion=""
ENV VERSION=${appVersion}
LABEL org.opencontainers.image.title="s3safe"
LABEL org.opencontainers.image.description="S3Safe is a lightweight CLI tool for securely backing up and restoring data from Amazon S3 and S3-compatible storage"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.authors="Jonas Kaninda <me@jkaninda.dev>"
LABEL org.opencontainers.image.version=${appVersion}
LABEL org.opencontainers.image.source="https://github.com/jkaninda/s3safe"

RUN apk --update add --no-cache tzdata ca-certificates
COPY --from=build /app/s3safe /usr/local/bin/s3safe
RUN chmod +x /usr/local/bin/s3safe

ENTRYPOINT ["/usr/local/bin/s3safe"]
