FROM node:22.23.1-bookworm-slim@sha256:6c74791e557ce11fc957704f6d4fe134a7bc8d6f5ca4403205b2966bd488f6b3 AS node-stage

FROM ubuntu:26.04

ARG TARGETARCH
ARG DEBIAN_FRONTEND=noninteractive
ARG CODE_SERVER_VERSION=4.129.0
ARG CODE_SERVER_AMD64_SHA256=66160d99431595b0e9f4245abae8d0f472289fb77c97007ce79f2d64d06507f8
ARG CODE_SERVER_ARM64_SHA256=e504da34067eacc04e03265eccf90e5d2c3fdc3f06c91c2028a5a80f1d05a5b8
ARG TTYD_VERSION=1.7.7
ARG TTYD_AMD64_SHA256=8a217c968aba172e0dbf3f34447218dc015bc4d5e59bf51db2f2cd12b7be4f55
ARG TTYD_ARM64_SHA256=b38acadd89d1d396a0f5649aa52c539edbad07f4bc7348b27b4f4b7219dd4165
ARG NPM_VERSION=12.0.1
ARG BRACE_EXPANSION_VERSION=5.0.12
ARG IP_ADDRESS_VERSION=10.3.1
ARG NPM_TAR_VERSION=7.5.22
ARG BASIC_FTP_VERSION=6.2.1
ARG NPM_UNDICI_VERSION=6.28.1
ARG VSCODE_UNDICI_VERSION=7.29.1
ARG CLAUDE_CODE_VERSION=2.1.218
ARG OPENCODE_VERSION=1.18.4
ARG CODEX_VERSION=0.145.0

COPY --from=node-stage /usr/local /usr/local
COPY common/sprintorio-service-entrypoint /usr/local/bin/sprintorio-service-entrypoint
COPY ide/start.sh /usr/local/bin/sprintorio-ide-start
COPY ide/shell.sh /usr/local/bin/sprintorio-shell
COPY ide/checkout.sh /usr/local/bin/sprintorio-checkout
COPY ide/terminal-session.sh /usr/local/bin/sprintorio-terminal-session
COPY ide/command-event.js /usr/local/lib/sprintorio-command-event.js
COPY ide/tmux.conf /etc/tmux.conf
COPY ide/tests/ftp-compatibility.mjs /usr/local/lib/sprintorio-ftp-compatibility.mjs
# VS Code's npm support extension is not the npm CLI package; keep its manifest identity unambiguous.
RUN rm -f /usr/bin/pebble \
    && apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl git ripgrep sudo tmux xz-utils \
    && case "${TARGETARCH}" in \
         amd64) package_arch=amd64; code_server_sha="$CODE_SERVER_AMD64_SHA256"; ttyd_arch=x86_64; ttyd_sha="$TTYD_AMD64_SHA256" ;; \
         arm64) package_arch=arm64; code_server_sha="$CODE_SERVER_ARM64_SHA256"; ttyd_arch=aarch64; ttyd_sha="$TTYD_ARM64_SHA256" ;; \
         *) echo "unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
       esac \
    && curl -fsSL "https://github.com/coder/code-server/releases/download/v${CODE_SERVER_VERSION}/code-server_${CODE_SERVER_VERSION}_${package_arch}.deb" -o /tmp/code-server.deb \
    && echo "${code_server_sha}  /tmp/code-server.deb" | sha256sum -c - \
    && apt-get install -y --no-install-recommends /tmp/code-server.deb \
    && curl -fsSL "https://github.com/tsl0922/ttyd/releases/download/${TTYD_VERSION}/ttyd.${ttyd_arch}" -o /usr/local/bin/ttyd \
    && echo "${ttyd_sha}  /usr/local/bin/ttyd" | sha256sum -c - \
    && chmod 0755 /usr/local/bin/ttyd \
    && npm install -g "npm@${NPM_VERSION}" \
    && npm install -g --allow-scripts=@anthropic-ai/claude-code "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
    && npm install -g --allow-scripts=opencode-ai "opencode-ai@${OPENCODE_VERSION}" \
    && npm install -g "@openai/codex@${CODEX_VERSION}" \
    && npm install --prefix /tmp/npm-patches --ignore-scripts --package-lock=false "brace-expansion@${BRACE_EXPANSION_VERSION}" \
         "ip-address@${IP_ADDRESS_VERSION}" "undici@${NPM_UNDICI_VERSION}" "tar@${NPM_TAR_VERSION}" \
    && rm -rf /usr/local/lib/node_modules/npm/node_modules/brace-expansion \
         /usr/local/lib/node_modules/npm/node_modules/ip-address \
         /usr/local/lib/node_modules/npm/node_modules/undici \
         /usr/local/lib/node_modules/npm/node_modules/tar \
    && cp -a /tmp/npm-patches/node_modules/brace-expansion \
         /tmp/npm-patches/node_modules/ip-address /tmp/npm-patches/node_modules/undici /tmp/npm-patches/node_modules/tar \
         /usr/local/lib/node_modules/npm/node_modules/ \
    && node -e 'const actual=require("/usr/local/lib/node_modules/npm/node_modules/brace-expansion/package.json").version;if(actual!==process.argv[1])throw new Error(`unexpected brace-expansion ${actual}`)' "${BRACE_EXPANSION_VERSION}" \
    && node -e 'const path="/usr/local/lib/node_modules/npm/node_modules/tar",actual=require(path+"/package.json").version;if(actual!==process.argv[1])throw new Error(`unexpected tar ${actual}`);require(path)' "${NPM_TAR_VERSION}" \
    && npm install --prefix /tmp/code-server-patches --ignore-scripts --package-lock=false \
         "js-yaml@4.3.2" "shell-quote@1.9.0" "tar@7.5.22" \
         "ip-address@${IP_ADDRESS_VERSION}" "undici@${VSCODE_UNDICI_VERSION}" "basic-ftp@${BASIC_FTP_VERSION}" \
    && rm -rf /usr/lib/code-server/node_modules/js-yaml \
         /usr/lib/code-server/lib/vscode/node_modules/shell-quote \
         /usr/lib/code-server/lib/vscode/node_modules/tar \
         /usr/lib/code-server/node_modules/ip-address \
         /usr/lib/code-server/node_modules/basic-ftp \
         /usr/lib/code-server/lib/vscode/node_modules/ip-address \
         /usr/lib/code-server/lib/vscode/node_modules/undici \
    && cp -a /tmp/code-server-patches/node_modules/js-yaml /usr/lib/code-server/node_modules/ \
    && cp -a /tmp/code-server-patches/node_modules/shell-quote /usr/lib/code-server/lib/vscode/node_modules/ \
    && cp -a /tmp/code-server-patches/node_modules/tar /usr/lib/code-server/lib/vscode/node_modules/ \
    && cp -a /tmp/code-server-patches/node_modules/ip-address /tmp/code-server-patches/node_modules/basic-ftp /usr/lib/code-server/node_modules/ \
    && cp -a /tmp/code-server-patches/node_modules/ip-address /tmp/code-server-patches/node_modules/undici /usr/lib/code-server/lib/vscode/node_modules/ \
    && node -e 'for(const [path,expected] of [["/usr/local/lib/node_modules/npm/node_modules/ip-address",process.argv[1]],["/usr/lib/code-server/node_modules/ip-address",process.argv[1]],["/usr/lib/code-server/lib/vscode/node_modules/ip-address",process.argv[1]],["/usr/local/lib/node_modules/npm/node_modules/undici",process.argv[2]],["/usr/lib/code-server/lib/vscode/node_modules/undici",process.argv[3]]]){const actual=require(path+"/package.json").version;if(actual!==expected)throw new Error(`unexpected ${path} ${actual}`);require(path)}' "${IP_ADDRESS_VERSION}" "${NPM_UNDICI_VERSION}" "${VSCODE_UNDICI_VERSION}" \
    && node -e 'const path="/usr/lib/code-server/node_modules/basic-ftp",actual=require(path+"/package.json").version;if(actual!==process.argv[1])throw new Error(`unexpected basic-ftp ${actual}`);require(path)' "${BASIC_FTP_VERSION}" \
    && node /usr/local/lib/sprintorio-ftp-compatibility.mjs /usr/lib/code-server \
    && node -e 'const fs=require("fs"),path="/usr/lib/code-server/lib/vscode/extensions/npm/package.json",manifest=JSON.parse(fs.readFileSync(path));manifest.name="npm-support";fs.writeFileSync(path,JSON.stringify(manifest,null,2)+"\n")' \
    && rm -rf /root/.npm /tmp/npm-patches /tmp/code-server-patches /var/lib/apt/lists/* \
    && rm -f /tmp/code-server.deb /usr/bin/pebble \
         /etc/ssl/private/ssl-cert-snakeoil.key /etc/ssl/certs/ssl-cert-snakeoil.pem \
    && mkdir -p /home/sprintorio \
    && chown 1000:1000 /home/sprintorio \
    && chmod 0755 /usr/local/bin/sprintorio-service-entrypoint /usr/local/bin/sprintorio-ide-start /usr/local/bin/sprintorio-shell /usr/local/bin/sprintorio-checkout /usr/local/bin/sprintorio-terminal-session \
    && code-server --version \
    && node --version \
    && npm --version \
    && opencode --version \
    && claude --version \
    && codex --version \
    && ttyd --version \
    && rm -rf /root/.config/code-server \
    && user_name=$(getent passwd 1000 | cut -d: -f1) \
    && printf '%s ALL=(ALL) NOPASSWD:ALL\n' "$user_name" > /etc/sudoers.d/sprintorio-builder \
    && chmod 0440 /etc/sudoers.d/sprintorio-builder

USER 1000:1000
ENV HOME=/home/sprintorio
EXPOSE 8080 7681
CMD ["/usr/local/bin/sprintorio-ide-start"]
