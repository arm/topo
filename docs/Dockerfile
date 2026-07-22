FROM node:26-slim

ENV ARM_DOCS_PACKAGE_VERSION=0.5.1
ENV PNPM_VERSION=10.33.3

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    echo "deb [trusted=yes] https://artifacts.tools.arm.com/tools-deb bionic main" > /etc/apt/sources.list.d/artifactory.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends "arm-docs-github-action=$ARM_DOCS_PACKAGE_VERSION" && \
    rm -f /etc/apt/sources.list.d/artifactory.list

RUN npm install -g pnpm@${PNPM_VERSION}

EXPOSE 3000
ENTRYPOINT ["arm-docs", "preview"]
CMD [ "--docs-root", "/app/docs" ]