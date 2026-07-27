FROM node:26-slim AS docs-base

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

FROM docs-base AS docs-preview

ARG PREVIEW_BRANCH

RUN apt-get update && apt-get install -y --no-install-recommends jq gh git

RUN --mount=type=secret,id=gh_token,env=GH_TOKEN gh repo clone arm/topo work -- --branch ${PREVIEW_BRANCH}

WORKDIR /work
RUN --mount=type=secret,id=gh_token,env=GH_TOKEN docs/scripts/generate-docs-config.sh --include-head
RUN arm-docs validate --docs-root docs
RUN mkdir /app && mv /tmp/arm-docs-build-topo-* /app/topo

WORKDIR /app
ENTRYPOINT [ "npx", "-y", "http-server" ]
