#!/usr/bin/env python3
"""External package against real Core storage, with no desktop/network mutations."""
import json, os, shutil, subprocess, sys, tempfile
from pathlib import Path
binary=Path(sys.argv[1]).resolve();root=Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory() as tmp:
    base=Path(tmp);package=base/'package'
    shutil.copytree(root,package,ignore=shutil.ignore_patterns('.git','test-results','__pycache__'))
    env={**os.environ,'XDG_DATA_HOME':str(base/'data'),'XDG_STATE_HOME':str(base/'state')}
    def call(*args,ok=True):
        p=subprocess.run([str(binary),*map(str,args)],env=env,text=True,capture_output=True)
        assert (p.returncode==0)==ok,p.stdout+p.stderr
        return json.loads(p.stdout)
    id='io.github.tcballard.github-contributions'
    call('validate',package);call('install',package)
    a=call('create',id,'small')['updated'];b=call('create',id,'large')['updated'];assert a!=b
    def entry(instance):return next(e for e in call('list')['installed'] if e['instanceId']==instance)
    def granted():return next(e for e in call('list')['catalog'] if e['packageId']==id)['githubAllowed']
    assert not granted()
    call('github-permission',id,'allow');assert granted()
    for instance,name in [(a,'tcballard'),(b,'octocat')]:
        call('save',instance,json.dumps({'revision':entry(instance)['placement']['revision'],'settings':{'username':name,'palette':'Theme accent'}}))
    call('save',a,json.dumps({'revision':entry(a)['placement']['revision'],'settings':{'username':12,'palette':'Theme accent'}}),ok=False)
    assert entry(a)['placement']['settings']['username']=='tcballard'
    assert entry(b)['placement']['settings']['username']=='octocat'
    call('hide',a);assert entry(b)['placement']['enabled']
    call('github-permission',id,'deny');assert not granted()
    call('github-permission',id,'allow')
    m=json.loads((package/'widget.json').read_text());m['version']='0.0.2';(package/'widget.json').write_text(json.dumps(m))
    call('update',package);assert not granted()
    assert entry(b)['placement']['settings']['username']=='octocat'
    call('github-permission',id,'allow');assert granted()
    call('rollback',id)
    assert entry(a)['placement']['settings']['username']=='tcballard'
    assert entry(b)['placement']['settings']['username']=='octocat'
print('PASS: standalone validation/install, independent instances, rejected settings, hide, revoke, update/re-grant and rollback')
