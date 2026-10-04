# Contributing

Read [CODING_STANDARD.md](CODING_STANDARD.md) before changing code. Run `go test ./...` and the architecture check described below. Run `go test -race ./...` for concurrency changes. Record commands and results in the pull request.

The architecture checker is [go-arch-lint v1.19.0](https://github.com/fe3dback/go-arch-lint/releases/tag/v1.19.0). Install that exact release and run:

```text
go-arch-lint check --project-path . --arch-file .go-arch-lint.yml
```

The CI workflow downloads the Linux amd64 release and verifies its SHA-256 digest before execution. The release's Windows amd64 ZIP has SHA-256 `f73aaefeca181a0b39f70ae4bf8c0a4d099a0ab5814140c36930ffe181fb456c`. This tool needs Go 1.25 to build from source, so use a release binary with Loggerhead's Go 1.24 toolchain. The checked-in config covers handwritten production packages and excludes only tests and vendored code.

The Go lint gate runs golangci-lint v2.14.0 with the curated checks and formatters in `.golangci.yml`. It reports findings introduced by each pull request or push to `main`; fix new findings without excluding production code or raising configured thresholds.

The CRAP gate runs go-crap v0.5.1 over fresh atomic coverage profiles from the exact base and head commits. On Linux, use `bash scripts/check-crap.sh BASE_SHA HEAD_SHA OUTPUT_DIR` from a clean head checkout, with full 40-character commit SHAs and a task-owned output directory. The script validates the pinned tool against golden fixtures, then retains coverage profiles, full JSON reports, a manifest, and per-function policy results. The GitHub Actions `Go CRAP policy` check runs the same command. This gate covers default build tags and handwritten production packages reached by `go test ./...`; the nested `quality/go-crap-fixtures` module supplies only golden tool tests.
