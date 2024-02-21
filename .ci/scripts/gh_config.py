#!/usr/bin/env python3
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

import util
import yaml
import ctx

github_config_names=['tm_github_com']

def get_config(name: str):
    util.check_type(name, str)
    factory = ctx.cfg_factory()
    gh_cfg = factory.github(name)
    technicalUser = gh_cfg.credentials()

    return {
        'httpUrl': gh_cfg.http_url(),
        'apiUrl': gh_cfg.api_url(),
        'disable_tls_validation': gh_cfg.tls_validation(),
        'technicalUser': {
            'username': technicalUser.raw.get('username'),
            'emailAddress': technicalUser.email_address(),
            'password': technicalUser.raw.get('password'),
            'authToken': technicalUser.auth_token(),
        },
    }

gh_configs = {
    'secrets': [get_config(gh_config_name) for gh_config_name in github_config_names]
}

yaml_body = yaml.dump(gh_configs)

print(yaml_body)
