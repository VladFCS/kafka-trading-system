# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go microservices workspace for a Kafka-based trading simulator. Service entrypoints should live in `cmd/<service>/main.go`. Shared contracts belong in `api/`, generated code in `gen/`, and service-specific business logic under `internal/<service>/`. Keep transport, domain, persistence, and integration concerns separated so each service owns its own boundaries clearly. Infrastructure manifests belong in `deployments/`, while local orchestration is managed with `compose.yaml`.

## Build, Test, and Development Commands
- `go test ./...`: run the full Go test suite.
- `go build ./...`: verify all packages compile.
- `docker compose up -d`: start local infrastructure such as Kafka and PostgreSQL.
- `docker compose down`: stop the local stack.
- `make proto`: regenerate protobuf and gRPC code when `.proto` files change.

If new commands are introduced later, keep this file aligned with the repo’s actual workflow.

## Coding Style & Naming Conventions
Use idiomatic Go. Format changes with `gofmt`. Keep package names lowercase, exported names in `PascalCase`, and unexported helpers in `camelCase`. Pass `context.Context` across service boundaries. Prefer clear interfaces, explicit domain models, and structured logging. Keep Kafka event names consistent and protobuf contracts versioned intentionally.

## Testing Guidelines
Place tests next to the packages they cover using `*_test.go`. Favor table-driven tests for validation, matching rules, and state transitions. Add integration coverage where behavior depends on Kafka, PostgreSQL, or gRPC contracts. For matching engine logic, test ordering, partial fills, duplicates, and edge cases explicitly.

## Commit & Pull Request Guidelines
Use concise Conventional Commit prefixes such as `feat:`, `fix:`, `refactor:`, `test:`, and `chore:`. PRs should summarize the affected service, list verification steps, and mention contract or event-schema changes when relevant.

## Architecture Priorities
Preserve service ownership, deterministic matching behavior, partition-aware Kafka design, idempotent consumers, and clean separation between write-side and read-side responsibilities. Prefer simple educational designs that still model realistic distributed-systems trade-offs.
