#!/usr/bin/env python3
"""TODO(Yuquan): Extend the muxed fMP4 representation only with matching timeline tests."""
import argparse
from bisect import bisect_right
import json
import math
import mmap
from pathlib import Path
import struct
import subprocess


def boxes(data):
    """Yield validated top-level BMFF boxes without copying the entire movie."""
    offset = 0
    while offset < len(data):
        if offset + 8 > len(data):
            raise ValueError("truncated MP4 box")
        size, kind = struct.unpack_from(">I4s", data, offset)
        header = 8
        if size == 1:
            if offset + 16 > len(data):
                raise ValueError("truncated extended MP4 box")
            size = struct.unpack_from(">Q", data, offset + 8)[0]
            header = 16
        elif size == 0:
            size = len(data) - offset
        if size < header or offset + size > len(data):
            raise ValueError("invalid MP4 box length")
        yield offset, offset + size, kind
        offset += size


def fragment(source, directory):
    directory = Path(directory)
    directory.mkdir(parents=True, exist_ok=True)
    if any(directory.glob("segment-*.mp4")):
        raise ValueError("output already contains segments; use a new directory")
    probe = json.loads(subprocess.check_output([
        "ffprobe", "-v", "error", "-show_streams", "-of", "json", str(source)
    ], text=True))
    video = next(stream for stream in probe["streams"] if stream["codec_type"] == "video")
    if video["codec_name"] != "h264":
        raise ValueError("expected H.264 video")
    audio = [stream for stream in probe["streams"] if stream["codec_type"] == "audio"]
    if any(stream["codec_name"] != "aac" for stream in audio):
        raise ValueError("expected AAC audio")
    with open(source, "rb") as file, mmap.mmap(file.fileno(), 0, access=mmap.ACCESS_READ) as data:
        structure = list(boxes(data))
        initialization = b"".join(data[start:end] for start, end, kind in structure if kind in (b"ftyp", b"moov"))
        if b"mvex" not in initialization:
            raise ValueError("input is not fragmented MP4")
        avcc = initialization.find(b"avcC")
        if avcc < 0 or initialization[avcc + 4] != 1:
            raise ValueError("missing AVC decoder configuration")
        codecs = ["avc1." + initialization[avcc + 5:avcc + 8].hex()]
        if audio:
            codecs.append("mp4a.40.2")
        offsets = []
        paths = []
        for i, (start, end, kind) in enumerate(structure):
            if kind != b"moof":
                continue
            if i + 1 >= len(structure) or structure[i + 1][2] != b"mdat":
                raise ValueError("fragment must have an adjacent mdat")
            end = structure[i + 1][1]
            offsets.append(start)
            path = directory / f"segment-{len(paths):06d}.mp4"
            # Identical initialization makes each stored object independently
            # verifiable/seekable. tfdt timestamps are never reset or rewritten.
            path.write_bytes(initialization + data[start:end])
            paths.append(path)
    if not paths:
        raise ValueError("no media fragments")
    starts = [None] * len(paths)
    last_end = 0.0
    command = ["ffprobe", "-v", "error", "-show_packets", "-show_entries",
               "packet=stream_index,pts_time,duration_time,pos,flags", "-of", "compact=p=0:nk=0", str(source)]
    with subprocess.Popen(command, stdout=subprocess.PIPE, text=True) as process:
        for line in process.stdout:
            fields = dict(part.split("=", 1) for part in line.strip().split("|") if "=" in part)
            if fields.get("pts_time", "N/A") == "N/A":
                continue
            pts = float(fields["pts_time"])
            duration = float(fields.get("duration_time", "0").replace("N/A", "0"))
            last_end = max(last_end, pts + duration)
            if int(fields["stream_index"]) == video["index"]:
                index = bisect_right(offsets, int(fields["pos"])) - 1
                if index < 0:
                    raise ValueError("video sample precedes its fragment")
                if starts[index] is None:
                    if "K" not in fields.get("flags", ""):
                        raise ValueError("fragment does not begin with a video keyframe")
                    starts[index] = pts
        if process.wait():
            raise ValueError("ffprobe packet scan failed")
    if any(start is None for start in starts):
        raise ValueError("fragment without video samples")
    # Metadata partitions the shared timeline; encoded packet timestamps remain exact.
    boundaries = [0] + [round(value * 1000) for value in starts[1:]] + [math.ceil(last_end * 1000)]
    segments = []
    for i, path in enumerate(paths):
        duration = boundaries[i + 1] - boundaries[i]
        if duration <= 0:
            raise ValueError("nonmonotonic fragment timeline")
        segments.append({"file": path.name, "start_ms": boundaries[i], "duration_ms": duration})
    info = {"representation": "continuous-fmp4-h264-aac-v1" if audio else "continuous-fmp4-h264-v1",
            "mime_type": 'video/mp4; codecs="' + ", ".join(codecs) + '"',
            "duration_ms": boundaries[-1], "segments": segments}
    (directory / "stream.json").write_text(json.dumps(info, indent=2) + "\n")
    return info


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("directory", type=Path)
    arguments = parser.parse_args()
    fragment(arguments.source, arguments.directory)
