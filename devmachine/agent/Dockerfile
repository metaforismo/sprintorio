FROM golang:1.25.13-alpine AS go-stage

FROM node:22.23.1-alpine3.23

ARG PROVIDER
ARG NPM_VERSION=12.0.1
ARG BRACE_EXPANSION_VERSION=5.0.12
ARG IP_ADDRESS_VERSION=10.3.1
ARG TAR_VERSION=7.5.22
ARG UNDICI_VERSION=6.28.1
ARG OPENSSL_MIN_VERSION=3.5.8-r0
ARG CLAUDE_CODE_VERSION=2.1.218
ARG OPENCODE_VERSION=1.18.4
ARG CODEX_VERSION=0.145.0

RUN apk add --no-cache --upgrade bash ca-certificates curl git make openssh-client \
         "libcrypto3>=${OPENSSL_MIN_VERSION}" "libssl3>=${OPENSSL_MIN_VERSION}" \
    && npm install -g "npm@${NPM_VERSION}" \
    && case "$PROVIDER" in \
         claude-code) npm install -g --allow-scripts=@anthropic-ai/claude-code "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" ;; \
         opencode) npm install -g --allow-scripts=opencode-ai "opencode-ai@${OPENCODE_VERSION}" ;; \
         codex) npm install -g "@openai/codex@${CODEX_VERSION}" ;; \
         *) echo "unsupported provider: $PROVIDER" >&2; exit 1 ;; \
       esac \
    && npm install --prefix /tmp/npm-patches --ignore-scripts --package-lock=false "brace-expansion@${BRACE_EXPANSION_VERSION}" \
         "ip-address@${IP_ADDRESS_VERSION}" "tar@${TAR_VERSION}" "undici@${UNDICI_VERSION}" \
    && rm -rf /usr/local/lib/node_modules/npm/node_modules/brace-expansion \
         /usr/local/lib/node_modules/npm/node_modules/ip-address \
         /usr/local/lib/node_modules/npm/node_modules/tar \
         /usr/local/lib/node_modules/npm/node_modules/undici \
    && cp -a /tmp/npm-patches/node_modules/brace-expansion \
         /tmp/npm-patches/node_modules/ip-address /tmp/npm-patches/node_modules/tar \
         /tmp/npm-patches/node_modules/undici /usr/local/lib/node_modules/npm/node_modules/ \
    && node -e 'const actual=require("/usr/local/lib/node_modules/npm/node_modules/brace-expansion/package.json").version;if(actual!==process.argv[1])throw new Error(`unexpected brace-expansion ${actual}`)' "${BRACE_EXPANSION_VERSION}" \
    && node -e 'for(const [name,expected] of [["ip-address",process.argv[1]],["tar",process.argv[2]],["undici",process.argv[3]]]){const path="/usr/local/lib/node_modules/npm/node_modules/"+name,actual=require(path+"/package.json").version;if(actual!==expected)throw new Error(`unexpected ${name} ${actual}`);require(path)}' "${IP_ADDRESS_VERSION}" "${TAR_VERSION}" "${UNDICI_VERSION}" \
    && rm -rf /tmp/npm-patches /root/.npm \
    && case "$PROVIDER" in \
         claude-code) claude --version ;; \
         opencode) opencode --version ;; \
         codex) codex --version ;; \
       esac \
    && npm --version

COPY --from=go-stage /usr/local/go /usr/local/go
ENV GOPATH=/home/sprintorio/go \
    GOCACHE=/home/sprintorio/.cache/go-build \
    GOMODCACHE=/home/sprintorio/go/pkg/mod \
    PATH="/usr/local/go/bin:${PATH}"

COPY common/sprintorio-agent-runner /usr/local/bin/sprintorio-agent-runner
RUN chmod 0755 /usr/local/bin/sprintorio-agent-runner \
    && mkdir -p /home/sprintorio /home/sprintorio/go /home/sprintorio/.cache/go-build \
    && chown -R 1000:1000 /home/sprintorio

USER 1000:1000
ENV HOME=/home/sprintorio
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/sprintorio-agent-runner"]
