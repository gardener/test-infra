#!/usr/bin/env python3
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

import os
import ctx
from gitutil import (
    GitHelper
)
from github.util import (
    GitHubRepositoryHelper,
)

it_label = "test/integration"
source_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "../")

repo_owner_name=os.getenv("SOURCE_GITHUB_REPO_OWNER_AND_NAME")
github_repository_owner,github_repository_name = repo_owner_name.split("/")


cfg_set = ctx.cfg_factory().cfg_set(os.getenv("CONCOURSE_CURRENT_CFG"))
github_cfg = cfg_set.github()

git_helper = GitHelper(
    repo=os.path.join(source_path, ".git"),
    github_cfg=github_cfg,
    github_repo_path=repo_owner_name
)

pull_request_number=git_helper.repo.git.config("--get", "pullrequest.id")

github_helper = GitHubRepositoryHelper(
    owner=github_repository_owner,
    name=github_repository_name,
    github_cfg=github_cfg,
)

pull_request = github_helper.repository.pull_request(pull_request_number)
labels = [str(label) for label in pull_request.issue().labels()]

print("Found labels {}".format(labels))

if it_label not in labels:
    print("{} is not set".format(it_label))
    exit(1)