FROM golang:1.22-bookworm AS go

ARG BAZELISK_VERSION=1.25.0
RUN CGO_ENABLED=0 GOOS=linux GOBIN=/opt/bazelisk/bin go install github.com/bazelbuild/bazelisk@v${BAZELISK_VERSION}

FROM debian:bookworm-slim

RUN apt-get update \
 && apt-get -y --no-install-recommends install \
      ca-certificates \
      git \
      python3 \
      python3-dev \
      python3-pip \
      python3-ply \
      python3-venv \
      build-essential \
 && rm -rf /var/lib/apt/lists/*

# Create unprivileged user
RUN groupadd -g 1002 ci
RUN useradd --create-home -g 1002 -u 1001 -ms /bin/bash ci
WORKDIR /home/ci
USER ci

# Add pre-commit and local binaries to PATH
ENV PATH=/home/ci/python-venv/bin:/home/ci/.local/bin:/usr/local/go/bin:$PATH

RUN python3 -m venv /home/ci/python-venv \
 && python3 -m pip install --no-cache-dir --upgrade pip==23.0 \
 && python3 -m pip install --no-cache-dir pre-commit==3.0.2

COPY --from=go /usr/local/go /usr/local/go
COPY --from=go /opt/bazelisk/bin/bazelisk /home/ci/.local/bin/bazelisk

ENV BAZELISK_HOME=/home/ci/.cache/bazelisk

# Pre-download some bazel version to save time in CI
RUN USE_BAZEL_VERSION=latest bazelisk --version
