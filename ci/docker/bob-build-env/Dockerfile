ARG GO_VERSION=1.18
FROM golang:${GO_VERSION}-bullseye

ARG BAZELISK_VERSION=1.24.0
RUN CGO_ENABLED=0 GOOS=linux GOBIN=/usr/local/bin go install github.com/bazelbuild/bazelisk@v${BAZELISK_VERSION}

# Install packages
# git required for blueprint
# build-essential needed for io_bazel_rules_go
RUN apt update \
 && apt -y --no-install-recommends install \
      ca-certificates \
      build-essential \
      gcc-aarch64-linux-gnu \
      git \
      ninja-build \
      python3 \
      python3-dev \
      python3-pip \
      python-is-python3 \
      python3-ply \
      rsync \
 && rm -rf /var/lib/apt/lists/*

RUN printf '%s\n' 'export PATH=/home/ci/.local/bin:/go/bin:/usr/local/go/bin:$PATH' \
     > /etc/profile.d/bob-build-env-path.sh


ARG USER_ID=1001
RUN useradd --create-home --shell /bin/bash --uid $USER_ID ci
USER ci
WORKDIR /home/ci

ENV PATH=/home/ci/.local/bin:$PATH
ENV BAZELISK_HOME=/home/ci/.cache/bazelisk

# Pre-download _latest_ bazel version to save time in CI
RUN bash -lc 'go version' && \
     USE_BAZEL_VERSION=latest bazelisk --version && \
     USE_BAZEL_VERSION=8.0.0 bazelisk --version
