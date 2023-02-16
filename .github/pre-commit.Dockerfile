FROM golang:1.20-bullseye as go

FROM debian:bookworm-slim

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
