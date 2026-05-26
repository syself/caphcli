#!/bin/bash

# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -Eeuo pipefail

# In the rescue system smartctl is always available. This is just needed if the
# script gets executed by hand (outside caph).
if ! type smartctl >/dev/null 2>&1; then
    apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq smartmontools
fi

mapfile -t devices < <(lsblk --nodeps --noheadings -o NAME,TYPE | awk '$2=="disk"{print $1}')

if [ ${#devices[@]} -eq 0 ]; then
    echo "ERROR: no disk devices found by lsblk"
    exit 1
fi

result=$(mktemp)
trap 'rm -f "$result"' EXIT

for dev in "${devices[@]}"; do
    echo "Checking /dev/$dev"
    { smartctl -H "/dev/$dev" || true; } \
        | { grep -vP '^(smartctl \d+\.\d+.*|Copyright|=+ START OF)' || true; } \
        | { grep -v '^$' || true; } \
        | { sed "s#^#/dev/$dev: #" || true; } \
        >> "$result"
done

errors=$(grep -v PASSED "$result" || true)
if [ -n "$errors" ]; then
    echo "check-all-disks FAILED"
    echo "$errors"
    exit 1
fi

echo "check-all-disks PASSED. All disks look healthy."
echo
cat "$result"
exit 0
