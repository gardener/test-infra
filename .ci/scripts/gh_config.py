#!/usr/bin/env python3
#
# Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
