#!/bin/sh




# Create a list of build.bp files

set -e

SRCDIR="$(dirname "${0}")"

LIST_FILE=bplist
TEMP_LIST_FILE="${LIST_FILE}.tmp"

cd "${SRCDIR}"

# Locate all build.bp under the current directory. Exclusions:
# * hidden directories (starting with .)
# * Bob build directories (these contain a file .out-dir)
find . -mindepth 1 \
     -type d \( -name ".*" -o -execdir test -e {}/.out-dir \; \) -prune \
     -o -name build.bp -print > "${TEMP_LIST_FILE}"

echo ./bob/Blueprints >> "${TEMP_LIST_FILE}"
echo ./bob/blueprint/Blueprints >> "${TEMP_LIST_FILE}"

LC_ALL=C sort "${TEMP_LIST_FILE}" -o "${TEMP_LIST_FILE}"

if cmp -s "${LIST_FILE}" "${TEMP_LIST_FILE}"; then
  rm "${TEMP_LIST_FILE}"
else
  mv "${TEMP_LIST_FILE}" "${LIST_FILE}"
fi
