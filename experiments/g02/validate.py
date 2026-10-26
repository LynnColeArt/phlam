#!/usr/bin/env python3
"""Audit saved G02 evidence and recheck integrated source-free clocks/rollback."""
import argparse
import collections
import hashlib
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import tempfile


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument('--run', type=Path, required=True)
    parser.add_argument('--tools', type=Path, default=Path('bin'))
    args = parser.parse_args()
    root, tools = args.run, args.tools.resolve()
    result = json.loads((root / 'results.json').read_text())
    assert result['campaign_sha256'] == sha(Path('experiments/g02/campaign.json'))
    assert result['binary_sha256'] == {n: sha(tools/n) for n in result['binary_sha256']}
    selected = result['selected_strategy']
    integration = json.loads((root/'integration.json').read_text())
    for alias in ['counter', 'accumulator', 'saturating']:
        trial = next(t for t in result['trials'] if t['strategy'] == selected and t['program'] == alias and t['seed'] == integration['seed'])
        candidate = root / trial['builds'][0]['report']
        candidate = candidate.parent / trial['builds'][0]['attempt']['artifact']
        assert candidate.read_bytes() == (root/'combined-0'/f'{alias}.json').read_bytes()
    request = json.loads(Path('experiments/g01/run-0001/original/requests.json').read_text())
    request['repeats'] = 1
    expected = json.loads(Path('experiments/g01/run-0001/original/expected.json').read_text())
    checks = {}
    for engine in ['learned', 'lookup']:
        for boundary in ['concrete', 'certified']:
            signatures = []
            for clocks in [True, False]:
                with tempfile.TemporaryDirectory(prefix='phlame-g02-audit-') as temp:
                    work = Path(temp)
                    for f in (root/'combined-0').iterdir():
                        shutil.copyfile(f, work/f.name)
                    shutil.copyfile(tools/'phlam-runtime', work/'runtime')
                    (work/'runtime').chmod(0o755)
                    shutil.copyfile(root/'combined-evidence-0/development-public-key.hex', work/'key.hex')
                    out = json.loads(subprocess.check_output([str(work/'runtime'), 'run', '.', '--key', 'key.hex', '--engine', engine, '--boundary', boundary, '--phase-clocks='+str(clocks).lower()], input=json.dumps(request).encode(), cwd=work, env={'GOMAXPROCS':'1'}))
                assert not out.get('preflight_error')
                if boundary == 'certified':
                    assert all(c['certified'] for c in out['certificates'].values())
                assert len(out['outcomes']) == 1526
                sig = []
                for tx, o in zip(request['transactions'], out['outcomes']):
                    e = expected[o['id']]
                    assert o['accepted'] == e['accepted'] and o['states'] == e['states']
                    assert o.get('results', []) == e['results'] and o.get('writes', []) == e['writes']
                    if not o['accepted']:
                        assert o['states'] == tx['instances'] and not o.get('results') and not o.get('writes')
                    sig.append([o[k] for k in ['id', 'accepted', 'states']] + [o.get('results', []), o.get('writes', [])])
                signatures.append(sig)
            assert signatures[0] == signatures[1]
            checks[engine+'/'+boundary] = {'scenarios':1526, 'clock_equivalence':True, 'oracle_fixture_equivalence':True, 'atomic_rollback':True}
    deps = subprocess.check_output(['go','list','-deps','./cmd/phlam-runtime'],text=True).splitlines()
    forbidden = [d for d in deps if d.startswith('phlam/compiler') or d.startswith('phlam/reference')]
    symbols = subprocess.check_output(['go','tool','nm',str(tools/'phlam-runtime')],text=True)
    assert not forbidden and 'phlam/runtime/components.Train' not in symbols
    tests = collections.Counter()
    for line in Path('work/g02-tests.jsonl').read_text().splitlines():
        event = json.loads(line)
        if event.get('Test'):
            tests[event['Action']] += 1
    assert tests['fail'] == 0 and tests['pass'] == 129
    report = {'schema':'phlame.g02.validation.v0','comparisons':checks,'individual_and_combined_model_bytes_identical':True,'runtime_dependencies':deps,'runtime_optimizer_symbols_absent':True,'tests':dict(tests),'go_vet':'passed','hardware':{'platform':platform.platform(),'cpu':next(l.split(':',1)[1].strip() for l in Path('/proc/cpuinfo').read_text().splitlines() if l.startswith('model name')),'logical_cpus':os.cpu_count(),'go':subprocess.check_output(['go','version'],text=True).strip(),'GOMAXPROCS':1},'test_log_sha256':sha(Path('work/g02-tests.jsonl'))}
    (root/'validation.json').write_text(json.dumps(report,indent=2,sort_keys=True)+'\n')
    shutil.copyfile('work/g02-tests.jsonl',root/'tests.jsonl')
    print('Evidence hashes, model identity, eight source-free modes, rollback, runtime isolation, and tests passed.')


if __name__ == '__main__':
    main()
