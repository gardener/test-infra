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

import xmltodict
import json
import sys
import os
import getopt
import uuid
import re
import glob


def convert_junit_01_xml_to_json(
        inputdir, index, targetdir, descfile):

    try:
        junit_files = glob.glob(inputdir + "/junit*.xml")
        for junit_file in junit_files:
            if "junit_runner.xml" in junit_file:
                continue  # special case
            with open(junit_file) as junit_file_reader:
                # force_list is required, otherwise xmltodict would return a single element instead of a list in case there is only one testcase in xml
                doc = xmltodict.parse(junit_file_reader.read(), attr_prefix='', cdata_key='text', dict_constructor=dict, force_list={'testcase': True})
            for testcase in doc['testsuite']['testcase']:
                if 'skipped' not in testcase:

                    # Generate unique filename
                    filename = os.path.join(targetdir, 'test-{}.json'.format(str(uuid.uuid4())[0:6]))
                    c = 0
                    while os.path.isfile(filename):
                        filename = os.path.join(targetdir, 'test-{}.json'.format(str(uuid.uuid4())[0:6]))
                        if c > 50:
                            help('ERROR: random filename for test result could not be generated')
                            sys.exit(1)
                        c += 1

                    # Create target json structure
                    res = {}
                    if 'failure' in testcase:
                        res['status'] = 'failure'
                    else:
                        res['status'] = 'success'

                    for key, value in testcase.items():
                        res[key] = value
                    res['name'] = res['name'].strip()
                    matchObj = re.match(r'\[(.*?)\].*', res['name'], re.M | re.I)
                    res['sig'] = matchObj.group(1).strip()
                    res['duration'] = int(float(res['time']))
                    res['test_desc_file'] = descfile
                    del res['time']

                    # Write json file
                    with open(filename, 'w') as f:
                        json_index = {"index": {"_index": index, "_type": "_doc"}}
                        f.write(json.dumps(json_index)+"\n")
                        f.write(json.dumps(res))

    except Exception as e:
        help('ERROR (convtojson.py): something went wrong during XML parsing: {}'.format(e))
        return None

    return 0


HELPMESSAGE = 'convtojson.py -f|--inputdir <xml inputdir> -i|--index <elastic search index> -t|--targetdir <directory for json files> -d|--descfile <test description file>'


def help(msg=HELPMESSAGE):
    print(msg, file=sys.stderr)


if __name__ == "__main__":
    filename = ''
    index = ''
    targetdir = ''
    descfile = ''

    try:
        opts, args = getopt.getopt(sys.argv[1:], "hf:i:t:d:",
                                   ["inputdir=", "index=", "targetdir=", "descfile="])
    except getopt.GetoptError:
        help()
        sys.exit(2)
    for opt, arg in opts:
        if opt == '-h':
            help()
            sys.exit()
        elif opt in ("-f", "--inputdir"):
            inputdir = arg
        elif opt in ("-i", "--index"):
            index = arg
        elif opt in ("-t", "--targetdir"):
            targetdir = arg
        elif opt in ("-d", "--descfile"):
            descfile = arg

    if inputdir == '' or targetdir == '':
        help()
        sys.exit(1)

    if not os.path.isdir(inputdir):
        help('ERROR (convtojson.py): directory "{}" does not exist'.format(inputdir))
        sys.exit(1)

    if not os.path.isdir(targetdir):
        help('ERROR (convtojson.py): directory "{}" does not exist'.format(targetdir))
        sys.exit(1)

    if convert_junit_01_xml_to_json(inputdir, index, targetdir, descfile) is None:
        sys.exit(1)
    else:
        sys.exit(0)
