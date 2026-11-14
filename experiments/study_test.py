#!/usr/bin/env python3
"""Regression checks for honest timing, failed probes and censored results."""
import http.server
import json
from pathlib import Path
import socket
import tempfile
import threading
import time
import unittest
from study_common import probe
from repair_playback_study import summarize, matrix, trace_for

CFG=json.loads((Path(__file__).parent/'scenarios/compact-repair.json').read_text())

class StudyTests(unittest.TestCase):
    def test_matrix_reuse_and_matching(self):
        cells=matrix(CFG)
        self.assertEqual(len(cells),72)
        self.assertEqual(len({tuple(c[k] for k in ('case','load','policy','grace','outage','delay','seed')) for c in cells}),72)
        counts={e:sum(e in c['experiments'] for c in cells) for e in ('E1','E2','E3','E4')}
        self.assertEqual(counts,{'E1':18,'E2':18,'E3':24,'E4':18})
        for seed in CFG['seeds']:
            traces=[trace_for(c) for c in cells if c['seed']==seed and c['load']=='high']
            self.assertTrue(all(t==traces[0] for t in traces))

    def test_failed_probe_is_not_recovery(self):
        sock=socket.socket();sock.bind(('127.0.0.1',0));port=sock.getsockname()[1];sock.close()
        result=probe(f'http://127.0.0.1:{port}',timeout=.1)
        self.assertFalse(result['ok']);self.assertIsNone(result['status'])
        self.assertGreaterEqual(result['ended'],result['started'])

    def test_delay_and_truncated_transfer(self):
        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                time.sleep(.08)
                self.send_response(200);self.send_header('Content-Length','100');self.end_headers();self.wfile.write(b'abc');self.close_connection=True
            def log_message(self,*args):pass
        server=http.server.ThreadingHTTPServer(('127.0.0.1',0),Handler)
        thread=threading.Thread(target=server.serve_forever);thread.start()
        try:
            result=probe(f'http://127.0.0.1:{server.server_port}',binary=True)
            self.assertFalse(result['ok']);self.assertGreaterEqual(result['ended']-result['started'],.07)
        finally:server.shutdown();server.server_close();thread.join()

    def test_censoring_open_stall_and_instance_accounting(self):
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp);cell=dict(case='peer',load='low',policy='L',grace=3,outage='persistent',delay=0,seed=1101,experiments=['E1'])
            def journal(name,rows):(path/(name+'.jsonl')).write_text(''.join(json.dumps(r)+'\n' for r in rows))
            journal('injections',[{'kind':'first_failure','observer_s':5}])
            journal('tasks-and-copy-verification',[{'observer_s':6,'copies':{'s':['p2']}},{'observer_s':8,'copies':{'s':None}}])
            journal('client-events',[{'viewer':0,'type':'stall_started','client_launch_s':0,'since_start_ms':4000,'observer_s':4}])
            journal('peer-traffic-and-policy',[{'ok':True,'peer_id':'p2','body':{'instance_id':i,'class_bytes':{'repair':b}}} for i,b in [('one',100),('one',200),('two',50)]])
            sessions=[{'viewer':0,'status':'stopped','ended_s':12}]
            row=summarize(path,cell,CFG,{'affected':['s'],'failed_peer':'p1'},sessions)
            self.assertIsNone(row['recovery_s']);self.assertTrue(row['recovery_censored'])
            self.assertEqual(row['traffic_bytes']['repair'],250)
            self.assertEqual(row['mean_post_fault_stall_s_all_sessions'],3.5)
            self.assertEqual(row['completion_rate'],0)
            self.assertGreater(row['unknown_copy_byte_seconds'],0)
            self.assertFalse(row['post_fault_exposure'])

if __name__=='__main__':unittest.main()
