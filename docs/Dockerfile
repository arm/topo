FROM node:26-slim

ARG NPM_CONFIG_REGISTRY
ENV NPM_CONFIG_REGISTRY=${NPM_CONFIG_REGISTRY}

ENV ARM_DOCS_PACKAGE_VERSION=0.6.0
ENV PNPM_VERSION=10.33.3

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    echo "deb [trusted=yes] https://artifacts.tools.arm.com/tools-deb bionic main" > /etc/apt/sources.list.d/artifactory.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends "arm-docs-github-action=$ARM_DOCS_PACKAGE_VERSION" && \
    rm -f /etc/apt/sources.list.d/artifactory.list

RUN npm install -g pnpm@${PNPM_VERSION}

