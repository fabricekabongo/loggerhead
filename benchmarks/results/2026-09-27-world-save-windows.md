# Isolated World Save sample, 2026-09-27

Raw output: [2026-09-27-world-save-windows.txt](2026-09-27-world-save-windows.txt).

- Source commit: `ec840cf0fdedd01362e40ee59254b1ea192e76f3` (the later report update does not change the measured Go code).
- Command: `GOMAXPROCS=1 go test ./world -run '^$' -bench '^BenchmarkWorldSave$' -benchmem -benchtime=10000x -cpu=1 -count=5`.
- Go: `go1.26.0 windows/amd64`; CPU: AMD Ryzen 7 4800H, 8 cores / 16 logical processors; physical RAM: 33,675,218,944 bytes.
- Insert fixture: seed 2401, namespace `benchmark`, 10,000 unique IDs, seed/index/axis SHA-256-derived coordinates in latitude `[-89,89)` and longitude `[-179,179)`. Each repetition creates a new World; all inserted keys, coordinates, spatial membership, and the final population are checked after timing.
- Local update fixture: seed 2402, namespace `benchmark`, 256 preloaded IDs with the same deterministic coordinate derivation within latitude `[12,12.1)` and longitude `[25,25.1)`. Ten thousand updates reuse those IDs with deterministic local movement; final coordinates, spatial membership, and unchanged population are checked after timing.
- Both cases measure engine `Save` calls on prepared inputs. The reported B/op is allocation during the timed calls, not retained memory per object. `ns/op` is engine operation time in these sequential cases, not TCP request latency.

These are bounded local samples for this partial #24 slice. They do not establish a capacity limit or a cross-host network result. The broader workload matrix and comparable release baselines remain open.
