#!/bin/sh
set -eu

if [ "${CI:-false}" != "true" ]; then
  echo "CI environment not detected, skipping script execution:"
  echo "  --> $*"
  exit 0
fi

# execute the parameters as the script
exec "$@"
