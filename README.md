# wire-unused

Finds provider sets and providers in a
[Google Wire](https://github.com/google/wire) `wire.Build()` call that
nothing in the injector depends on.

In a large `wire.Build()` with many sets pulled in from other packages,
it's tedious to work out which sets can be dropped. wire-unused reports
all of them in one pass.

## Install

```sh
go install github.com/tandetat/wire-unused@latest
```

Requires Go 1.23 or newer.

## Usage

Pass a package pattern or a directory. Files with the `wireinject` build
tag are loaded automatically.

```sh
wire-unused ./...
wire-unused ./cmd/server
```

Example output:

```
Analyzing: InitializeDeps
  wire.Build() at /src/app/wire_injectors.go:17

  1 provider sets, 2 standalone providers

  Unused provider sets:
    (none)

  Unused providers:
    - pkg_b.NewServiceB
        provides: *pkg_b.ServiceB
        /src/app/wire_injectors.go:19
```

A set is reported only when none of its providers are needed. If one
provider in a set is used, the set stays.

## How it works

1. Load the packages with full type information.
2. Find every function that calls `wire.Build()` and sort its arguments
   into provider sets, standalone provider functions, `wire.Struct` and
   `wire.Bind`.
3. Follow each provider set to its `wire.NewSet()` definition,
   recursing into nested sets, and record each provider's input and
   output types.
4. Starting from the fields of each `wire.Struct` target, walk the
   inputs backwards, following `wire.Bind` from interface to concrete
   type. Every provider reached is needed.
5. Report the sets and standalone providers that were never reached.

## Limitations

- The walk starts only from `wire.Struct` targets. An injector that
  returns a type built by a provider, like
  `func InitApp() (*App, error)` with `NewApp` in the build, has no
  starting point, so every set gets reported as unused.
- `wire.Value`, `wire.InterfaceValue` and `wire.FieldsOf` are ignored.
- Exits with status 0 even when it finds unused providers, so it can't
  fail a CI job on its own yet.

## Development

```sh
go test ./...
```

Each test fixture under `testdata/` is a small Go module with its own
`wire_injectors.go`.

## License

MIT. See [LICENSE](LICENSE).
