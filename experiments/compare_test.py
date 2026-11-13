#!/usr/bin/env python3
"""TODO(Yuquan): Extend metric validation when paired reports gain new statistics."""
import json
from pathlib import Path
import tempfile
import unittest
from compare import compare

class PairingTests(unittest.TestCase):
    def rows(self):
        return [dict(group=g,seed=seed,trace_sha256=str(seed),initial_peers=[{'stored_bytes':10}],completion_rate=rate,mean_completed_rebuffering_ratio=None,uploaded_bytes=100) for seed in [1,2] for g,rate in [('fixed-round_robin',1),('adaptive-round_robin',.5)]]
    def test_failure_rates_remain_in_paired_difference(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'runs.jsonl';path.write_text(''.join(json.dumps(row)+'\n' for row in self.rows()))
            result=json.loads(compare(path).read_text());row=next(row for row in result if row['metric']=='completion_rate')
            self.assertEqual(row['mean_difference'],-.5);self.assertEqual(row['pairs'],2);self.assertEqual(row['bootstrap_95_percent_interval'],[-.5,-.5])
    def test_unmatched_trace_is_rejected(self):
        rows=self.rows();rows[1]['trace_sha256']='different'
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'runs.jsonl';path.write_text(''.join(json.dumps(row)+'\n' for row in rows))
            with self.assertRaises(ValueError):compare(path)

if __name__=='__main__':unittest.main()
