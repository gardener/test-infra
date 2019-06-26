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
import json
import ctx
import base64

registry_secret_names=[]

def get_secret(name: str):
    util.check_type(name, str)
    factory = ctx.cfg_factory()
    registry_cfg = factory.container_registry(name).raw

    dockerjsonsecret = {
        'auths': {
            registry_cfg.get('host'): {
                'username': registry_cfg.get('username'),
                'password': registry_cfg.get('password'),
                'email': registry_cfg.get('email'),
            }
        }
    }

    encoded_secret = base64.b64encode(json.dumps(dockerjsonsecret).encode()).decode()

    return {
        'name': name,
        'dockerconfigjson': encoded_secret,
    }

registry_secrets = {
    'secrets': {
        'pullSecrets': [get_secret(gregistry_secret_name) for gregistry_secret_name in registry_secret_names]
    }
}

yaml_body = yaml.dump(registry_secrets)

print(yaml_body)
