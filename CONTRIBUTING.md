# Contributing

Read [CODING_STANDARD.md](CODING_STANDARD.md) before changing code. Run `go test ./...` and the architecture check described below. Run `go test -race ./...` for concurrency changes. Record commands and results in the pull request.

The architecture checker is [go-arch-lint v1.19.0](https://github.com/fe3dback/go-arch-lint/releases/tag/v1.19.0). Install that exact release and run:

```text
go-arch-lint check --project-path . --arch-file .go-arch-lint.yml
```

The CI workflow downloads the Linux amd64 release and verifies its SHA-256 digest before execution. The release's Windows amd64 ZIP has SHA-256 `f73aaefeca181a0b39f70ae4bf8c0a4d099a0ab5814140c36930ffe181fb456c`. This tool needs Go 1.25 to build from source, so use a release binary with Loggerhead's Go 1.24 toolchain. The checked-in config covers handwritten production packages and excludes only tests and vendored code.

The Go lint gate runs golangci-lint v2.14.0 with the curated checks and formatters in `.golangci.yml`. It reports findings introduced by each pull request or push to `main`; fix new findings without excluding production code or raising configured thresholds.
