#!/bin/bash
#
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

mkdir -p "$(dirname $0)/../.github" "$(dirname $0)/../.github/ISSUE_TEMPLATE"

for file in `find "${GARDENER_HACK_DIR}"/../.github -name '*.md'`; do
  cat "$file" |\
    sed 's/operating Gardener/working with Test Machinery/g' |\
    sed 's/to the Gardener project/for Test Machinery/g' |\
    sed 's/to Gardener/to Test Machinery/g' |\
    sed 's/- Gardener version:/- Gardener version (if relevant):\n- Test Machinery version:/g' |\
    sed 's/\/area TODO/\/area testing/g' \
  > "$(dirname $0)/../.github/${file#*.github/}"
done
