#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Local script runner for recursive markdown-link-check.
# Runs a dockerized version of this program: https://github.com/tcort/markdown-link-check
# Based on: https://github.com/gaurav-nelson/github-action-markdown-link-check/blob/master/entrypoint.sh
set -o errexit
set -o errtrace
set -o nounset
set -o pipefail
trap 'echo "ERROR at line ${LINENO}"' ERR

# Allow users to optionally specify additional docker options, path,
# or docker alternative (such as podman).
DOCKER=${1:-`command -v docker`}
DOCKER_RUN_OPTS=${2:-}
DOCKER_VOLUME_OPTS=${3:-}
PROVIDER_DIR=${4:-}

# In CI, we use a Github Action instead of this script.
if [ ${CI:-} ]; then
  echo "Running inside of Github Actions. Exiting"
  exit 0
fi

if [ -z "${PROVIDER_DIR}" ]; then
  echo "Please specify the directory containing the kubernetes provider"
  exit  1
fi

echo "==> Checking Markdown links..."

error_file="markdown-link-check-errors.txt"
output_file="markdown-link-check-output.txt"

rm -f "./${error_file}" "./${output_file}"

${DOCKER} run ${DOCKER_RUN_OPTS} --rm -i -t \
  -v ${PROVIDER_DIR}:/github/workspace:${DOCKER_VOLUME_OPTS} \
  -w /github/workspace \
  --entrypoint /usr/bin/find \
  docker.io/robertbeal/markdown-link-checker \
  website \( -type f -name "*.md" -or -name "*.markdown" \) -exec markdown-link-check --config .markdownlinkcheck.json --quiet --verbose {} \; \
  | tee -a "${output_file}"

touch "${error_file}"
PREVIOUS_LINE=""
while IFS= read -r LINE; do
  if [[ $LINE = *"FILE"* ]]; then
    PREVIOUS_LINE=$LINE
    if [[ $(tail -1 "${error_file}") != *FILE* ]]; then
        echo -e "\n" >> "${error_file}"
        echo "$LINE" >> "${error_file}"
    fi
  elif [[ $LINE = *"✖"* ]] && [[ $PREVIOUS_LINE = *"FILE"* ]]; then
    echo "$LINE" >> "${error_file}"
  else
    PREVIOUS_LINE=""
  fi
done < "${output_file}"

if grep -q "ERROR:" "${output_file}"; then
  echo -e "==================> MARKDOWN LINK CHECK FAILED <=================="
  if [[ $(tail -1 "${error_file}") = *FILE* ]]; then
    sed '$d' "${error_file}"
  else
    cat "${error_file}"
  fi
  printf "\n"
  echo -e "=================================================================="
  exit 1
else
  echo -e "==================> MARKDOWN LINK CHECK SUCCESS <=================="
  printf "\n"
  echo -e "[✔] All links are good!"
  printf "\n"
  echo -e "==================================================================="
fi

rm -f "./${error_file}" "./${output_file}"

exit 0
