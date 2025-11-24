#!/usr/bin/env sh

# Ensure nsjail chroot directory exists when sandbox is enabled
if [ "${PATHFINDER_SANDBOX_ENABLED}" = "true" ]; then
  mkdir -p /tmp/nsjail_root
  chmod 755 /tmp/nsjail_root
fi

if [ $# -eq 0 ]; then
  /usr/bin/pathfinder version
else
  /usr/bin/pathfinder "$@"
fi