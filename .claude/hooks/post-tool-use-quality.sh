#!/usr/bin/env bash
set -uo pipefail

payload="$(cat)"
project_dir="${CLAUDE_PROJECT_DIR:-}"
if [[ -z "$project_dir" ]]; then
  project_dir="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
fi
export GOCACHE="${GOCACHE:-$project_dir/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$project_dir/.cache/go-mod}"

if ! command -v jq >/dev/null 2>&1; then
  echo "jq not found; skip Claude quality hook input parsing" >&2
  exit 0
fi

file_path="$(printf '%s' "$payload" | jq -r '.tool_response.filePath // .tool_response.file_path // .tool_input.file_path // .tool_input.path // empty' 2>/dev/null || true)"
if [[ -z "$file_path" ]]; then
  exit 0
fi

if [[ "$file_path" = /* ]]; then
  abs_path="$file_path"
else
  abs_path="$project_dir/$file_path"
fi

if [[ ! -e "$abs_path" ]]; then
  exit 0
fi

rel_path="$abs_path"
case "$abs_path" in
  "$project_dir"/*) rel_path="${abs_path#"$project_dir"/}" ;;
esac

status=0

go_tool_available() {
  local tool="$1"
  (cd "$project_dir" && go tool "$tool" --help >/dev/null 2>&1)
}

case "$abs_path" in
  *.go)
    if command -v gofmt >/dev/null 2>&1; then
      gofmt -w "$abs_path" || status=1
    fi
    ;;
  *.json)
    jq empty "$abs_path" >/dev/null || status=1
    ;;
esac

if [[ "$rel_path" == "api/openapi.yaml" ]]; then
  if go_tool_available vacuum; then
    (cd "$project_dir" && go tool vacuum lint -d --fail-severity warn -r vacuum-ruleset.yaml api/openapi.yaml) || status=1
  else
    echo "go tool vacuum not available; skip api/openapi.yaml lint" >&2
  fi
fi

if go_tool_available gitleaks; then
  (cd "$project_dir" && go tool gitleaks dir --redact --no-banner "$abs_path") || status=1
else
  echo "go tool gitleaks not available; skip changed-file secret scan" >&2
fi

case "$rel_path" in
  lefthook.yml|vacuum-ruleset.yaml|.lefthook/*)
    if go_tool_available lefthook; then
      (cd "$project_dir" && go tool lefthook validate) || status=1
    else
      echo "go tool lefthook not available; skip lefthook validate" >&2
    fi
    ;;
esac

exit "$status"
