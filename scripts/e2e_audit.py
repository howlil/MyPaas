#!/usr/bin/env python3
import os
import sys
import time
import json
import urllib.request
import urllib.error

# An automated script to audit the live MyPaas system

def e2e_audit():
    # 1. Load API Token and URL
    token = os.environ.get("MYPAAS_API_TOKEN")
    if not token:
        print("Error: MYPAAS_API_TOKEN environment variable is not set.")
        print("Please grab your API token from the MyPaas dashboard and run:")
        print("export MYPAAS_API_TOKEN=\"mp_...\"")
        print("python3 scripts/e2e_audit.py")
        sys.exit(1)

    api_url = os.environ.get("MYPAAS_API_URL", "http://localhost:8080")
    print(f"Starting MyPaas Automated Audit against {api_url}")
    print("-" * 50)

    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }

    def api_request(method, endpoint, data=None):
        url = f"{api_url}{endpoint}"
        req = urllib.request.Request(url, method=method, headers=headers)
        if data:
            req.data = json.dumps(data).encode("utf-8")
        try:
            with urllib.request.urlopen(req) as response:
                return json.loads(response.read().decode("utf-8")), response.status
        except urllib.error.HTTPError as e:
            return {"error": e.read().decode("utf-8")}, e.code
        except Exception as e:
            return {"error": str(e)}, 500

    # 1. Test Auth & Settings
    print("[1/5] Testing Authentication & Admin Settings...")
    res, status = api_request("GET", "/admin/settings")
    if status != 200:
        print(f"❌ Failed to fetch settings: {res}")
        sys.exit(1)
    print("✅ Authentication successful. Admin settings accessible.")

    # 2. Test Project Creation (Static)
    print("[2/5] Creating a test static project...")
    project_payload = {
        "name": "audit-test-static",
        "repo_url": "https://github.com/nabilrn/mypaas-sample-static",
        "branch": "main"
    }
    res, status = api_request("POST", "/projects", project_payload)
    if status != 201:
        print(f"❌ Failed to create project: {res}")
        sys.exit(1)
    
    project_id = res["data"]["id"]
    print(f"✅ Project created successfully with ID: {project_id}")

    # 3. Test Deployment Engine
    print("[3/5] Triggering deployment for the test project...")
    res, status = api_request("POST", f"/projects/{project_id}/deployments")
    if status not in (201, 202):
        print(f"❌ Failed to trigger deployment: {res}")
        sys.exit(1)
    
    deployment_id = res["data"]["id"]
    print(f"✅ Deployment triggered with ID: {deployment_id}")
    
    print("      Waiting for deployment to finish (polling every 5s)...")
    max_retries = 30
    success = False
    for i in range(max_retries):
        time.sleep(5)
        d_res, d_status = api_request("GET", f"/projects/{project_id}/deployments")
        if d_status == 200:
            latest = d_res.get("data", [])
            if len(latest) > 0:
                current_status = latest[0]["status"]
                print(f"      Status: {current_status}")
                if current_status == "running":
                    success = True
                    break
                elif current_status in ("failed", "stopped"):
                    print(f"❌ Deployment failed with status: {current_status}")
                    break
    
    if not success:
        print("❌ Deployment did not finish successfully within the time limit.")
    else:
        print("✅ Deployment completed and is now running.")

    # 4. Test Monitoring / Metrics
    if success:
        print("[4/5] Fetching runtime metrics...")
        m_res, m_status = api_request("GET", f"/projects/{project_id}/metrics")
        if m_status == 200:
            print("✅ Metrics fetched successfully:")
            print("  ", m_res)
        else:
            print(f"❌ Failed to fetch metrics: {m_res}")

    # 5. Cleanup
    print("[5/5] Cleaning up audit resources...")
    del_res, del_status = api_request("DELETE", f"/projects/{project_id}")
    if del_status == 204:
        print("✅ Test project deleted successfully.")
    else:
        print(f"❌ Failed to delete test project: {del_res}")

    print("-" * 50)
    print("🎉 Automated Audit Complete!")

if __name__ == "__main__":
    e2e_audit()
