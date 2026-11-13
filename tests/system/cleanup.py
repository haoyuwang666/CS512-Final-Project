#!/usr/bin/env python3
"""TODO(Yuquan): Extend cleanup acceptance with disk failure injection."""
import json
from cluster import Cluster,wait,ing

def main():
    with Cluster(extra=1000000, repair=False) as c:
        m=c.fixture(count=1);sid=m['segments'][0]['segment_id']
        assert len(c.call(f'/segments/{sid}/locations')['locations'])==3
        plan=c.call('/videos/fixture/cleanup'); assert len(plan['tasks'])==1
        t=plan['tasks'][0];target=t['target']['peer_id']
        c.stop(target)
        c.call('/videos/fixture/cleanup','POST',plan)
        assert len(c.call(f'/segments/{sid}/locations')['locations'])==2
        leader=c.call('/cluster')['leader_url'];key=next(x['id'] for x in c.members if x['http_url']==leader)
        c.stop(key)
        wait(lambda:c.call('/maintenance')['tasks'][t['task_id']]['state']=='pending')
        c.start(target)
        wait(lambda:c.call('/maintenance')['tasks'][t['task_id']]['state']=='completed')
        stats=ing.request(t['target']['url']+'/api/v1/stats')
        assert stats['stored_bytes']==0
        assert not ing.request(t['target']['url']+'/api/v1/inventory')['receipts']
        c.stop(target);c.start(target)
        wait(lambda:ing.request(t['target']['url']+'/api/v1/health'))
        assert not ing.request(t['target']['url']+'/api/v1/inventory')['receipts']
        assert c.call('/videos/fixture/cleanup')['tasks']==[]
        for p in c.call(f'/segments/{sid}/locations')['locations']:
            ing.request(p['url']+f'/api/v1/segments/{sid}/receipt')
        print(json.dumps({'status':'PASS','checks':['reviewed plan','offline target retry','leader failover resumes cleanup','two retained replicas','deleted data stays deleted after restart'],'stats':stats}))
def balanced_cleanup():
    with Cluster(extra=1000000, repair=False) as c:
        manifest=c.fixture(count=6)
        plan=c.call('/videos/fixture/cleanup')
        assert len(plan['tasks'])==6
        from collections import Counter
        targets=Counter(t['target']['peer_id'] for t in plan['tasks'])
        assert sorted(targets.values())==[2,2,2], targets
        c.call('/videos/fixture/cleanup','POST',plan)
        wait(lambda:all(c.call('/maintenance')['tasks'][t['task_id']]['state']=='completed' for t in plan['tasks']))
        sizes=[]
        for peer in c.peers:
            stats=ing.request(peer['url']+'/api/v1/stats')
            assert stats['objects']==4, stats
            sizes.append(stats['stored_bytes'])
        assert len(set(sizes))==1, sizes
        for seg in manifest['segments']:
            locations=c.call('/segments/'+seg['segment_id']+'/locations')['locations']
            assert len(locations)==2
            for peer in locations:
                ing.request(peer['url']+'/api/v1/segments/'+seg['segment_id']+'/receipt')
        print(json.dumps({'status':'PASS','checks':['balanced cleanup plan','equal physical bytes for distinct equal-sized segments','two verified replicas per segment'],'bytes_per_peer':sizes}))

if __name__=='__main__':
    main()
    balanced_cleanup()
