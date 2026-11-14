#!/usr/bin/env python3
"""TODO(Haoyu): Add partial-ingest orphan collection to deletion acceptance."""
import json
import urllib.error
from cluster import Cluster,wait,ing

def main():
    with Cluster(repair=False) as c:
        manifest=c.fixture(count=3)
        # Repeated uploads must not strand tasks built from an earlier receipt.
        # Confirm an alias too, so metadata and the canonical Peer receipt differ.
        for seg in manifest['segments']:
            sid=seg['segment_id']
            data=(c.path/'fixture'/f'{sid}.bin').read_bytes()
            for peer in c.call(f'/segments/{sid}/locations')['locations']:
                op=f"alias-{sid}-{peer['peer_id']}"
                receipt=ing.request(peer['url']+f'/api/v1/segments/{sid}','PUT',data,
                    {'X-Operation-ID':op,'X-Size-Bytes':str(len(data)),
                     'X-SHA256':seg['sha256'],'X-Representation-Version':'1'})
                c.call(f'/segments/{sid}/replicas','POST',{'operation_id':'confirm-'+op,'receipt':receipt})
        c.stop('peer-1')
        c.call('/videos/fixture/delete','POST',{})
        state=c.call('/videos/fixture/delete')
        assert state['state']=='DELETING' and state['total']==6,state
        c.call('/videos/fixture/delete','POST',{})
        try:
            c.call('/videos/fixture/manifest')
            raise AssertionError('deleting video is still playable')
        except RuntimeError as error:
            assert str(error).startswith(('404','409')),error
        leader=c.call('/cluster')['leader_url']
        key=next(m['id'] for m in c.members if m['http_url']==leader)
        c.stop(key)
        wait(lambda:c.call('/videos/fixture/delete')['state']=='DELETING')
        c.start('peer-1')
        wait(lambda:c.call('/videos/fixture/delete')['state']=='DELETED')
        c.call('/videos/fixture/delete','POST',{})
        for peer in c.peers:
            stats=ing.request(peer['url']+'/api/v1/stats')
            assert stats['objects']==0 and stats['stored_bytes']==0,stats
        c.stop('peer-1');c.start('peer-1')
        wait(lambda:ing.request(c.peers[0]['url']+'/api/v1/health'))
        assert ing.request(c.peers[0]['url']+'/api/v1/stats')['stored_bytes']==0
        print(json.dumps({'status':'PASS','checks':['equivalent PUT/confirmation aliases','durable delete intent','idempotent delete','manifest hidden','leader failover','offline peer resumes','all replicas and bytes deleted','peer restart']}))
if __name__=='__main__':main()
