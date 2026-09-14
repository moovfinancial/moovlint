# moovlint — custom Go analyzers for Moov engineering conventions

Custom [golangci-lint module plugin](https://golangci-lint.run/docs/plugins/module-plugins/) that enforces Moov-specific Go coding standards.

## Analyzers

| Analyzer | Status | Description |
|---|---|---|
| `spanevents` | shipping | Detects `logger.Info().Log()`/`logger.Warn().Log()` calls in service/repo code and suggests `telemetry.AddEvent` or `telemetry.RecordError`. |
| `spanrequired` | shipping | Checks exported methods on service structs taking `context.Context` have a `telemetry.StartSpan` call. Advisory severity while false-positive rate is calibrated. |
| `spanlifecycle` | shipping | Checks that spans created with `telemetry.StartSpan` or `StartLinkedRootSpan` are ended with `defer span.End()`. |
| `spancontext` | shipping | Detects `End()` or `SetName()` calls on spans retrieved from context via `trace.SpanFromContext`. |
| `mockcheck` | shipping | Detects test replacements passed to same-module interfaces, including embedded-interface overrides across packages. Retains the same-package `mock*`/`fake*`/`stub*` check. Client interfaces are allowed by default; the exclusion is configurable. |
| `validationflag` | shipping | Checks that `Validate() error` methods wrap `mvalidation.ValidateStruct` returns with `errors.Flag(..., errors.NotValid)`. |
| `grpcstatus` | shipping | Checks that gRPC handler methods return errors through `GrpcErrorStatus`. |
| `grpcserver` | shipping | Checks that gRPC controller structs embed their generated `Unimplemented*Server` type. |
| `httpdecodeflag` | shipping | Checks that HTTP request body decode errors are wrapped with `errors.Flag(..., errors.NotSerializable)`. |
| `midusage` | shipping | Detects `mid.MustParseID` outside test files and direct equality comparisons on `mid.ID`; use `Equals`. |
| `oteltags` | shipping | Checks that `otel` struct tags use lower snake case and do not include `omitempty`; flags map/slice-of-struct/nested types. |
| `controllerassert` | shipping | Checks that HTTP controller structs with `AppendRoutes` have a compile-time interface assertion. |
| `repoerrorflags` | advisory | Checks that repository methods flag expected database errors (AlreadyExists→NotUnique, NotFound→NotFound) with the correct `errors.Flag`. |
| `timeinject` | shipping | Detects `time.Now()` calls in service methods that have a `stime.TimeService` field on their receiver, and `time.Now` passed as a clock value instead of an injected clock. |
| `contextcancel` | shipping | Checks that `context.WithCancel`/`WithTimeout`/`WithDeadline` results have a corresponding `defer cancel()`. |
| `nolintguard` | shipping | Checks that `//nolint` directives target a specific linter and include an explanation. |
| `blankdiscard` | shipping | Detects `_ =` blank discards of error and `sql.Result` returns without an inline justification comment. |
| `uuidgen` | shipping | Detects `uuid.New*` used for ID generation in mid-based services; requires `mid.NewRandomID` so entity IDs carry their type. |
| `requiregoroutine` | shipping | Detects `require.*` and `t.Fatal`/`FailNow` calls inside goroutine closures (go statements, httptest handlers, callbacks) in test files. |
| `spanname` | shipping | Checks span names passed to `telemetry.StartSpan`/`StartLinkedRootSpan`/`SetName` are lower-kebab-case. |
| `testsleep` | advisory | Detects `time.Sleep` used for synchronization in test files; suggests `require.Eventually` or an injected clock. |
| `logformat` | shipping | Detects `%w` verbs in Moov logger format strings; wrapping verbs are only valid in `fmt.Errorf`. |
| `moneyfloat` | shipping | Detects float types used for monetary values (fields and params named amount, balance, fee, or total). |
| `spanerrors` | advisory | Checks that functions which create a span record returned errors with `telemetry.RecordError` before returning. |
| `mapderef` | advisory | Detects `m[k].Field` dereferences on maps of pointers or interfaces without a comma-ok check. |
| `subtestassert` | shipping | Detects assertion objects created from the outer test's `t` used inside `t.Run` closures. |
| `ctornilguard` | advisory | Checks exported `New*` constructors nil-check pointer and interface dependencies before storing them. Also flags method-level checks of dependencies validated by a private implementation's constructor. |
| `enumcast` | advisory | Detects unchecked conversions of raw strings to enum-like named string types outside validation and mapper functions. |
| `fixtureplacement` | opt-in | Flags test helpers that build same-module data models outside configured fixture packages. |
| `modelplacement` | opt-in | Flags exported request, response, and row models in service or repository files. File and type conventions are configurable. |

### Configurable checks

Placement checks are disabled by default. Enable them for a review pass before
adding them to CI. Once enabled, findings fail the command like other analyzers;
the Go analysis API has no separate warning severity.

```sh
moovlint -fixtureplacement -fixtureplacement.enabled \
  -modelplacement -modelplacement.enabled ./...
```

The CLI also accepts `-mockcheck.allow-interfaces`,
`-fixtureplacement.fixture-packages`, `-modelplacement.model-files`,
`-modelplacement.implementation-files`, and `-modelplacement.model-types`.
Each value is a Go regular expression. File patterns match base names;
package patterns match import paths.

Set the same options in the custom plugin configuration:

```yaml
linters:
  enable:
    - moovlint
  settings:
    custom:
      moovlint:
        type: module
        settings:
          mockcheck:
            allow-interfaces: 'Client$'
          fixtureplacement:
            enabled: true
            fixture-packages: '(^|/)(fixtures|testfixtures|testutil)(/|$)'
          modelplacement:
            enabled: true
            model-files: '^models?(_.*)?\.go$'
            implementation-files: '^(service|repository|repo)(_.+)?\.go$'
            model-types: '(Request|Response|Row|Record|Result|Query|Outcome|Page)$'
```

`mockcheck.allow-interfaces` matches `import/path.Interface`. Use a specific
boundary name to permit another external dependency, or `^$` to allow none.
Cross-package matching needs module metadata from the analyzer driver. Without
it, the check uses same-package interfaces only. Forwarding wrappers that only
embed an interface are not test replacements unless they override a method.

Placement checks skip generated code, unexported types, service implementations,
and structs with behavior other than `Validate`. Fixture helpers must return
one populated model, with an optional error. Helpers that also return a test
environment are excluded. Model placement checks only package-level declarations.

The method-level constructor check uses direct returns of private struct literals
after an early nil-error guard. Other construction paths, dependency writes, or
an exposed field address suppress the finding. It does not infer contracts from
complex constructor control flow.

## Repository checks

Non-Go file checks run via `moovlint repo [path]`:

| Checker | Description |
|---|---|
| `migrations` | Sequential naming, no `IF NOT EXISTS`, no direct renames, no `NOT NULL` without `DEFAULT`, and empty-string `CHECK` constraints on key `TEXT NOT NULL` columns. |
| `structtags` | JSON tags use camelCase, preserve `ID` casing, timestamps use `On`/`At` suffix. |
| `protobuf` | Field numbers are permanent and unique; gaps between numbers are reserved; PII-named fields carry the `sensitive` option or a Non-PII justification comment. |
| `sqlnow` | No `now()`/`CURRENT_TIMESTAMP` in sqlc query files; pass a timestamp parameter so values are testable. |
| `gomodreplace` | `replace` directives in `go.mod` carry a tracking ticket comment (`// TODO(LINEAR-123)`). |

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
2. Register it in `analyzers.go`, shared by the plugin and CLI
3. Add testdata under `testdata/<name>/` with `// want` comments
4. `make test`

## Registering in a repo

```yaml
# .custom-gcl.yml
version: v2.11.4
plugins:
  - module: 'github.com/moovfinancial/moovlint'
    version: v0.2.0  # use path: /path/to/moovlint to test unreleased analyzers

# .golangci.yml
linters:
  enable:
    - moovlint
  settings:
    custom:
      moovlint:
        type: module
        description: Moov engineering conventions
        settings: {}
```

The new placement checks and the mock/constructor extensions require a build
from this source until the next release. For a separate moovlint-only config,
run `./custom-gcl run --config=.golangci-moovlint.yml ./...` so subpackages are
included. Build the custom binary with a Go toolchain at least as new as the
target module's Go version.
