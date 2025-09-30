# Repository Guidelines

## Project Structure & Module Organization
WhisperTray is a Go 1.22 app with the entry point in `cmd/whisper-tray`. Cross-platform core lives under `internal/` with focused packages (`internal/audio`, `internal/hotkey`, `internal/whisper`, etc.), while macOS assets are in `resources/`. `scripts/` holds install helpers, and `prompts/` plus `CLAUDE.md` record product behaviour. Build outputs land in `bin/`, and vendored Whisper bindings are staged in `vendor/whisper.cpp`.

## Build, Test, and Development Commands
- `make install-deps` downloads Go modules and prepares the Whisper bindings.
- `make dev` compiles a fast local binary when `vendor/whisper.cpp` already exists.
- `make all` or `make build` runs the full pipeline, ensuring `libwhisper.a` is rebuilt.
- `make run` builds then launches `./bin/whisper-tray`.
- `make clean` removes `bin/` and the vendored dependency when you need a fresh setup.

## Coding Style & Naming Conventions
Follow idiomatic Go: use tabs for indentation and keep identifiers in MixedCaps (types) or lowerCamelCase (funcs/vars). Always run `gofmt`, and prefer `goimports` to maintain imports. Log through `internal/logging` and zerolog structured fields; use short `ctx` parameters and package-local structs for cohesion. Configuration lives in `internal/config` and should be marshalled via the existing helpers.

## Testing Guidelines
Write unit tests with Go's standard library (`testing`) in files suffixed `_test.go`. Run the full suite with `make test`; target a package with `make test TEST=./internal/audio`. Tests that touch CGO audio bindings should use fake devices or the `internal/audio` abstractions so they pass on CI. Capture race conditions with `go test -race ./...` before large refactors.

## Commit & Pull Request Guidelines
Commit subjects are short, imperative, and may include scope tags (e.g., `Refactor Makefile...`). Group related edits per commit and note linked issues or PR numbers in the body. PRs should summarize user impact, list platform checks (macOS at minimum), and attach screenshots or log snippets when UI or tray behaviour changes.

## Security & Configuration Notes
Document any new permissions in `README.md`, especially Accessibility and Microphone requirements. Avoid checking in model binaries; rely on the auto-download flow in `internal/whisper`. Protect secrets by using local `.env` files and never committing entries under `resources/` or `scripts/` that embed credentials.
