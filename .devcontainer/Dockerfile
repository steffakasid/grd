# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.154.0/containers/alpine/.devcontainer/base.Dockerfile

# [Choice] Alpine version: 3.12, 3.11, 3.10
ARG VARIANT="3.12"
FROM mcr.microsoft.com/vscode/devcontainers/base:0-alpine-${VARIANT}

# ** [Optional] Uncomment this section to install additional packages. **
RUN apk update \
    && apk add --no-cache zsh=~5.8-r1

# Install Go
ARG GO_VERSION="1.15.6"
RUN apk add --no-cache --virtual .build-deps gcc musl-dev openssl go && \
    curl https://dl.google.com/go/go${GO_VERSION}.src.tar.gz --output go.src.tar.gz && \
    tar -C /usr/local -xzf go.src.tar.gz && \
    cd /usr/local/go/src/ && \
    ./make.bash && \
    apk del .build-deps
