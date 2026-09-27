# Isolated World Save sample, 2026-09-27

Raw output: [2026-09-27-world-save-windows.txt](2026-09-27-world-save-windows.txt).

- Source commit: `5483f862dfde1e891b7c1a6103cacd565ffb9ab2` (the later report-only commit does not change the measured Go code).
- Command: `GOMAXPROCS=1 go test ./world -run '^$' -bench '^BenchmarkWorldSave$' -benchmem -benchtime=10000x -cpu=1 -count=5`.
- Go: `go1.26.0 windows/amd64`; CPU: AMD Ryzen 7 4800H, 8 cores / 16 logical processors; physical RAM: 33,675,218,944 bytes.
- Insert fixture: seed 2401, namespace `benchmark`, 10,000 unique IDs, seeded coordinates in latitude `[-89,89)` and longitude `[-179,179)`. Each repetition creates a new World; all inserted keys, coordinates, and the final population are checked after timing.
- Local update fixture: seed 2402, namespace `benchmark`, 256 preloaded IDs within latitude `[12,12.1)` and longitude `[25,25.1)`. Ten thousand updates reuse those IDs with deterministic local movement; final coordinates and unchanged population are checked after timing.
- Both cases measure engine `Save` calls on prepared inputs. The reported B/op is allocation during the timed calls, not retained memory per object. `ns/op` is engine operation time in these sequential cases, not TCP request latency.

These are bounded local samples for this partial #24 slice. They do not establish a capacity limit or a cross-host network result. The broader workload matrix and comparable release baselines remain open.
