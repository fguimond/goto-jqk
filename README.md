# goto-jqk

A REST API server written in Go, scaffolded with a clean layered architecture,
a `kubectl`-style command-line interface, structured JSON logging, and an
auto-generated OpenAPI schema.

## Features

- **CLI** built with [cobra](https://github.com/spf13/cobra) /
  [viper](https://github.com/spf13/viper) — `run`, `version`, plus the
  cobra-provided `completion` and `help` commands.
- **Structured logging** with the standard library `log/slog` (JSON output).
- **OpenAPI 3.1** schema and interactive docs generated automatically from Go
  types via [huma](https://github.com/danielgtaylor/huma).
- **Layered design**: HTTP handler → service → store (in-memory today).
- **UUID v4** identifiers.
- **CI** (golangci-lint + build/test) and **release automation**
  (release-please) via GitHub Actions.

## Requirements

- Go 1.25 or newer

## Getting started

```bash
# Build the binary into ./bin
make build

# Or run directly
go run ./cmd/goto-jqk run
```

The server listens on `:8080` by default.

## Commands

```bash
goto-jqk run                 # start the REST API server
goto-jqk run --addr :9090    # listen on a custom address
goto-jqk version             # print version, commit and build date (JSON)
goto-jqk completion bash     # generate a shell completion script
goto-jqk help                # show help
```

### Configuration

Configuration is resolved by viper in the following order (highest priority
first):

1. Command-line flags (e.g. `--log-level`, `--addr`)
2. Environment variables prefixed with `GOTO_JQK_` (e.g. `GOTO_JQK_LOG_LEVEL=debug`)
3. A config file at `$HOME/.goto-jqk.yaml` (or `--config <path>`)

| Setting     | Flag          | Env var                | Default   |
| ----------- | ------------- | ---------------------- | --------- |
| Log level   | `--log-level` | `GOTO_JQK_LOG_LEVEL`   | `info`    |
| Listen addr | `--addr`      | `GOTO_JQK_ADDR`        | `:8080`   |

## API

The API is versioned under `/api/v1`.

| Method   | Path                                         | Description               | Success |
| -------- | -------------------------------------------- | ------------------------- | ------- |
| `POST`   | `/api/v1/games`                              | Create a game             | `201`   |
| `GET`    | `/api/v1/games`                              | List games                | `200`   |
| `DELETE` | `/api/v1/games/{id}`                         | Delete a game             | `204`   |
| `PATCH`  | `/api/v1/games/{gameId}/decks`               | Add decks to a game       | `200`   |
| `GET`    | `/api/v1/games/{gameId}/cards`               | List a game's cards       | `200`   |
| `POST`   | `/api/v1/games/{gameId}/players`             | Create a player           | `201`   |
| `DELETE` | `/api/v1/games/{gameId}/players/{playerId}`  | Remove a player from a game | `204` |
| `POST`   | `/api/v1/decks`                              | Create a deck             | `201`   |
| `GET`    | `/api/v1/decks`                              | List decks                | `200`   |
| `GET`    | `/healthz`                                   | Liveness check            | `200`   |

Adding decks to a game takes an [RFC 6902](https://datatracker.ietf.org/doc/html/rfc6902)
patch document. Only the `add` operation against the append pointer `/-` is supported, and
the patch is applied atomically: if any deck is unknown or already assigned to a game, none
are added.

Assigning a deck to a game *moves* its cards: they leave the deck and are appended to the
game deck, so a card is only ever in one place. An assigned deck therefore reports
`"remaining": 0`, and the game reports the running total as `gameDeckRemaining`.

The game resource carries that count rather than the cards, so listing games stays a fixed
size per game. The cards themselves are served by `GET /api/v1/games/{gameId}/cards`, which
returns them in the order they sit in the game deck: decks in the order they were added,
each contributing its cards in deck order. That is the order they will be dealt in.

Players belong to exactly one game and are only reachable through it, so they are created
and removed on game-scoped routes. A new player holds no cards.

Additional endpoints provided automatically by huma:

| Path             | Description                        |
| ---------------- | ---------------------------------- |
| `/docs`          | Interactive API documentation      |
| `/openapi.json`  | OpenAPI 3.1 schema (JSON)          |
| `/openapi.yaml`  | OpenAPI 3.1 schema (YAML)          |

### Examples

Every JSON response also carries a `"$schema"` field linking to the resource schema
(e.g. `http://localhost:8080/schemas/Game.json`); it is elided below for readability.
The calls form one sequence — later commands reuse the IDs returned by earlier ones.

```bash
# Create a game
curl -sS -X POST http://localhost:8080/api/v1/games \
  -H 'Content-Type: application/json' \
  -d '{"name":"Chess"}'
# {"id":"f81d4fae-7dec-4d0e-a765-00a0c91e6bf6","name":"Chess","decks":[],
#  "gameDeckRemaining":0,"players":[]}

# Create an unassigned deck (52 cards of a standard deck)
curl -sS -X POST http://localhost:8080/api/v1/decks \
  -H 'Content-Type: application/json' \
  -d '{}'
# {"id":"1b4e28ba-2fa1-11d2-883f-0016d3cca427","remaining":52}

# Create a deck already assigned to the game. Its 52 cards move straight into
# the game deck, so the deck itself comes back empty.
curl -sS -X POST http://localhost:8080/api/v1/decks \
  -H 'Content-Type: application/json' \
  -d '{"gameId":"f81d4fae-7dec-4d0e-a765-00a0c91e6bf6"}'
# {"id":"6ba7b810-9dad-11d1-80b4-00c04fd430c8","gameId":"f81d4fae-7dec-4d0e-a765-00a0c91e6bf6","remaining":0}

# Add the unassigned deck to the game (RFC 6902 patch document). Its cards join
# those of the deck assigned above, for 104 in the game deck.
curl -sS -X PATCH http://localhost:8080/api/v1/games/f81d4fae-7dec-4d0e-a765-00a0c91e6bf6/decks \
  -H 'Content-Type: application/json' \
  -d '[{"op":"add","path":"/-","value":"1b4e28ba-2fa1-11d2-883f-0016d3cca427"}]'
# {"id":"f81d4fae-7dec-4d0e-a765-00a0c91e6bf6","name":"Chess",
#  "decks":["6ba7b810-9dad-11d1-80b4-00c04fd430c8","1b4e28ba-2fa1-11d2-883f-0016d3cca427"],
#  "gameDeckRemaining":104,"players":[]}

# List the game's cards, in the order they will be dealt
curl -sS http://localhost:8080/api/v1/games/f81d4fae-7dec-4d0e-a765-00a0c91e6bf6/cards
# [{"suit":"heart","value":"ace"},{"suit":"heart","value":"2"}, … 104 cards]

# Add a player to the game
curl -sS -X POST http://localhost:8080/api/v1/games/f81d4fae-7dec-4d0e-a765-00a0c91e6bf6/players \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice"}'
# {"id":"7d444840-9dc0-11d1-b245-5ffdce74fad2","gameId":"f81d4fae-7dec-4d0e-a765-00a0c91e6bf6",
#  "name":"Alice","cards":[]}

# Remove the player from the game
curl -sS -X DELETE \
  http://localhost:8080/api/v1/games/f81d4fae-7dec-4d0e-a765-00a0c91e6bf6/players/7d444840-9dc0-11d1-b245-5ffdce74fad2 -i
# HTTP/1.1 204 No Content

# Delete a game
curl -sS -X DELETE http://localhost:8080/api/v1/games/f81d4fae-7dec-4d0e-a765-00a0c91e6bf6 -i
# HTTP/1.1 204 No Content

# Health check
curl -sS http://localhost:8080/healthz
# {"status":"ok"}
```

## Project layout

```
cmd/goto-jqk/         Application entry point (main)
internal/
  cli/                cobra/viper commands (root, run, version)
  api/                HTTP server assembly and middleware
  handler/            HTTP handler layer (huma operations, OpenAPI)
  service/            Business logic
  store/              Persistence (in-memory implementation)
  model/              Domain types
  logging/            slog JSON logger setup
  version/            Build metadata injected at link time
```

The layers depend inward only: `handler → service → store`. The store is defined
as an interface, so swapping the in-memory implementation for a real database
later requires no changes to the service or handler layers.

## Development

```bash
make test     # go test ./...
make lint     # golangci-lint run
make tidy     # go mod tidy
make build    # build with version info injected
```

## Continuous integration

- **`.github/workflows/ci.yml`** — runs golangci-lint and build/vet/test on
  every push and pull request.
- **`.github/workflows/release-please.yml`** — maintains a release PR from
  [Conventional Commits](https://www.conventionalcommits.org/). When the release
  PR is merged, it tags a release and builds binaries for
  `linux`, `darwin`, and `windows` on both `amd64` and `arm64`, injecting the
  released version into the binary and attaching them to the GitHub release.

## Releasing

This project uses [release-please](https://github.com/googleapis/release-please).
Commit using Conventional Commits (`feat:`, `fix:`, `feat!:`, …); release-please
opens and maintains a release PR. Merging that PR creates the tag, the GitHub
release, and the attached cross-platform binaries.

## License

No license has been chosen yet. Add a `LICENSE` file to declare one.
