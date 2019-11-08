# CNCF Conformance Certification Pull Request

## What the script does

1. Clone the fork of https://github.com/cncf/k8s-conformance of the fork owner.
2. Sync the fork with the upstream
3. Create a new random named branch
4. Modify files
   1. Scan https://console.cloud.google.com/storage/browser/k8s-conformance-gardener/ for conformance test result files (e2e.log and junit_01.xml) per provider and k8s version
   2. Filter previously scanned items, which are not certified yet by evaluating https://github.com/cncf/k8s-conformance
   3. Create/download files `PRODUCT.yaml` (describes the product), `README.md` (describes how to install), `e2e.log`, `junit_01.xml` for each provider and k8s version tuple that is missing
5. Commit and push changes


## How to use
Invoke the script as shown below. This will create a new branch on your k8s-conformance fork, which contains files for not yet k8s certified gardener provider versions.

    export FORK_ONWER=yourForkName
    python3 create_pr_k8s_conformance.py

Afterwards check the changes in your branch manually. If the changes are fine, create a pull request to the upstream repository https://github.com/cncf/k8s-conformance .