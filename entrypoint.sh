#!/usr/bin/env sh

if [ $# -eq 0 ]; then
  /usr/bin/pathfinder version
else
  /usr/bin/pathfinder "$@"
fi
