#!/usr/bin/env python3
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

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
