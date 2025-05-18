FROM golang:1.24.3 AS build
WORKDIR /app
ARG appVersion=""
# Copy the source code.
COPY . .
# Installs Go dependencies
RUN go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'github.com/jkaninda/s3safe/utils.Version=${appVersion}'" -o /app/s3safe


FROM alpine:3.21.3
ENV TZ=UTC
ARG appVersion=""
ENV VERSION=${appVersion}
LABEL org.opencontainers.image.title="s3safe"
LABEL org.opencontainers.image.description="A simple and secure backup tool for S3 storage"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.authors="Jonas Kaninda <me@jonaskaninda.dev>"
LABEL org.opencontainers.image.version=${appVersion}
LABEL org.opencontainers.image.source="github.com/jkaninda/s3safe"

RUN apk --update add --no-cache tzdata ca-certificates
COPY --from=build /app/s3safe /usr/local/bin/s3safe
RUN chmod +x /usr/local/bin/s3safe

ENTRYPOINT ["/usr/local/bin/s3safe"]
