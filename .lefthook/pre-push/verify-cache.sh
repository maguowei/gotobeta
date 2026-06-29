#!/usr/bin/env bash
set -euo pipefail

cache_schema_version="v1"
cache_enabled="${LEFTHOOK_VERIFY_CACHE:-1}"
force_verify="${LEFTHOOK_VERIFY_FORCE:-0}"
# 生成项目默认 make verify（generate/tidy/lint/secrets/vuln/coverage/build）；
# 生成器仓库自身默认 make smoke（生成+验证多个子项目，更强门禁）。
# 两者语义不同，无需统一。
verify_command="${LEFTHOOK_VERIFY_COMMAND:-make verify}"

project_root="$(git rev-parse --show-toplevel)"
cd "$project_root"

workspace_is_clean() {
  [[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]]
}

run_verify() {
  echo "pre-push verify: running ${verify_command}"
  bash -c "$verify_command" && return 0
  local status="$?"
  echo "pre-push verify: command failed with status ${status}" >&2
  exit "$status"
}

if [[ "$cache_enabled" == "0" || "$cache_enabled" == "false" ]]; then
  echo "pre-push verify: cache disabled by LEFTHOOK_VERIFY_CACHE=${cache_enabled}"
  run_verify
  exit 0
fi

if ! workspace_is_clean; then
  echo "pre-push verify: 工作区存在未提交或未跟踪变更，跳过缓存读取和写入"
  run_verify
  exit 0
fi

head_commit="$(git rev-parse HEAD)"
{ read -r go_version; read -r goos; read -r goarch; } < <(go env GOVERSION GOOS GOARCH)
cache_dir="${LEFTHOOK_VERIFY_CACHE_DIR:-$project_root/.cache/lefthook/pre-push}"
cache_key="$(
  printf '%s\n%s\n%s\n%s\n%s\n' \
    "$cache_schema_version" \
    "$head_commit" \
    "$go_version" \
    "$goos" \
    "$goarch" |
  git hash-object --stdin
)"
cache_file="$cache_dir/$cache_key.ok"

if [[ "$force_verify" != "1" && "$force_verify" != "true" && -f "$cache_file" ]]; then
  echo "pre-push verify: 命中缓存，跳过已验证提交 ${head_commit}"
  exit 0
fi

run_verify

if ! workspace_is_clean; then
  echo "pre-push verify: verify 后工作区出现变更，本次不写入缓存"
  exit 0
fi

mkdir -p "$cache_dir"
tmp_file="$cache_file.$$"
{
  printf 'schema=%s\n' "$cache_schema_version"
  printf 'head=%s\n' "$head_commit"
  printf 'go_version=%s\n' "$go_version"
  printf 'goos=%s\n' "$goos"
  printf 'goarch=%s\n' "$goarch"
  printf 'verified_at=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
} > "$tmp_file"
mv "$tmp_file" "$cache_file"
echo "pre-push verify: 已缓存提交 ${head_commit} 的成功检查结果"
