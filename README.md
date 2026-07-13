# moovlint — custom Go analyzers for Moov engineering conventions

Custom [golangci-lint module plugin](https://golangci-lint.run/docs/plugins/module-plugins/) that enforces Moov-specific Go coding standards.

## Analyzers

| Analyzer | Status | Description |
|---|---|---|
| `spanevents` | shipping | Detects `logger.Info().Log()`/`logger.Warn().Log()` calls in service/repo code and suggests `telemetry.AddEvent` or `telemetry.RecordError`. |
| `spanrequired` | shipping | Checks exported methods on service structs taking `context.Context` have a `telemetry.StartSpan` call. Advisory severity while false-positive rate is calibrated. |
| `spanlifecycle` | shipping | Checks that spans created with `telemetry.StartSpan` or `StartLinkedRootSpan` are ended with `defer span.End()`. |
| `spancontext` | shipping | Detects `End()` or `SetName()` calls on spans retrieved from context via `trace.SpanFromContext`. |
| `mockcheck` | shipping | Detects hand-rolled `mock*`/`fake*`/`stub*` test structs that implement interfaces from their own Go package; uses `test.NewEnvironment`, `eventingtest`, or real services instead. |
| `validationflag` | shipping | Checks that `Validate() error` methods wrap `mvalidation.ValidateStruct` returns with `errors.Flag(..., errors.NotValid)`. |
| `grpcstatus` | shipping | Checks that gRPC handler methods return errors through `GrpcErrorStatus`. |
| `grpcserver` | shipping | Checks that gRPC controller structs embed their generated `Unimplemented*Server` type. |
| `httpdecodeflag` | shipping | Checks that HTTP request body decode errors are wrapped with `errors.Flag(..., errors.NotSerializable)`. |
| `midusage` | shipping | Detects `mid.MustParseID` usage outside test files. |
| `oteltags` | shipping | Checks that `otel` struct tags use lower snake case and do not include `omitempty`; flags map/slice-of-struct/nested types. |
| `controllerassert` | shipping | Checks that HTTP controller structs with `AppendRoutes` have a compile-time interface assertion. |
| `repoerrorflags` | advisory | Checks that repository methods flag expected database errors (AlreadyExists→NotUnique, NotFound→NotFound) with the correct `errors.Flag`. |
| `timeinject` | shipping | Detects `time.Now()` calls in service methods that have a `stime.TimeService` field on their receiver. |
| `contextcancel` | shipping | Checks that `context.WithCancel`/`WithTimeout`/`WithDeadline` results have a corresponding `defer cancel()`. |
| `nolintguard` | shipping | Checks that `//nolint` directives target a specific linter and include an explanation. |

## Repository checks

Non-Go file checks run via `moovlint repo [path]`:

| Checker | Description |
|---|---|
| `migrations` | Sequential naming, no `IF NOT EXISTS`, no direct renames, no `NOT NULL` without `DEFAULT`. |
| `structtags` | JSON tags use camelCase, preserve `ID` casing, timestamps use `On`/`At` suffix. |
| `protobuf` | Field numbers are permanent and unique; gaps between numbers are reserved. |

## Development

```
make check         # Run lint + test (CI gate)
make test          # Run analyzer tests (analysistest)
make build         # Compile everything
make custom-gcl    # Build custom golangci-lint binary with moovlint plugins
moovlint repo .    # Run repository-level checks (migrations, structtags, protobuf)
```

## Adding an analyzer

1. Create `analyzers/<name>/<name>.go` with an `analysis.Analyzer`
2. Register it in `plugin.go` (`BuildAnalyzers`) and `cmd/moovlint/main.go`
3. Add testdata under `testdata/<name>/` with `// want` comments
4. `make test`

## Registering in a repo

```yaml
# .custom-gcl.yml
version: v2.11.4
plugins:
  - module: 'github.com/moovfinancial/moovlint'
    version: v0.1.0  # or path: for local dev

# .golangci.yml
linters:
  settings:
    custom:
      moovlint:
        type: module
        description: Moov engineering conventions
        settings: {}
```
