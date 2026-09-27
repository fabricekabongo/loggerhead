# Coding standard

Follow the [Google Go Style Guide](https://google.github.io/styleguide/go/guide) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). This document adds Loggerhead-specific rules. Correctness and safe failure come before measured performance.

## Changes and tests

For behavior changes, demonstrate a failing test at the public interface, make the smallest fix, and record the passing test. Use small coherent changes. Test success, invalid input, failure, and recovery when they apply. Prefer deterministic in-process tests; add a focused protocol or network test when the behavior crosses that boundary. Check errors and propagate useful context. Do not panic for malformed client input or expected operational failure.

Use `go test -race` for concurrency changes. A passing race check does not prove spatial correctness: after writers stop, verify `(namespace, id)` locations, population, and quadtree membership. Keep instrumented correctness runs separate from performance baselines.

## Resource ownership

The creator of each goroutine, connection, timer, and background task owns its shutdown. Blocking work needs cancellation or deadlines. Bound worker counts, retries, queues, buffers, query results, and memory growth. Long-lived loops need cancellation and bounded work per iteration. Document shared-state ownership and lock order; never copy mutex-bearing values or accept unsynchronized access. Do not add unbounded recursion. The existing quadtree needs a bounded-depth contract or a reviewed migration exception before its recursion changes.

## Package dependencies

Keep package dependencies aligned with `.go-arch-lint.yml`. `main` is the composition root and may wire all packages. `admin` may use `clustering` and `config`; `clustering` may use `config`, `query`, and `world`; `query` may use `world`; `server` may use `query`. `config` and `world` stay independent of other Loggerhead packages. Do not add reverse or unlisted package dependencies. New packages need an explicit component and allowed dependency rule.

Preserve caller-supplied `(namespace, id)` identity and the documented TCP protocol. Explain local exceptions in code review. Performance claims need corrected, reproducible benchmarks with raw results and configuration metadata.

## Quality limits

New functions need cyclomatic complexity at most 10 and unrounded CRAP at most 10. Changed functions at or below CRAP 10 stay at or below 10; legacy functions above 10 improve or use a named, reviewed exception without worsening. Changed production code needs at least 80% mutation efficacy and 80% mutant coverage, with every survivor reviewed. These measures supplement behavioral tests and race checks. CI tooling and the baseline policy for these limits are being added under issue #30; do not treat this document alone as enforcement.
