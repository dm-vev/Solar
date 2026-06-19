# Contributing

## Development Workflow

- Keep changes small and focused.
- Run `go fmt ./...` before submitting changes.
- Run `go test -race ./internal/...` and add tests for behavior changes.
- Keep protocol changes covered by packet or session tests.
- Run `golangci-lint run` before submitting (CI enforces this).

## Code Style

### Errors
- Wrap errors with `fmt.Errorf("...: %w", err)` to preserve context.
- Never panic in library or handler code; return an error instead.
- Validate user input (config, CLI args, protocol payloads) before use.
- Use sentinel errors (`var ErrXxx = errors.New(...)`) for expected conditions.

### Logging
- Use `log/slog` with structured fields, not formatted strings.
- Components should accept a `*slog.Logger` via constructor or setter.
- Do not call `slog.Default()` inside library code; use the injected logger.

### Concurrency
- Prefer `sync.RWMutex` for read-heavy state.
- Use `context.Context` for cancellation; never create `context.Background()` inside components.
- Close resources in reverse order of creation.
- Use `sync.WaitGroup` to guarantee goroutine shutdown before return.

### Packages
- Avoid `init()` functions; register components explicitly at startup.
- Avoid package-level mutable state; use structs with constructors.
- Define interfaces at the point of use (consumer side), not the producer side.
- Keep packages cohesive: one responsibility per package.

### Naming
- Exported identifiers get doc comments.
- Use mixedCaps (not snake_case) for Go identifiers.
- Constants for protocol values, dimensions, and magic numbers.
- Alias imports only to avoid name collisions; prefer the real package name.

### Testing
- Prefer table-driven tests.
- Use `t.Parallel()` where safe.
- Use `t.TempDir()` for filesystem tests.
- Test the public API, not implementation details.

### Configuration
- All config keys must be validated in `Config.Validate()`.
- Unknown config keys should cause an error, not be silently ignored.
- Reject relative paths that traverse above the working directory.

## Third-party Code

Files under `third_party/` are reference or vendor material. Do not edit them unless the change is explicitly about vendored/reference assets.
