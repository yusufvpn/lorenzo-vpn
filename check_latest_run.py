import urllib.request
import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

token = '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'
headers = {
    'User-Agent': 'Mozilla/5.0',
    'Authorization': f'token {token}'
}

run_id = '37499064367'
url = f'https://api.github.com/repos/yusufvpn/lorenzo-vpn/actions/runs/{run_id}/jobs'
req = urllib.request.Request(url, headers=headers)

try:
    with urllib.request.urlopen(req) as resp:
        data = json.loads(resp.read().decode('utf-8'))
        job = data.get('jobs', [])[0]
        print("Job:", job.get('name'), "| Status:", job.get('status'), "| Conclusion:", job.get('conclusion'))
        for s in job.get('steps', []):
            print(f"  Step [{s.get('name')}]: status={s.get('status')}, conclusion={s.get('conclusion')}")
except Exception as e:
    print('Error:', e)
