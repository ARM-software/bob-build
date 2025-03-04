FROM golang:1.22-bullseye as go

ARG BAZELISK_VERSION=1.25.0
RUN CGO_ENABLED=0 GOOS=linux GOBIN=/opt/bazelisk/bin go install github.com/bazelbuild/bazelisk@v${BAZELISK_VERSION}

FROM debian:bullseye-slim

RUN apt update \
 && apt -y --no-install-recommends install \
      ca-certificates \
      git \
      python3 \
      python3-dev \
      python3-pip \
      python3-ply \
      build-essential \
 && rm -rf /var/lib/apt/lists/*

# Create unprivileged user
RUN groupadd -g 1002 ci
RUN useradd --create-home -g 1002 -u 1001 -ms /bin/bash ci
WORKDIR /home/ci
USER ci

# Add local binaries from `pip install --user` to PATH
ENV PATH /home/ci/.local/bin:/usr/local/go/bin:$PATH

# Upgrade `pip`
RUN python3 -m pip install --user --no-cache-dir --upgrade pip==23.0
RUN python3 -m pip install --user --no-cache-dir pre-commit==3.0.2

COPY --from=go /usr/local/go /usr/local/go
COPY --from=go /opt/bazelisk/bin/bazelisk /home/ci/.local/bin/bazelisk

ENV BAZELISK_HOME /home/ci/.cache/bazelisk

# Pre-download some bazel version to save time in CI
RUN USE_BAZEL_VERSION=latest bazelisk --version
