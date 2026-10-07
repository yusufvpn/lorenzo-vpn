import urllib.request
import json
import sys

sys.stdout.reconfigure(encoding='utf-8')

token = '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'
job_id = '112351172501'

class NoAuthRedirectHandler(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        new_req = super().redirect_request(req, fp, code, msg, headers, newurl)
        if new_req and 'Authorization' in new_req.headers:
            del new_req.headers['Authorization']
        return new_req

opener = urllib.request.build_opener(NoAuthRedirectHandler)

log_url = f"https://api.github.com/repos/yusufvpn/lorenzo-vpn/actions/jobs/{job_id}/logs"
req = urllib.request.Request(log_url, headers={
    'User-Agent': 'Mozilla/5.0',
    'Authorization': f'token {token}'
})

try:
    with opener.open(req) as resp:
        content = resp.read().decode('utf-8', errors='ignore')
        lines = content.splitlines()
        print("\n".join(lines[-80:]))
except Exception as e:
    print("Error:", e)
