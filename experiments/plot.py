#!/usr/bin/env python3
"""TODO(Tian): Replace this diagnostic SVG with run-level comparison plots and uncertainty for formal experiments."""
import argparse
import html
import json
from pathlib import Path


def plot(path):
    rows = [json.loads(line) for line in path.read_text().splitlines() if line.strip()]
    if not rows:
        raise ValueError("no session rows")
    height = 90 + 35 * len(rows)
    parts = [f'<svg xmlns="http://www.w3.org/2000/svg" width="900" height="{height}" viewBox="0 0 900 {height}">', '<rect width="100%" height="100%" fill="white"/>', '<text x="20" y="28" font-family="sans-serif">Diagnostic session rebuffering (not a controlled comparison)</text>']
    for i, row in enumerate(rows):
        y = 60 + i * 35
        label = html.escape(f"{row['run_id']} / {row['viewer']} / {row['status']}")
        ratio = max(0, min(1, row.get("rebuffering_ratio", 0)))
        parts.append(f'<text x="20" y="{y+15}" font-size="12" font-family="sans-serif">{label}</text><rect x="400" y="{y}" width="{400*ratio}" height="20" fill="#174d73"/><text x="{410+400*ratio}" y="{y+15}" font-size="12">{ratio:.2%}</text>')
    parts.append("</svg>")
    output = path.with_name("diagnostic.svg")
    output.write_text("\n".join(parts))
    return output


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("sessions", type=Path)
    print(plot(parser.parse_args().sessions))
