#!/bin/sh
set -eu

if [ "${CI:-false}" != "true" ]; then
  echo "CI environment not detected, skipping script execution:"
  echo "  --> $*"
  exit 0
fi

echo "==> Signing environment:"

env | grep --invert-match '_TOKEN'

echo "==> Executing signing process:"
echo "  --> $*"
# execute the parameters as the script
exec "$@"
