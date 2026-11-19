#!/usr/bin/env python3
"""TODO(team): Verify binary assets and vendor exemptions without weakening source TODO checks."""
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
import check_scaffold as scaffold


class ScaffoldTest(unittest.TestCase):
    def test_binary_assets_are_excluded_but_sources_remain_checked(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root/'architecture.png').write_bytes(b'\x89PNG\r\n\x1a\n')
            (root/'main.tex').write_text('\\documentclass{article}\n')
            with patch.object(scaffold, 'ROOT', root):
                files = scaffold.source_files()
                self.assertEqual(files, [root/'main.tex'])
                self.assertEqual(scaffold.purpose(files[0]), '')

    def test_only_explicit_vendor_paths_are_exempt(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            vendor = root/'docs/checkpoint/usenix-2020-09.sty'
            vendor.parent.mkdir(parents=True)
            vendor.write_text('% upstream template\n')
            custom = vendor.with_name('custom.sty')
            custom.write_text('% local style without purpose\n')
            with patch.object(scaffold, 'ROOT', root):
                self.assertIn('TODO', scaffold.purpose(vendor))
                self.assertEqual(scaffold.purpose(custom), '')
                self.assertEqual(vendor.read_text(), '% upstream template\n')


if __name__ == '__main__':
    unittest.main()
