#!/usr/bin/env python3
"""TODO(Yuquan): Share isolated real-process fixtures without attaching to user clusters."""
import importlib.util
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

ROOT=Path(__file__).resolve().parents[2]
spec=importlib.util.spec_from_file_location('ingest',ROOT/'scripts/ingest/ingest.py')
ing=importlib.util.module_from_spec(spec); spec.loader.exec_module(ing)

def wait(action,timeout=45):
    deadline=time.monotonic()+timeout; last=None
    while time.monotonic()<deadline:
        try:
            result=action()
            if result: return result
        except (Exception,) as error: last=error
        time.sleep(.15)
    raise AssertionError(f'condition timed out: {last}')

def ports(n):
    sockets=[socket.socket() for _ in range(n)]
    try:
        for s in sockets:s.bind(('127.0.0.1',0))
        return [s.getsockname()[1] for s in sockets]
    finally:
        for s in sockets:s.close()

class Cluster:
    def __init__(self,policy='fixed',controllers=3,repair=True,extra=0,rates=None):
        self.temp=tempfile.TemporaryDirectory(prefix='cs512-repair-'); self.path=Path(self.temp.name)
        # Reserve the entire topology together: independent batches can reuse
        # a just-released ephemeral port before any process has bound it.
        allocated=ports(3+2*controllers)
        pp=allocated[:3];hp=allocated[3:3+controllers];rp=allocated[3+controllers:]
        self.peers=[{'peer_id':f'peer-{i+1}','url':f'http://127.0.0.1:{p}'} for i,p in enumerate(pp)]
        self.members=[{'id':f'controller-{i+1}','http_url':f'http://127.0.0.1:{p}','raft_address':f'127.0.0.1:{rp[i]}'} for i,p in enumerate(hp)]
        self.controllers=[m['http_url'] for m in self.members]; self.processes={}; self.config={}; self.logs=[]
        rates=rates or [10000000]*3
        for i,p in enumerate(self.peers):
            self.config[p['peer_id']]=('peer',{'PEER_ID':p['peer_id'],'HTTP_ADDR':f'127.0.0.1:{pp[i]}','DATA_DIR':str(self.path/p['peer_id']),'UPLOAD_BYTES_PER_SECOND':str(rates[i]),'QUOTA_BYTES':'16777216','CONTROLLERS':','.join(self.controllers)})
        for i,m in enumerate(self.members):
            self.config[m['id']]=('controller',{'NODE_ID':m['id'],'HTTP_ADDR':f'127.0.0.1:{hp[i]}','RAFT_BIND':m['raft_address'],'RAFT_MEMBERS':json.dumps(self.members),'PEERS':json.dumps(self.peers),'DATA_DIR':str(self.path/m['id']),'BOOTSTRAP':str(i==0).lower(),'PLACEMENT_POLICY':policy,'REPAIR_ENABLED':str(repair).lower(),'REPAIR_GRACE':'1s','EXTRA_REPLICA_BYTES':str(extra)})
    def start(self,key):
        if key in self.processes and self.processes[key].poll() is None: raise RuntimeError('process already running')
        binary,extra=self.config[key]; log=open(self.path/f'{key}.log','ab'); self.logs.append(log)
        self.processes[key]=subprocess.Popen([str(ROOT/'bin'/binary)],cwd=ROOT,env={**os.environ,**extra},stdout=log,stderr=log)
    def stop(self,key):
        p=self.processes[key]
        if p.poll() is None:
            p.kill(); p.wait(timeout=10)
    def call(self,path,method='GET',value=None):return ing.controller(self.controllers,path,method,value)
    def __enter__(self):
        try:
            for key in self.config:self.start(key)
            for p in self.peers: wait(lambda p=p:ing.request(p['url']+'/api/v1/health'))
            wait(lambda:self.call('/cluster').get('leader_url'))
            return self
        except BaseException:
            self.__exit__(*__import__('sys').exc_info()); raise
    def __exit__(self,kind,error,tb):
        for key in self.processes:self.stop(key)
        for log in self.logs:log.close()
        if error:
            for log in self.path.glob('*.log'):print(log.name,log.read_text(errors='replace')[-2000:])
        self.temp.cleanup()
    def fixture(self,count=4,duration=150,size=32768,video='fixture'):
        import hashlib
        directory=self.path/video;directory.mkdir();segments=[]
        for i in range(count):
            data=(hashlib.sha256(f'{video}-{i}'.encode()).digest()*((size+31)//32))[:size]; sid=f'{video}--{i:06d}'
            (directory/f'{sid}.bin').write_bytes(data)
            segments.append({'segment_id':sid,'index':i,'duration_ms':duration,'size_bytes':size,'sha256':hashlib.sha256(data).hexdigest()})
        path=directory/'manifest.json';path.write_text(json.dumps({'schema_version':1,'video_id':video,'representation':'synthetic-bytes-v1','segments':segments}))
        ing.ingest(path,self.controllers);return json.loads(path.read_text())
