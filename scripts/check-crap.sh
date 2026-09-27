#!/usr/bin/env bash
set -euo pipefail

if (( $# != 3 )); then
  echo 'usage: scripts/check-crap.sh BASE_SHA HEAD_SHA OUTPUT_DIR' >&2
  exit 2
fi

base_sha=$1
head_sha=$2
output_dir=$3
if [[ ! $base_sha =~ ^[0-9a-f]{40}$ || ! $head_sha =~ ^[0-9a-f]{40}$ ]]; then
  echo 'base and head must be full commit SHAs' >&2
  exit 2
fi

repo_root=$(git rev-parse --show-toplevel)
if [[ -n $(git -C "$repo_root" status --porcelain) ]]; then
  echo 'CRAP analysis requires a clean checkout at the exact head commit' >&2
  exit 2
fi
if [[ $(git -C "$repo_root" rev-parse HEAD) != "$head_sha" ]]; then
  echo "checkout HEAD does not match requested head $head_sha" >&2
  exit 2
fi
if [[ $(go env GOOS) != linux || $(go env GOARCH) != amd64 || $(go env CGO_ENABLED) != 1 ]]; then
  echo 'CRAP comparison requires Linux amd64 with CGO_ENABLED=1 to match the declared build scope' >&2
  exit 2
fi
git -C "$repo_root" cat-file -e "$base_sha^{commit}"

mkdir -p "$output_dir"
output_dir=$(cd "$output_dir" && pwd)
# A failed retry must never publish reports left by an earlier invocation.
rm -f -- "$output_dir"/base.cover "$output_dir"/base.json \
  "$output_dir"/head.cover "$output_dir"/head.json \
  "$output_dir"/manifest.txt "$output_dir"/policy.txt
scratch=$(mktemp -d)
cleanup() { rm -rf -- "$scratch"; }
trap cleanup EXIT

archive="$scratch/go-crap.tar.gz"
curl --fail --location --silent --show-error --retry 2 --max-time 60 \
  'https://github.com/padiazg/go-crap/releases/download/v0.5.1/go-crap_0.5.1_linux_amd64.tar.gz' \
  --output "$archive"
echo '5d1dff5bfc5cbc89022efa09aebc79887e3100bbbcb2bddec9b80285e6c5bdfc  '"$archive" | sha256sum --check
tar -xzf "$archive" -C "$scratch" go-crap
chmod +x "$scratch/go-crap"
pwsh -NoProfile -File "$repo_root/quality/go-crap-fixtures/verify.ps1" -GoCrap "$scratch/go-crap"

printf 'base_sha=%s\nhead_sha=%s\ngo_crap=v0.5.1\ngo_crap_archive_sha256=%s\ncoverage_mode=atomic\nscope=go_test_./..._default_build_tags\n' \
  "$base_sha" "$head_sha" '5d1dff5bfc5cbc89022efa09aebc79887e3100bbbcb2bddec9b80285e6c5bdfc' > "$output_dir/manifest.txt"
go version >> "$output_dir/manifest.txt"
go env GOOS GOARCH CGO_ENABLED >> "$output_dir/manifest.txt"

for side in base head; do
  if [[ $side == base ]]; then
    sha=$base_sha
  else
    sha=$head_sha
  fi
  source_dir="$scratch/$side"
  mkdir -p "$source_dir"
  git -C "$repo_root" archive "$sha" | tar -xf - -C "$source_dir"
  (
    cd "$source_dir"
    go test -covermode=atomic -coverprofile="$output_dir/$side.cover" ./...
    "$scratch/go-crap" scan ./... --coverage-profile="$output_dir/$side.cover" \
      --format=json --no-progress --timeout=5m --output="$output_dir/$side.json"
  )
  test -s "$output_dir/$side.cover"
  test -s "$output_dir/$side.json"
  grep -Fxq 'mode: atomic' "$output_dir/$side.cover"
done

"$scratch/go-crap" version >> "$output_dir/manifest.txt"
sha256sum "$output_dir"/{base,head}.{cover,json} >> "$output_dir/manifest.txt"
(
  cd "$repo_root"
  go run ./tools/crapgate --base-report "$output_dir/base.json" --head-report "$output_dir/head.json" \
    --base-ref "$base_sha" --head-ref "$head_sha"
) | tee "$output_dir/policy.txt"
