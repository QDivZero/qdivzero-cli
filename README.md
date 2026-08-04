<p align="center"><img src="assets/qdiv0-mark.png" alt="QDivZero" width="256"></p>

# qdivzero-cli

Command-line client for the QDivZero API (https://api.qdiv0.com), built on the qdivzero-go SDK.

[![CI](https://github.com/QDivZero/qdivzero-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/QDivZero/qdivzero-cli/actions/workflows/ci.yml)
[![Releases](https://img.shields.io/github/v/release/QDivZero/qdivzero-cli)](https://github.com/QDivZero/qdivzero-cli/releases)

## Install

- **Install script**: installs the latest release to `~/.local/bin`:

  ```sh
  curl -fsSL https://raw.githubusercontent.com/QDivZero/qdivzero-cli/main/scripts/install.sh | sh
  ```

- **Homebrew**: `brew install QDivZero/tap/qdivzero` (once the tap is populated)
- **Binaries**: download directly from [GitHub Releases](https://github.com/QDivZero/qdivzero-cli/releases)

## Configure

Configure authentication before using the CLI:

- **Interactive**:

  ```sh
  qdivzero configure
  ```

- **Non-interactive**:

  ```sh
  qdivzero configure --token <token>
  qdivzero configure --email <e> --password <p>
  ```

Use `--force` to overwrite an existing credentials file. The CLI writes `~/.qdivzero/credentials`, shared with all qdivzero SDKs.

## Usage

| Command | Description |
| --- | --- |
| `qdivzero me` | Show the authenticated user |
| `qdivzero accounts list` | List accounts |
| `qdivzero instances list\|show\|start\|stop\|delete\|create` | Manage GPU instances; `create` deploys models and requires `--huggingface-repo-id` |
| `qdivzero serving-endpoints list\|create` | Manage serving endpoints |
| `qdivzero models list` | List models |
| `qdivzero api-keys list\|create\|revoke` | Manage API keys |
| `qdivzero chat [prompt]` | Chat with a model; interactive REPL without a prompt, one-shot with one; SSE streaming with `--model`/`--max-tokens`/`--temperature`/`--no-stream`/`--list-models` |
| `qdivzero version` | Print CLI and SDK versions |

The global flag `--json` produces machine-readable output for any command.

## Examples

```sh
# Configure with an access token
qdivzero configure --token <token>

# List instances
qdivzero instances list

# Deploy a model on a GPU instance
qdivzero instances create --name my-gpu \
  --huggingface-repo-id TheBloke/Mixtral-8x7B-Instruct-v0.1-GGUF --serverless

# Start an instance
qdivzero instances start <id>

# List models
qdivzero models list

# One-shot chat
qdivzero chat --model <model-id> "explain HTTP"

# Interactive REPL
qdivzero chat --model <model-id>

# Machine-readable output
qdivzero instances list --json
```

## Development

```sh
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

Build a single binary:

```sh
go build -o qdivzero main.go
```

### Release

Tag a release with `v*` and goreleaser builds binaries and updates the Homebrew tap. Publishing the Homebrew formula requires the `GORELEASER_TOKEN` secret.

## License

[MIT](LICENSE)
