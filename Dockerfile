FROM --platform=${BUILDPLATFORM} golang:1.25.5 AS build

ARG TARGETARCH BUILDPLATFORM

WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN dpkg --add-architecture arm64 && \
	apt-get update && \
	apt-get install -y \
		gcc-aarch64-linux-gnu \
		libsqlite3-dev:arm64

# Install magika CLI
RUN curl -LsSf https://github.com/google/magika/releases/latest/download/magika-installer.sh | sh

RUN if [ "${TARGETARCH}" = "amd64" ]; then \
  go build -a -o 'snips.sh'; \
else \
  CC=aarch64-linux-gnu-gcc GOARCH=${TARGETARCH} CGO_ENABLED=1 go build -ldflags "-linkmode external -extldflags -static" -a -o 'snips.sh'; \
fi

FROM --platform=${BUILDPLATFORM} ubuntu:22.04

COPY --from=build /root/.local/bin/magika /usr/local/bin/magika
COPY --from=build /build/snips.sh /usr/bin/snips.sh

ENV SNIPS_HTTP_INTERNAL=http://0.0.0.0:8080
ENV SNIPS_SSH_INTERNAL=ssh://0.0.0.0:2222

EXPOSE 8080 2222

ENTRYPOINT [ "/usr/bin/snips.sh" ]
