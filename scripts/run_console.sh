#!/usr/bin/env bash
# TODO(Haoyu): Provide one local entry point while keeping tools in the user's own installation.
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$HOME/.local/bin:/usr/local/go/bin:/opt/homebrew/bin:/usr/local/bin:$PATH"
for tool in go node pnpm python3 ffmpeg ffprobe; do
  if ! command -v "$tool" >/dev/null; then
    echo "缺少工具 $tool，请安装后重新运行 make console。" >&2
    exit 1
  fi
done
pnpm --dir frontend install --frozen-lockfile
make backend frontend
exec ./bin/local-console
