#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Check examples dir for formatting errors.
set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

#--------------
# Functions
#--------------

function formatting_check() {
  cd _examples
  for dir in $(ls); do
    cd ${dir}
    terraform fmt --check --list=false || (echo "Formatting errors found in dir: _examples/${dir}"; exit 1)
    cd -
  done
}

function formatting_diff() {
  cd _examples
  for dir in $(ls); do
    cd ${dir}
    terraform fmt --check -diff || (echo "Formatting errors found in dir: _examples/${dir}"; exit 1)
    cd -
  done
}

function formatting_fix() {
  cd _examples
  for dir in $(ls); do
    cd ${dir}
    terraform fmt
    cd -
  done
}

#--------------
# Main
#--------------

input="${1:-}"

case ${input} in
diff)
  formatting_diff
  ;;
fix)
  formatting_fix
  ;;
*)
  formatting_check
  ;;
esac
