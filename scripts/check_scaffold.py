#!/usr/bin/env python3
"""TODO(Haoyu): Maintain the per-file purpose index and audit owner TODO markers when extending the scaffold."""
import argparse
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EXCLUDED = {"node_modules", "dist", "bin", ".data", "__pycache__", ".git", ".venv", ".DS_Store", ".local-reference", ".pnpm-store"}


def source_files():
    return sorted(path for path in ROOT.rglob("*") if path.is_file()
                  and not any(part in EXCLUDED for part in path.relative_to(ROOT).parts)
                  and ("results" not in path.relative_to(ROOT).parts or path.name == "README.md"))


def purpose(path):
    if path.name == "go.sum":
        return "TODO(Haoyu)：Go 自动管理的依赖完整性校验；格式禁止添加用途注释，本索引代为说明。"
    text = path.read_text()
    return next((line.strip().replace("|", "/").replace("`", "") for line in text.splitlines() if "TODO" in line), "")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write-index", action="store_true")
    args = parser.parse_args()
    files = source_files()
    missing = [str(path.relative_to(ROOT)) for path in files if path.name != "FILE_INDEX.md" and not purpose(path)]
    if missing:
        raise SystemExit("Missing purpose TODO: " + ", ".join(missing))
    if args.write_index:
        lines = ["# 文件用途与 TODO 索引", "", "TODO(Haoyu)：新增文件后运行 python3 scripts/check_scaffold.py --write-index 更新本表。", "",
                 "每个手写源文件均有用途 TODO。go.sum 是格式受限的自动校验文件，不可插入注释；其用途在下表说明。pnpm-lock.yaml 支持注释，因此保留文件头 TODO。编译产物、依赖目录、缓存和运行数据不纳入源文件要求。", "",
                 "| 文件 | 用途 / 后续负责人 |", "|---|---|"]
        for path in files:
            if path.name != "FILE_INDEX.md":
                relative = path.relative_to(ROOT).as_posix()
                lines.append(f"| [{relative}]({relative}) | {purpose(path)} |")
        (ROOT / "FILE_INDEX.md").write_text("\n".join(lines) + "\n")
    print(json.dumps({"source_files": len(source_files()), "missing_todo": missing}, ensure_ascii=False))


if __name__ == "__main__":
    main()
