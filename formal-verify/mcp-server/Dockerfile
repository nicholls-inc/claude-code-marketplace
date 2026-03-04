FROM --platform=linux/amd64 ubuntu:22.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget unzip ca-certificates libgomp1 libicu70 \
    && rm -rf /var/lib/apt/lists/*

ARG DAFNY_VERSION=4.11.0
RUN wget -q "https://github.com/dafny-lang/dafny/releases/download/v${DAFNY_VERSION}/dafny-${DAFNY_VERSION}-x64-ubuntu-22.04.zip" \
    -O /tmp/dafny.zip \
    && unzip -q /tmp/dafny.zip -d /opt \
    && rm /tmp/dafny.zip

ENV PATH="/opt/dafny:${PATH}"
RUN dafny --version

WORKDIR /work
ENTRYPOINT ["dafny"]
