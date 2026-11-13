#!/usr/bin/env python3
"""TODO(Yuquan): Extend repair acceptance with sustained network partitions and power-loss faults."""
import hashlib
import json
import urllib.request
import time
from cluster import Cluster,wait,ing

def main():
    with Cluster() as c:
        manifest=c.fixture(count=1);seg=manifest['segments'][0]; sid=seg['segment_id']
        wait(lambda:len(c.call('/maintenance')['peers'])==3)
        loc=c.call(f'/segments/{sid}/locations')['locations'];original=[p['peer_id'] for p in loc]
        target=next(p for p in c.peers if p['peer_id'] not in original)
        c.stop(original[0])
        pending=wait(lambda:next((t for t in (c.call('/maintenance')['tasks'] or {}).values() if t['state']=='pending'),None))
        leader=c.call('/cluster')['leader_url']; interrupted=next(m['id'] for m in c.members if m['http_url']==leader)
        c.stop(interrupted)
        def repaired():
            tasks=c.call('/maintenance')['tasks'] or {}
            return [t for t in tasks.values() if t['state']=='completed' and t['target']['peer_id']==target['peer_id']]
        task=wait(repaired)[0]
        assert task['task_id']==pending['task_id'], 'new leader must resume the committed task'
        c.start(interrupted)
        wait(lambda:ing.request(leader+'/api/v1/health'))
        # Both original holders are gone: bytes must now come from the repaired third Peer.
        c.stop(original[1])
        with urllib.request.urlopen(target['url']+f'/api/v1/segments/{sid}',timeout=5) as response:data=response.read()
        assert hashlib.sha256(data).hexdigest()==seg['sha256']
        leader=wait(lambda:c.call('/cluster').get('leader_url'));key=next(m['id'] for m in c.members if m['http_url']==leader)
        c.stop(key);wait(lambda:c.call('/maintenance')['tasks'][task['task_id']]['state']=='completed')
        c.start(key);wait(lambda:c.call('/maintenance')['tasks'][task['task_id']]['state']=='completed')
        wait(lambda:ing.request(leader+'/api/v1/health'))
        c.start(original[1]);wait(lambda:ing.request(next(p['url'] for p in c.peers if p['peer_id']==original[1])+'/api/v1/health'))
        # Lose quorum: no new durable task or completion may be accepted.
        for member in c.members[1:]:c.stop(member['id'])
        time.sleep(2)
        try:
            ing.request(c.controllers[0]+'/api/v1/tasks/'+task['task_id']+'/ack','POST',{'operation_id':'no-quorum','receipt':{}})
            raise AssertionError('ack succeeded without quorum')
        except __import__('urllib.error',fromlist=['HTTPError']).HTTPError as error:assert error.code==503
        print(json.dumps({'status':'PASS','checks':['heartbeat/inventory','automatic repair after grace','download after both original replicas stop','durable task across leader failover/restart','no-quorum ack rejection'],'task':task},indent=2))

if __name__=='__main__':main()
