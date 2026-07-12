FROM golang:1.26.0 AS build

ARG TARGETARCH

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN apt-get update && apt-get install -y --no-install-recommends \
    jq just \
    gcc-aarch64-linux-gnu \
    && rm -rf /var/lib/apt/lists/*

ENV VENDOR_DIR=/opt
ENV TARGETOS=linux
RUN just vendor-onnxruntime

ENV ONNX_DIR=/opt/onnxruntime
ENV OUTPUT=/build/snips.sh
RUN if [ "${TARGETARCH}" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc; \
    fi && \
    just build

FROM gcr.io/distroless/cc-debian12

COPY --from=build /opt/onnxruntime/lib /opt/onnxruntime/lib
COPY --from=build /build/snips.sh /usr/bin/snips.sh

ENV LD_LIBRARY_PATH=/opt/onnxruntime/lib
ENV SNIPS_HTTP_INTERNAL=http://0.0.0.0:8080
ENV SNIPS_SSH_INTERNAL=ssh://0.0.0.0:2222

EXPOSE 8080 2222

ENTRYPOINT ["/usr/bin/snips.sh"]
