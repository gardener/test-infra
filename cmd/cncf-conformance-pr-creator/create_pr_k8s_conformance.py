#!/usr/bin/env python3
import glob
import json
import os
import random
import re
import shutil
import string
import subprocess
import sys

import ccc.github
import google.cloud.storage
from google.cloud.exceptions import NotFound
import semver.version

import ctx
import gitutil
import github.util
import model


repo_name = 'k8s-conformance'
product_yaml_filename = "PRODUCT.yaml"
google_credentials_filename = os.path.dirname(os.path.realpath(__file__)) + '/' + 'google_credentials.json'
repo_path = os.environ['FORK_OWNER'] + '/' + repo_name
upstream_repo = 'https://github.com/cncf/k8s-conformance'
cfg_factory = ctx.cfg_factory()
github_cfg = cfg_factory.github('github_com')
github_api = ccc.github.github_api(github_cfg)
gh = github.util.GitHubRepositoryHelper(
    owner='cncf',
    name=repo_name,
    github_api=github_api,
)
repo = gh.repository

script_dir = os.path.dirname(os.path.realpath(__file__))
temlate_dir = script_dir + '/'
gs_bucket_name = 'k8s-conformance-gardener'
conformance_tests_passed_string = "0 Failed | 0 Pending"

try:
    contact_email_address = os.environ['CONTACT_EMAIL_ADDRESS']
except KeyError:
    print("Error: Environment variable CONTACT_EMAIL_ADDRESS not set. This is required for opening any conformance PR.")
    sys.exit(1)

provider_list = ['gce', 'aws', 'azure', 'openstack', 'alicloud', 'vsphere']
content_product_yaml = {}
content_product_yaml['gardener'] = """vendor: SAP
name: Gardener (https://github.com/gardener/gardener) shoot cluster deployed on {0}
version: {1}
website_url: https://gardener.cloud
repo_url: https://github.com/gardener/
documentation_url: https://github.com/gardener/documentation/wiki
product_logo_url: https://raw.githubusercontent.com/gardener/documentation/master/images/logo_w_saplogo.svg
type: installer
description: The Gardener implements automated management and operation of Kubernetes clusters as a service and aims to support that service on multiple Cloud providers.
contact_email_address: {2}"""
content_product_yaml['sap-cp'] = """vendor: SAP
name: Cloud Platform - Gardener (https://github.com/gardener/gardener) shoot cluster deployed on {0}
version: {1}
website_url: https://cloudplatform.sap.com/index.html
documentation_url: https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/
product_logo_url: https://www.sap.com/dam/application/shared/logos/sap-logo-svg.svg
type: hosted
description: The Gardener implements automated management and operation of Kubernetes clusters as a service and aims to support that service on multiple Cloud providers.
contact_email_address: {2}"""


class QuitHack(object):  # shortcut-hack for quitting with a simple 'q'
    def __repr__(self):
        print('q -> quick exit')
        sys.exit(0)


q = QuitHack()


def get_gardener_version():
    gardener_version_file = os.environ['LANDSCAPE_COMPONENTS_HOME'] + '/gardener/VERSION'
    try:
        with open(gardener_version_file, 'r') as gardener_version_file_reader:
            gardener_version = gardener_version_file_reader.read()
    except IOError:
        print("Error: File '" + gardener_version_file + "' does not appear to exist.")
        sys.exit(1)
    return gardener_version


def id_generator(size=4, chars=string.ascii_uppercase + string.digits):
    return ''.join(random.choice(chars) for _ in range(size))


def cloneForkedRepo():
    gitHelper = gitutil.GitHelper.clone_into(repo_name, github_cfg, repo_path)
    print('INFO: Cloned ' + repo_path + ' repository into ' + repo_name + ' directory')
    return gitHelper


def syncForkAndUpstream(gitHelper):
    # wd = os.getcwd()
    os.chdir(repo_name + "/")
    subprocess.run(["git", "checkout", "master"])
    subprocess.run(["git", "remote", "add", "upstream", upstream_repo])
    subprocess.run(["git", "remote", "-v"])
    subprocess.run(["git", "fetch", "upstream"])
    subprocess.run(["git", "rebase", "upstream/master"])
    gitHelper.push("@", "refs/heads/master")


def createNewBranch(product_name, provider_name, k8s_version):
    random_id = id_generator()
    branch_name_random = product_name + "-" + provider_name + "-" + k8s_version + "-" + random_id
    subprocess.run(["git", "checkout", "-b", branch_name_random])
    return branch_name_random


def inplace_change(filename, old_string, new_string):
    # Safely read the input filename using 'with'
    with open(filename) as f:
        s = f.read()
        if old_string not in s:
            print('"{old_string}" not found in {filename}.'.format(**locals()))
            return

    # Safely write the changed content, if found in the file
    with open(filename, 'w') as f:
        print('Changing "{old_string}" to "{new_string}" in {filename}'.format(**locals()))
        s = s.replace(old_string, new_string)
        f.write(s)


def activate_google_application_credentials(cfg_factory: model.ConfigFactory):
    config = cfg_factory._cfg_element(cfg_type_name='gcp',
                                      cfg_name='gardener_cloud_storage_read')
    google_credentials_content = config.raw['service_account_key']
    with open(google_credentials_filename, "w") as google_credentials:
        json.dump(google_credentials_content, google_credentials)
    print(google_credentials_filename + " created")
    os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = google_credentials_filename


def get_provider_k8s_version_tuples():
    # determine the smallest version still submittable to the cncf repo. Check the repo's folder structure for it.
    min_version = min(map(lambda x: semver.version.Version.parse(re.sub(r'^v', '', x), optional_minor_and_patch=True), glob.glob('v1.*')))
    storage_client = google.cloud.storage.Client()
    prefix = "ci-gardener-e2e-conformance"
    bucket = storage_client.get_bucket(gs_bucket_name)
    iterator = bucket.list_blobs(prefix=prefix, delimiter='/')
    provider_version_tuples = list()
    for page in iterator.pages:
        for blob_name in page.prefixes:
            provider = re.search('conformance-(\w+?)-', blob_name).group(1)
            if provider not in provider_list:
                continue  # if the provider is not supported, continue
            k8s_version = re.search('-(v\d+\.\d+)\/', blob_name).group(1)
            if semver.version.Version.parse(re.sub(r'^v', '', k8s_version), optional_minor_and_patch=True) < min_version:
                continue  # never submit PRs for versions that have already been archived
            provider_version_tuples.append((provider, k8s_version))
    return provider_version_tuples


def process_product_provider_k8sVersion(gardener_version, product_name, provider, k8s_version):
    # check, if k8s version has already been processed
    provider_path = k8s_version + '/' + product_name + '-' + provider
    if os.path.isdir(provider_path):
        print("skipping " + product_name + '-' + provider + '-' + k8s_version + ' because directory already exists.')
        return 0  # continue if directory for provider already exists
    if provider == 'gce':
        provider_path = k8s_version + '/' + product_name + '-gcp'
        if os.path.isdir(provider_path):
            print("skipping " + product_name + '-gcp-' + k8s_version + ' because directory already exists.')
            return 0  # continue if directory for provider already exists

    # create new branch
    subprocess.run(["git", "clean", "-f", "-d"])
    branch_name = createNewBranch(product_name, provider, k8s_version)

    # modify files
    modify_files_for_product(gardener_version, product_name, provider, k8s_version, provider_path)

    # push changes
    commitAndPushChanges(gitHelper, branch_name)

    # submit PR
    createPullRequest(product, pv_tuple, branch_name)


def modify_files_for_product(gardener_version, product_name, provider, k8s_version, provider_path):
    os.makedirs(provider_path)
    os.chdir(provider_path)

    # create PRODUCT.yaml
    try:
        f = open(product_yaml_filename, 'w')
        f.write(content_product_yaml[product_name].format(provider.upper(), gardener_version, contact_email_address))
        f.close()
    except IOError:
        print("Was not able to open " + f)

    # create readme
    shutil.copyfile(temlate_dir + '/gardener_readme.txt', 'README.md')

    # download e2e.log and junit_01.xml
    downloadingE2eLogFileSuccessful = download_files_from_gcloud_storage(provider, k8s_version)
    os.chdir('../..')
    if not downloadingE2eLogFileSuccessful:
        # TODO: sysexit 1, since we need results from all providers
        try:
            subprocess.run(['rm', '-Rf', provider_path])
            print('Deleted unclean directory: ' + provider_path)
        except:
            print('Error while deleting unclean directory ' + provider_path)
            sys.exit(1)


def download_files_from_gcloud_storage(provider, k8s_version):
    e2e_log_file = 'e2e.log'
    storage_client = google.cloud.storage.Client()
    blob_prefix = 'ci-gardener-e2e-conformance-' + provider + '-' + k8s_version + '/'
    bucket = storage_client.get_bucket(gs_bucket_name)

    # find last modified directory
    prefix_blobs = bucket.list_blobs(prefix=blob_prefix, delimiter=None)
    last_blob_name = ''
    directory_names = set()
    for blob in prefix_blobs:
        last_blob_name = blob.name
        directory_name = re.search('\/(\d{10})\/', blob.name).group(1)
        directory_names.add(directory_name)
    if not directory_names:
        print("Error: Couldn't find propper bucket directory. Last blob name: " + last_blob_name)
        sys.exit(1)

    directory_names = list(directory_names)
    directory_names = sorted(directory_names, reverse=True)

    for latest_dictionary in directory_names:
        # download e2e.log
        try:
            source_blob_name = blob_prefix + latest_dictionary + '/e2e.log'
            print('Evaluating ' + source_blob_name)
            blob = bucket.blob(source_blob_name)
            blob.download_to_filename(e2e_log_file)
        except NotFound:
            print('File names e2e.log not found, trying with old logic...')
            source_blob_name = blob_prefix + latest_dictionary + '/build-log.txt'
            print('Evaluating ' + source_blob_name)
            blob = bucket.blob(source_blob_name)
            blob.download_to_filename(e2e_log_file)
        if isConformanceTestSuccessful(e2e_log_file, k8s_version):
            # download junit_01.xml
            source_blob_name = blob_prefix + latest_dictionary + '/artifacts/junit_01.xml'
            blob = bucket.blob(source_blob_name)
            blob.download_to_filename('junit_01.xml')
            return True
        else:
            os.remove(e2e_log_file)
    print(
            "Error: Couldn't find a e2e.log file with '" + conformance_tests_passed_string +
            "' string, which is required")
    return False


def isConformanceTestSuccessful(e2e_log_file, k8s_version):
    if not os.path.isfile(e2e_log_file):
        print("Error: file " + e2e_log_file + " does not exist.")
        sys.exit(1)
    isConformanceTestSuccessful = False
    pattern = "^(FAIL|SUCCESS).*Passed.*Failed.*"
    e2e_log_status_line = ""

    with open(e2e_log_file) as f:
        for line in f:
            match = re.search(pattern, line)
            if match:
                e2e_log_status_line = match[0]
                break
    if not e2e_log_status_line:
        print("Error: line of e2e log is not of format " + pattern + ". Actual line content: '" + e2e_log_status_line + "'")
        return False
    if conformance_tests_passed_string in e2e_log_status_line:
        # this is certification requirement
        isConformanceTestSuccessful = True
    else:
        print(
                "Warning: " + e2e_log_file + " does not contain '" +
                conformance_tests_passed_string + "' string. Checking next bucket.")
        isConformanceTestSuccessful = False
    return isConformanceTestSuccessful


def commitAndPushChanges(gitHelper, branch_name):
    subprocess.run(["git", "add", "*PRODUCT.yaml"])
    subprocess.run(["git", "add", "*e2e.log"])
    subprocess.run(["git", "add", "*junit_01.xml"])
    subprocess.run(["git", "add", "*README.md"])
    subprocess.run(["git", "commit", "-s",  "-m", "gardener supporting new k8s release"])
    gitHelper.push("@", f"refs/heads/{branch_name}")
    # subprocess.run(["git", "push", "origin", branch_name])
    print('Branch "' + branch_name + '" was successfully created on https://github.com/' +
          repo_path + '/branches . TODO: sanity-check the pull requests that were automatically created in "' + upstream_repo + '"')


def createPullRequest(product, provider_tuple, branch_name):
    pr_head = "gardener:" + branch_name
    title = "Conformance Results for " + provider_tuple[1] + "/" + product + "-" + provider_tuple[0]
    with open(".github/PULL_REQUEST_TEMPLATE.md") as pr_template:
        body = pr_template.read()

    body = re.sub('\[ \]', '[X]', body)
    body = body + "\n cc @hendrikKahl @dguendisch"
    pr = repo.create_pull(
        title=title,
        base="master",
        head=pr_head,
        body=body
    )
    print('Opened PR ' + title + " at " + pr.url)


try:
    subprocess.run(['rm', '-Rf', repo_name])
except subprocess.CalledProcessError as e:
    print('Directory ' + repo_name + ' does not exist.')
subprocess.run(["git", "config", "--global", "user.name", github_cfg.credentials().username()])
subprocess.run(["git", "config", "--global", "user.email",
                github_cfg.credentials().email_address()])

gitHelper = cloneForkedRepo()
syncForkAndUpstream(gitHelper)
activate_google_application_credentials(cfg_factory=cfg_factory)
provider_version_tuples = get_provider_k8s_version_tuples()
gardener_version = get_gardener_version()
upstream_prs = list(map(lambda x: str(x.head), repo.pull_requests(state='open')))

for product in content_product_yaml.keys():
    for pv_tuple in provider_version_tuples:
        subprocess.run(["git", "checkout", "master"])
        matcher = re.compile("gardener:" + product + "-" + pv_tuple[0] + "-" + pv_tuple[1])
        match = list(filter(matcher.search, upstream_prs))
        if match:
            print('Skipping ' + product + "-" + pv_tuple[0] + "-" + pv_tuple[1]
                      + " because a PR for this combination already exists.")
            continue
        process_product_provider_k8sVersion(gardener_version, product, pv_tuple[0], pv_tuple[1])
