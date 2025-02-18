ARG BASE_UBUNTU
ARG BASE_RUST

### 1: Build vector
FROM $BASE_RUST as build-vector

RUN apt-get update \
    && apt-get install -yq \
      ca-certificates make bash cmake libclang1-9 llvm-9 libsasl2-dev protobuf-compiler software-properties-common

# Install librdkafka-dev >=2.0 because bundled version (1.9.2) has bugs with CA certificates location.
# https://github.com/confluentinc/librdkafka/commit/f8830a28652532009e3f16854cb9d5004d9de06b
RUN apt-get update \
    && apt-get install -yq \
      wget build-essential libsasl2-dev libcurl4-openssl-dev libzstd-dev libssl-dev zlib1g-dev libc6-dev unzip \
    && wget https://github.com/edenhill/librdkafka/archive/v2.0.2.zip \
    && unzip v2.0.2.zip \
    && cd librdkafka-2.0.2 \
    && ./configure \
    && make \
    && make install

WORKDIR /vector
RUN git clone --depth 1 --branch v0.27.0 https://github.com/vectordotdev/vector.git \
    && cd vector

# Download and cache dependencies
WORKDIR /vector/vector
RUN cargo fetch

RUN cargo build \
    --release \
    -j $(($(nproc) /2)) \
    --offline \
    --no-default-features \
    --features "api,api-client,enrichment-tables,sources-host_metrics,sources-internal_metrics,sources-file,sources-kubernetes_logs,transforms,sinks-prometheus,sinks-blackhole,sinks-elasticsearch,sinks-file,sinks-loki,sinks-socket,sinks-console,sinks-vector,sinks-kafka,sinks-splunk_hec,unix,rdkafka?/dynamic-linking,rdkafka?/gssapi-vendored,vrl-cli" \
    && strip target/release/vector


### 2: Final image
FROM $BASE_UBUNTU
RUN mkdir -p /etc/vector \
    && apt-get update \
    && apt-get install -yq ca-certificates tzdata inotify-tools gettext procps wget \
    && rm -rf /var/cache/apt/archives/*

# libssl.1
RUN wget http://nz2.archive.ubuntu.com/ubuntu/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2.16_amd64.deb \
    && dpkg -i libssl1.1_1.1.1f-1ubuntu2.16_amd64.deb
# libcurl.4
RUN apt-get install -yq libcurl4

COPY --from=build-vector /vector/vector/target/release/vector /usr/bin/vector

COPY --from=build-vector /usr/local/lib/librdkafka.* /usr/local/lib/
COPY --from=build-vector /usr/local/lib/pkgconfig/rdkafka.pc /usr/local/lib/pkgconfig/rdkafka.pc

ENV LD_LIBRARY_PATH=/usr/local/lib

COPY reloader /usr/bin/reloader
ENTRYPOINT ["/usr/bin/vector"]
