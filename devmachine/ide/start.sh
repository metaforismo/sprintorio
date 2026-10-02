#!/bin/sh
set -eu

rm -f /workspace/.sprintorio-ready
mkdir -p "$HOME/.local/share/code-server" "$HOME/.config/code-server"
if [ -d /opt/sprintorio-home-template ] && [ ! -f "$HOME/.sprintorio-template-seeded" ]; then
  cp -a /opt/sprintorio-home-template/. "$HOME/"
  touch "$HOME/.sprintorio-template-seeded"
fi
user_settings_dir="$HOME/.local/share/code-server/User"
if [ ! -e "$user_settings_dir/settings.json" ]; then
  mkdir -p "$user_settings_dir"
  printf '%s\n' '{"workbench.colorTheme":"Default Dark Modern"}' > "$user_settings_dir/settings.json"
fi

home_dir=${HOME:-/home/sprintorio}
credential_helper=${SPRINTORIO_GIT_CREDENTIAL_HELPER:-$home_dir/.sprintorio-git-credential}

install_git_credential_helper() {
  case "$credential_helper" in /* ) ;; * ) echo "invalid credential helper path" >&2; exit 1 ;; esac
  helper_dir=${credential_helper%/*}
  mkdir -p "$helper_dir"
  old_umask=$(umask)
  umask 077
  cat > "$credential_helper" <<'SCRIPT'
#!/bin/sh
case "${1:-get}" in
  get ) ;;
  store|erase ) exit 0 ;;
  * ) exit 0 ;;
esac
if [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "missing active GitHub token" >&2
  exit 1
fi
printf '%s\n' username=x-access-token
printf '%s\n' "password=$GITHUB_TOKEN"
SCRIPT
  chmod 0700 "$credential_helper"
  umask "$old_umask"
}

configure_git_helper() {
  git -C "$1" config --unset-all core.askPass >/dev/null 2>&1 || true
  git -C "$1" config --unset-all credential.helper >/dev/null 2>&1 || true
  git -C "$1" config credential.helper "$credential_helper"
}

configure_bare_git_helper() {
  git --git-dir="$1" config --unset-all core.askPass >/dev/null 2>&1 || true
  git --git-dir="$1" config --unset-all credential.helper >/dev/null 2>&1 || true
  git --git-dir="$1" config credential.helper "$credential_helper"
}

install_git_credential_helper
git config --global --unset-all core.askPass >/dev/null 2>&1 || true
git config --global --unset-all credential.helper >/dev/null 2>&1 || true
git config --global credential.helper "$credential_helper"
export GIT_TERMINAL_PROMPT=0

if [ ! -d /workspace/.git ] && [ -n "${SPRINTORIO_REPO_URL:-}" ]; then
  git clone --branch "${SPRINTORIO_BASE_BRANCH:-main}" --single-branch "$SPRINTORIO_REPO_URL" /workspace
fi

mkdir -p /workspace/repos /workspace/tasks

if [ -d /workspace/.git ]; then
  configure_git_helper /workspace
fi
for bare_repository in /workspace/repos/*.git; do
  [ -d "$bare_repository" ] || continue
  configure_bare_git_helper "$bare_repository"
done
for checkout in /workspace/tasks/*; do
  [ -e "$checkout/.git" ] || continue
  configure_git_helper "$checkout"
done

if [ -d /workspace/.git ] && [ -n "${SPRINTORIO_WORKING_BRANCH:-}" ]; then
  git -C /workspace config --global --add safe.directory /workspace
  if git -C /workspace ls-remote --exit-code --heads origin "$SPRINTORIO_WORKING_BRANCH" >/dev/null 2>&1; then
    git -C /workspace fetch origin "$SPRINTORIO_WORKING_BRANCH"
    git -C /workspace checkout -B "$SPRINTORIO_WORKING_BRANCH" FETCH_HEAD
  else
    git -C /workspace checkout "$SPRINTORIO_WORKING_BRANCH" 2>/dev/null || git -C /workspace checkout -b "$SPRINTORIO_WORKING_BRANCH"
  fi
  hooks=/workspace/.git/hooks
  cat > "$hooks/post-commit" <<'SCRIPT'
#!/bin/sh
commit=$(git rev-parse HEAD)
curl -fsS -X POST -H 'Content-Type: application/json' --data "{\"source\":\"git\",\"event_type\":\"git.commit_created\",\"payload\":{\"commit\":\"$commit\"}}" "$SPRINTORIO_COLLECTOR_URL/event" >/dev/null 2>&1 || true
SCRIPT
  cat > "$hooks/post-checkout" <<'SCRIPT'
#!/bin/sh
branch=$(git branch --show-current)
curl -fsS -X POST -H 'Content-Type: application/json' --data "{\"source\":\"git\",\"event_type\":\"git.branch_changed\",\"payload\":{\"branch\":\"$branch\"}}" "$SPRINTORIO_COLLECTOR_URL/event" >/dev/null 2>&1 || true
SCRIPT
  cat > "$hooks/pre-push" <<'SCRIPT'
#!/bin/sh
curl -fsS -X POST -H 'Content-Type: application/json' --data '{"source":"git","event_type":"git.push_started","payload":{}}' "$SPRINTORIO_COLLECTOR_URL/event" >/dev/null 2>&1 || true
SCRIPT
  chmod 0755 "$hooks/post-commit" "$hooks/post-checkout" "$hooks/pre-push"
fi

default_workspace=/workspace/tasks
[ ! -d /workspace/.git ] || default_workspace=/workspace
code-server --bind-addr 0.0.0.0:8080 --auth none --disable-telemetry "$default_workspace" &
code_server_pid=$!
ttyd --port 7681 --writable --url-arg --terminal-type xterm-256color /usr/local/bin/sprintorio-terminal-session &
ttyd_pid=$!

cleanup() {
  kill "$code_server_pid" "$ttyd_pid" 2>/dev/null || true
  wait "$code_server_pid" "$ttyd_pid" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

attempt=0
while ! curl -fsS http://127.0.0.1:8080/healthz >/dev/null 2>&1 || ! kill -0 "$ttyd_pid" 2>/dev/null; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 300 ]; then
    echo "developer services did not become ready" >&2
    exit 1
  fi
  sleep 0.1
done
touch /workspace/.sprintorio-ready

while kill -0 "$code_server_pid" 2>/dev/null && kill -0 "$ttyd_pid" 2>/dev/null; do
  sleep 1
done
exit 1
