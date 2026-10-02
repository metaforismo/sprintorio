#!/bin/sh
set -eu

if [ ! -t 0 ]; then
  exec /bin/bash "$@"
fi

export HISTCONTROL=ignoredups
export PROMPT_COMMAND='sprintorio_status=$?; sprintorio_command=$(history 1); SPRINTORIO_EXIT_CODE=$sprintorio_status SPRINTORIO_COMMAND=$sprintorio_command node /usr/local/lib/sprintorio-command-event.js >/dev/null 2>&1 || true; history -a'
exec /bin/bash "$@"
