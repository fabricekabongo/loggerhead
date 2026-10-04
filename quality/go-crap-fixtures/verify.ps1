param([string] $GoCrap)

$ErrorActionPreference = 'Stop'
$fixtureDir = $PSScriptRoot
$repoRoot = Split-Path (Split-Path $fixtureDir -Parent) -Parent
if (-not $GoCrap) {
    $GoCrap = Join-Path (Split-Path $repoRoot -Parent) '.tools\go-crap-v0.5.1\go-crap.exe'
}
if (-not (Test-Path -LiteralPath $GoCrap)) {
    throw "Pinned go-crap binary not found: $GoCrap"
}
$previousGoCache = $env:GOCACHE
$tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd([IO.Path]::DirectorySeparatorChar)
$scratch = Join-Path $tempRoot ('loggerhead-crap-fixtures-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $scratch | Out-Null
$env:GOCACHE = Join-Path $scratch 'gocache'

Push-Location $fixtureDir
try {
    function Invoke-FixtureScan([string] $name, [string] $runPattern) {
        $profile = Join-Path $scratch "$name.cover"
        $report = Join-Path $scratch "$name.json"
        & go test -covermode=atomic -coverprofile $profile -run $runPattern ./... 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "go test failed for $name (exit $LASTEXITCODE)" }
        & $GoCrap scan ./... --coverage-profile $profile --format json --no-progress --output $report 2>&1 | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "go-crap failed for $name (exit $LASTEXITCODE)" }
        return Get-Content -Raw -LiteralPath $report | ConvertFrom-Json
    }

    function Get-Entry($document, [string] $functionName) {
        $entry = @($document.entries | Where-Object function -eq $functionName)
        if ($entry.Count -ne 1) { throw "Expected one report entry for $functionName; found $($entry.Count)" }
        return $entry[0]
    }

    function Assert-Near([string] $label, [double] $actual, [double] $expected) {
        if ([Math]::Abs($actual - $expected) -gt 0.000001) {
            throw "$label expected $expected, got $actual"
        }
    }

    $zero = Invoke-FixtureScan 'zero' '^$'
    $partial = Invoke-FixtureScan 'partial' '^TestScorePartial$'
    $full = Invoke-FixtureScan 'full' '^TestScoreFull$|^TestCounterAdd$'

    $zeroScore = Get-Entry $zero 'Score'
    $partialScore = Get-Entry $partial 'Score'
    $fullScore = Get-Entry $full 'Score'
    foreach ($score in @($zeroScore, $partialScore, $fullScore)) {
        if ($score.cyclomatic -ne 5) { throw "Score complexity expected 5, got $($score.cyclomatic)" }
    }
    Assert-Near 'Score coverage at 0%' $zeroScore.coverage 0
    Assert-Near 'Score CRAP at 0%' $zeroScore.crap 30
    Assert-Near 'Score coverage at 75%' $partialScore.coverage 75
    Assert-Near 'Score CRAP at 75%' $partialScore.crap 5.390625
    Assert-Near 'Score coverage at 100%' $fullScore.coverage 100
    Assert-Near 'Score CRAP at 100%' $fullScore.crap 5

    $receiver = Get-Entry $full 'Counter.Add'
    if ($receiver.receiver -ne 'Counter' -or $receiver.file -ne 'fixture.go') {
        throw "Receiver/path mismatch: $($receiver | ConvertTo-Json -Compress)"
    }
    $uncovered = Get-Entry $full 'Uncovered'
    Assert-Near 'Uncovered function coverage' $uncovered.coverage 0
    Assert-Near 'Uncovered function CRAP' $uncovered.crap 6

    $partialProfile = Get-Content -LiteralPath (Join-Path $scratch 'partial.cover')
    if (-not ($partialProfile | Where-Object { $_ -match 'fixture\.go:4\.\d+,\d+\.\d+ 5 1$' })) {
        throw 'Expected Score prelude statements to share a five-statement coverage block'
    }

    'go-crap v0.5.1 fixture checks passed: CC=5 CRAP=30 / 5.390625 / 5; receiver, path, multi-statement block, and uncovered-function checks passed.'
}
finally {
    Pop-Location
    $env:GOCACHE = $previousGoCache
    $resolved = (Resolve-Path -LiteralPath $scratch).Path
    if (-not $resolved.StartsWith($tempRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to remove scratch directory outside $tempRoot`: $resolved"
    }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}
