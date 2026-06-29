#!/usr/bin/env bash
set -euo pipefail

msg_file="${1:-}"
if [[ -z "$msg_file" || ! -f "$msg_file" ]]; then
  echo "commit message file is required" >&2
  exit 1
fi

subject="$(sed -n '1p' "$msg_file")"
if [[ -z "$subject" ]]; then
  echo "commit message subject is required" >&2
  exit 1
fi

if [[ "$subject" == Merge\ * || "$subject" == Revert\ * ]]; then
  exit 0
fi

types='feat|fix|docs|style|refactor|test|chore|perf|ci|build|revert'
pattern="^(${types})(\([A-Za-z0-9._/-]+\))?!?: .+"
if [[ ! "$subject" =~ $pattern ]]; then
  echo "commit message must follow Conventional Commits, for example: feat: 增加订单查询接口" >&2
  exit 1
fi

description="$(printf '%s' "$subject" | sed -E 's/^[a-z]+(\([^)]+\))?!?: //')"
if [[ ${#description} -gt 100 ]]; then
  echo "commit message description must be 100 characters or fewer" >&2
  exit 1
fi

if [[ ! "$description" =~ [一-龥] ]]; then
  echo "commit message description must use Chinese" >&2
  exit 1
fi
