ARG GO_VERSION=1.18
FROM golang:${GO_VERSION}-bullseye as bazelisk

ARG BAZELISK_VERSION=1.11.0
RUN CGO_ENABLED=0 GOOS=linux GOBIN=/opt/bazelisk/bin go install github.com/bazelbuild/bazelisk@v${BAZELISK_VERSION}

FROM debian:bullseye-slim

# Install packages
# git required for blueprint
# build-essential needed for io_bazel_rules_go
RUN apt update \
 && apt -y --no-install-recommends install \
      ca-certificates \
      build-essential \
      git \
      python3 \
      python3-dev \
      python3-pip \
      python3-ply \
 && rm -rf /var/lib/apt/lists/*


ARG USER_ID=1001
RUN useradd --create-home --shell /bin/bash --uid $USER_ID ci
USER ci
WORKDIR /home/ci

COPY --from=bazelisk /opt/bazelisk/bin/bazelisk .local/bin/bazelisk

ENV PATH /home/ci/.local/bin:$PATH
ENV BAZELISK_HOME /home/ci/.cache/bazelisk

# Pre-download some bazel version to save time in CI
RUN USE_BAZEL_VERSION=latest bazelisk --version && \
	USE_BAZEL_VERSION=6.0.0 bazelisk --version && \
	USE_BAZEL_VERSION=5.4.0 bazelisk --version
