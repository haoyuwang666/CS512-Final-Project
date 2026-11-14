#!/usr/bin/env python3
"""TODO(Yuquan): Add timeline fixtures when supporting additional BMFF layouts."""
import struct
import unittest

from fragment_mp4 import validate_decode_timeline


def box(kind, payload):
    return struct.pack(">I4s", 8 + len(payload), kind) + payload


def fragment(start, durations=None, count=2, default=512):
    tfhd = box(b"tfhd", struct.pack(">III", 0x20008, 1, default))
    tfdt = box(b"tfdt", struct.pack(">IQ", 0x1000000, start))
    if durations is None:
        trun = struct.pack(">II", 0, count)
    else:
        trun = struct.pack(">II", 0x100, len(durations)) + b"".join(struct.pack(">I", d) for d in durations)
    return box(b"moof", box(b"traf", tfhd + tfdt + box(b"trun", trun)))


class DecodeTimelineTests(unittest.TestCase):
    def test_initial_alignment_uses_actual_duration(self):
        # FFprobe 7.1 reports nominal 512 even when the first stored video
        # sample lasts 775 ticks to align the muxed audio/video start.
        data = fragment(0, [775, 512]) + fragment(1287)
        self.assertEqual(validate_decode_timeline(data), {1: [(0, 1287), (1287, 2311)]})

    def test_rejects_real_gap_and_overlap(self):
        for start in (1023, 1025):
            with self.subTest(start=start), self.assertRaisesRegex(ValueError, "gap/overlap"):
                validate_decode_timeline(fragment(0) + fragment(start))

    def test_reads_trex_default_and_tfdt_version_zero(self):
        trex = box(b"trex", struct.pack(">IIIIII", 0, 1, 1, 1024, 0, 0))
        init = box(b"moov", box(b"mvex", trex))
        tfhd = box(b"tfhd", struct.pack(">II", 0x20000, 1))
        tfdt = box(b"tfdt", struct.pack(">II", 0, 0))
        trun = box(b"trun", struct.pack(">II", 0x201, 2) + struct.pack(">iII", 80, 100, 120))
        data = init + box(b"moof", box(b"traf", tfhd + tfdt + trun))
        self.assertEqual(validate_decode_timeline(data), {1: [(0, 2048)]})

    def test_rejects_missing_or_zero_duration(self):
        for data in (fragment(0, default=0), fragment(0, [512, 0])):
            with self.subTest(data=data), self.assertRaises(ValueError):
                validate_decode_timeline(data)

    def test_rejects_truncated_sample_table(self):
        tfhd = box(b"tfhd", struct.pack(">III", 8, 1, 512))
        tfdt = box(b"tfdt", struct.pack(">II", 0, 0))
        trun = box(b"trun", struct.pack(">III", 0x100, 2, 512))
        with self.assertRaisesRegex(ValueError, "sample table"):
            validate_decode_timeline(box(b"moof", box(b"traf", tfhd + tfdt + trun)))


if __name__ == "__main__":
    unittest.main()
