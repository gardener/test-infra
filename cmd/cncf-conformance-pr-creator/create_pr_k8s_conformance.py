#!/usr/bin/env python3
import shutil

import util
import types
import os
import sys
import subprocess
import git
import string
import random
import gitutil
import stat
import re
from pprint import pprint
from gitutil import GitHelper
from distutils.version import StrictVersion
from shutil import copyfile
from google.cloud import storage
from model import ConfigFactory

# from gardener-cicd-libs
import ccc.cfg

repo_name = 'k8s-conformance'
product_yaml_filename = "PRODUCT.yaml"
google_credentials_filename = os.path.dirname(os.path.realpath(__file__)) + '/' + 'google_credentials.json'
repo_path = os.environ['FORK_OWNER'] + '/' + repo_name
upstream_repo = 'https://github.com/cncf/k8s-conformance'
ctx = util.ctx()
f = ctx.cfg_factory()
ce = f.cfg_set('external_active')
ci = f.cfg_set('internal_active')

script_dir = os.path.dirname(os.path.realpath(__file__))
temlate_dir = script_dir + '/'
gs_bucket_name = 'k8s-conformance-gardener'
conformance_tests_passed_string = "0 Failed | 0 Flaked | 0 Pending"

provider_list = ['gce', 'aws', 'azure', 'openstack', 'alicloud']
content_product_yaml = {}
content_product_yaml['gardener'] = """vendor: SAP
name: Gardener (https://github.com/gardener/gardener) shoot cluster deployed on {0}
version: {1}
website_url: https://gardener.cloud
repo_url: https://github.com/gardener/
documentation_url: https://github.com/gardener/documentation/wiki
product_logo_url: https://raw.githubusercontent.com/gardener/documentation/master/images/logo_w_saplogo.svg
type: installer
description: The Gardener implements automated management and operation of Kubernetes clusters as a service and aims to support that service on multiple Cloud providers."""
content_product_yaml['sap-cp'] = """vendor: SAP
name: Cloud Platform - Gardener (https://github.com/gardener/gardener) shoot cluster deployed on {0}
version: {1}
website_url: https://cloudplatform.sap.com/index.html
documentation_url: https://help.sap.com/viewer/65de2977205c403bbc107264b8eccf4b/Cloud/en-US/
product_logo_url: https://www.sap.com/dam/application/shared/logos/sap-logo-svg.svg
type: hosted
description: The Gardener implements automated management and operation of Kubernetes clusters as a service and aims to support that service on multiple Cloud providers."""


class QuitHack(object):  # shortcut-hack for quitting with a simple 'q'
    def __repr__(self):
        print('q -> quick exit')
        sys.exit(0)


q = QuitHack()


def get_gardener_version():
    gardener_version_file = os.environ['LANDSCAPE_COMPONENTS_HOME'] + '/gardener/VERSION'
    try:
        with open(gardener_version_file, 'r') as gardener_version_file_reader:
            gardener_veresion = gardener_version_file_reader.read()
    except IOError:
        print("Error: File '" + gardener_version_file + "' does not appear to exist.")
        sys.exit(1)
    return gardener_veresion


def id_generator(size=6, chars=string.ascii_uppercase + string.digits):
    return ''.join(random.choice(chars) for _ in range(size))


def cloneForkedRepo():
    github_cfg = f.github('github_com')
    subprocess.run(["git", "config", "--global", "user.name", github_cfg.credentials().username()])
    subprocess.run(["git", "config", "--global", "user.email",
                    github_cfg.credentials().email_address()])
    gitHelper = GitHelper.clone_into(repo_name, github_cfg, repo_path)
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


def createNewBranch():
    branch_name_random = id_generator()
    subprocess.run(["git", "checkout", "-b", branch_name_random])
    return branch_name_random


def getLowestVersion(versions):
    lowestVersion = versions[0]
    for i in range(len(versions) - 1):
        if (StrictVersion(versions[i]) < StrictVersion(versions[i+1])) is True:
            lowestVersion = versions[i]
        else:
            lowestVersion = versions[i+1]
    return lowestVersion


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


def modifyFiles(product_name):
    gardener_version = get_gardener_version()
    subprocess.run(["git", "clean", "-f", "-d"])

    activate_google_application_credentials()

    provider_version_tuples = get_provider_k8s_version_tuples()
    for provider_version_tuple in provider_version_tuples:
        provider = provider_version_tuple[0]
        k8s_version = provider_version_tuple[1]
        modify_files_for_product(gardener_version=gardener_version,
                                 product_name=product_name,
                                 provider=provider,
                                 k8s_version=k8s_version)

def activate_google_application_credentials():
    cfg_factory = ccc.cfg.cfg_factory()

    config = cfg_factory._cfg_element(cfg_type_name='gcloud_account',
                                      cfg_name='gardener_cloud_storage_read')
    google_credentials_content = config.raw['credentials']['storage_object_read']['auth_secret']
    google_credentials = open(google_credentials_filename, "w")
    google_credentials.write(google_credentials_content)
    google_credentials.close()
    print(google_credentials_filename + " created")
    os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = google_credentials_filename


def get_provider_k8s_version_tuples():
    storage_client = storage.Client()
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
            if k8s_version == 'v1.13': # TODO: use semver and skip all older versions, only recent 3 versions shall be considered
                continue  # k8s release 1.13 can't be added anymore, since only recent 3 release versions are considered
            provider_version_tuples.append((provider, k8s_version))
    return provider_version_tuples


def modify_files_for_product(gardener_version, product_name, provider, k8s_version):
    provider_path = k8s_version + '/' + product_name + '-' + provider
    if os.path.isdir(provider_path):
        print("skipping " + product_name + '-' + provider + '-' + k8s_version + ' because directory already exists.')
        return 0  # continue if directory for provider already exists
    if provider == 'gce':
        provider_path = k8s_version + '/' + product_name + '-gcp'
        if os.path.isdir(provider_path):
            print("skipping " + product_name + '-gcp-' + k8s_version + ' because directory already exists.')
            return 0  # continue if directory for provider already exists
    os.makedirs(provider_path)
    os.chdir(provider_path)

    # create PRODUCT.yaml
    try:
        f = open(product_yaml_filename, 'w')
        f.write(content_product_yaml[product_name].format(provider.upper(), gardener_version))
        f.close()
    except IOError:
        print("Was not able to open " + f)

    # create readme
    copyfile(temlate_dir + '/gardener_readme.txt', 'README.md')

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
    storage_client = storage.Client()
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
    log_status_line_number = -5
    if k8s_version in ['v1.10', 'v1.11', 'v1.12', 'v1.13', 'v1.14']:
        log_status_line_number = -4
    with open(e2e_log_file) as fp:
        e2e_log_status_line = (list(fp)[log_status_line_number])
    pattern = re.compile("^(FAIL|SUCCESS).*Passed.*Failed.*Flaked.*")
    if not pattern.match(e2e_log_status_line):
        print("Error: " + str(log_status_line_number) + " line of e2e.log is not of format ^(FAIL|SUCCESS).*Passed.*Failed.*Flaked.*$. Actual line content: '" + e2e_log_status_line + "'")
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
    subprocess.run(["git", "commit", "-m", "gardener supporting new k8s release"])
    gitHelper.push("@", f"refs/heads/{branch_name}")
    # subprocess.run(["git", "push", "origin", branch_name])
    print('Branch "' + branch_name + '" was successfully created on https://github.com/' +
          repo_path + '/branches . TODO: manually send pull request to "' + upstream_repo + '"')


try:
    subprocess.run(['rm', '-Rf', repo_name])
except subprocess.CalledProcessError as e:
    print('Directory ' + repo_name + ' does not exist.')
gitHelper = cloneForkedRepo()
syncForkAndUpstream(gitHelper)
branch_name = createNewBranch()
modifyFiles('sap-cp')
commitAndPushChanges(gitHelper, branch_name)

subprocess.run(["git", "checkout", "master"])
branch_name = createNewBranch()
modifyFiles('gardener')
commitAndPushChanges(gitHelper, branch_name)
