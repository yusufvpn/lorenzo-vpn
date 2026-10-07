import urllib.request
import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

token = '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'
headers = {
    'User-Agent': 'Mozilla/5.0',
    'Authorization': f'token {token}'
}

url = 'https://api.github.com/repos/yusufvpn/lorenzo-vpn/actions/runs'
req = urllib.request.Request(url, headers=headers)

try:
    with urllib.request.urlopen(req) as resp:
        data = json.loads(resp.read().decode('utf-8'))
        runs = data.get('workflow_runs', [])
        for run in runs[:5]:
            run_id = run.get('id')
            status = run.get('status')
            conclusion = run.get('conclusion')
            html_url = run.get('html_url')
            name = run.get('name')
            print(f"Run {run_id}: {name} - Status: {status}, Conclusion: {conclusion}")
            print(f"URL: {html_url}")
            
            # Fetch jobs for this run
            jobs_url = f"https://api.github.com/repos/yusufvpn/lorenzo-vpn/actions/runs/{run_id}/jobs"
            jreq = urllib.request.Request(jobs_url, headers=headers)
            with urllib.request.urlopen(jreq) as jresp:
                jdata = json.loads(jresp.read().decode('utf-8'))
                for job in jdata.get('jobs', []):
                    print(f"  Job: {job.get('name')} -> {job.get('status')}, {job.get('conclusion')}")
                    for step in job.get('steps', []):
                        if step.get('conclusion') == 'failure':
                            print(f"    FAILED STEP: {step.get('name')} (Conclusion: {step.get('conclusion')})")
            print("-" * 50)
except Exception as e:
    print('Error:', e)
