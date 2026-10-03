# Agent client configurations

Sprintorio's integration surface is the CLI plus MCP. A model name alone does not define an MCP connection: the surrounding harness launches stdio servers or calls remote HTTP servers. These examples use the official client formats reviewed on 2026-10-03. They are configuration documentation, not evidence that the named external accounts have been connected.

Local stdio launches `/absolute/path/sprintorio mcp` with `SPRINTORIO_URL`, `SPRINTORIO_WORKSPACE` and `SPRINTORIO_TOKEN` supplied through the process environment/secret manager. Remote clients call an HTTPS `/mcp` endpoint with their own workspace agent token in the Authorization header. The server checks every HTTP caller against the workspace API. Keep real secrets out of source control and copied configuration text.

| Client/harness | Local stdio format | Remote HTTP format | Verification boundary |
| --- | --- | --- | --- |
| [Claude Code](https://code.claude.com/docs/en/mcp) | `mcpServers` entry with `type: stdio`, `command`, `args`, `env` | `type: http`, `url`, `headers`; `${VAR}` interpolation | Official format reviewed; account test NOT RUN |
| [Cursor](https://cursor.com/docs/mcp) | `mcpServers`, `command`, `args`, `env` | `url`, `headers`; `${env:VAR}` interpolation | Official format reviewed; account test NOT RUN |
| [Codex](https://learn.chatgpt.com/docs/extend/mcp?surface=cli) | TOML `mcp_servers`, `command`, `args`, `env_vars` | `url`, `bearer_token_env_var` | Official format reviewed; account test NOT RUN |
| [Grok/xAI API](https://docs.x.ai/developers/tools/remote-mcp) | Remote-MCP API does not launch this local binary | MCP tool `server_url`, `server_label`, `headers` or authorization configuration | Requires an externally reachable HTTPS deployment; cloud/account test NOT RUN |
| [Muse Code](https://dev.meta.ai/docs/muse-code/extending) | `mcp_servers`, `transport: stdio`, `command`, `args`, `env` | `transport: streamable_http`, `url`, `headers`; `${VAR}` interpolation | Official format reviewed; Muse Code account test NOT RUN |
| [OpenCode](https://opencode.ai/docs/mcp-servers/) | `mcp`, `type: local`, `command` array, `environment` | `type: remote`, `url`, `headers` | Official format reviewed; account test NOT RUN |
| [Gemini CLI](https://geminicli.com/docs/tools/mcp-server/) | `mcpServers`, `command`, `args`, `env` | `httpUrl`, `headers`; use `httpUrl` for Streamable HTTP | Official format reviewed; account test NOT RUN |

Muse Code is the harness configured here. Muse Spark is a model that may be selected by a supporting harness; this is not a claim that a standalone Muse Spark/Meta AI chat can accept the same configuration.

## Local examples

Replace only the binary path. Set the three environment variables before launching the client. Settings → Agents also generates client-specific examples without inserting the token value.

Claude Code project `.mcp.json`:

```json
{"mcpServers":{"sprintorio":{"type":"stdio","command":"/absolute/path/sprintorio","args":["mcp"],"env":{"SPRINTORIO_URL":"${SPRINTORIO_URL}","SPRINTORIO_WORKSPACE":"${SPRINTORIO_WORKSPACE}","SPRINTORIO_TOKEN":"${SPRINTORIO_TOKEN}"}}}}
```

Cursor `.cursor/mcp.json`:

```json
{"mcpServers":{"sprintorio":{"command":"/absolute/path/sprintorio","args":["mcp"],"env":{"SPRINTORIO_URL":"${env:SPRINTORIO_URL}","SPRINTORIO_WORKSPACE":"${env:SPRINTORIO_WORKSPACE}","SPRINTORIO_TOKEN":"${env:SPRINTORIO_TOKEN}"}}}}
```

Codex `config.toml`:

```toml
[mcp_servers.sprintorio]
command = "/absolute/path/sprintorio"
args = ["mcp"]
env_vars = ["SPRINTORIO_URL", "SPRINTORIO_WORKSPACE", "SPRINTORIO_TOKEN"]
```

Muse Code settings:

```json
{"mcp_servers":{"sprintorio":{"transport":"stdio","command":"/absolute/path/sprintorio","args":["mcp"],"env":{"SPRINTORIO_URL":"${SPRINTORIO_URL}","SPRINTORIO_WORKSPACE":"${SPRINTORIO_WORKSPACE}","SPRINTORIO_TOKEN":"${SPRINTORIO_TOKEN}"},"enabled":true,"mode":"required"}}}
```

OpenCode configuration:

```json
{"mcp":{"sprintorio":{"type":"local","command":["/absolute/path/sprintorio","mcp"],"environment":{"SPRINTORIO_URL":"{env:SPRINTORIO_URL}","SPRINTORIO_WORKSPACE":"{env:SPRINTORIO_WORKSPACE}","SPRINTORIO_TOKEN":"{env:SPRINTORIO_TOKEN}"},"enabled":true}}}
```

Gemini CLI `settings.json`:

```json
{"mcpServers":{"sprintorio":{"command":"/absolute/path/sprintorio","args":["mcp"],"env":{"SPRINTORIO_URL":"${SPRINTORIO_URL}","SPRINTORIO_WORKSPACE":"${SPRINTORIO_WORKSPACE}","SPRINTORIO_TOKEN":"${SPRINTORIO_TOKEN}"}}}}
```

## Remote examples

Run `sprintorio mcp-http` with the API origin and workspace configured. Its default address is `127.0.0.1:8091`; put HTTPS termination in front of an explicitly configured listener for remote use. Exact browser Origins may be allowed with `--origins`; there is no wildcard allowlist. Native clients can omit Origin. See [AGENTS_API.md](../AGENTS_API.md) for deployment and limits.

Codex can forward a token from its environment without storing the value:

```toml
[mcp_servers.sprintorio]
url = "https://mcp.example.com/mcp"
bearer_token_env_var = "SPRINTORIO_TOKEN"
```

Cursor:

```json
{"mcpServers":{"sprintorio":{"url":"https://mcp.example.com/mcp","headers":{"Authorization":"Bearer ${env:SPRINTORIO_TOKEN}"}}}}
```

Muse Code:

```json
{"mcp_servers":{"sprintorio":{"transport":"streamable_http","url":"https://mcp.example.com/mcp","headers":{"Authorization":"Bearer ${SPRINTORIO_TOKEN}"},"enabled":true,"mode":"required"}}}
```

For Grok, construct the MCP tool descriptor inside the calling application so the token value comes from its secret environment at request time:

```python
import os

sprintorio_tool = {
    "type": "mcp",
    "server_url": os.environ["SPRINTORIO_MCP_URL"],
    "server_label": "sprintorio",
    "headers": {"Authorization": "Bearer " + os.environ["SPRINTORIO_TOKEN"]},
}
```

Pass that descriptor through the xAI API's documented MCP tool mechanism. The URL must be reachable by the remote service; a local loopback address cannot establish a cloud connection. Native SDK field names may differ (`extra_headers` instead of `headers`). No Grok consumer-chat connector or public hosting is implied by this example.

## Verify a connection

After setup, list the three tools, call `discover`, request `schema` for one operation, then use `action` for a permitted read. Use a disposable workspace for a write and verify its persistence through the app. Test read-only/revoked tokens and one stale delivery-plan version. Local fixture/SDK tests, actual API persistence, public HTTPS reachability and each named harness account are separate results. Record them in [VERIFICATION.md](../VERIFICATION.md); do not infer account success from these configuration examples.
