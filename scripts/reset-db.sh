#!/bin/sh
# (Re)creates the database from scratch on Ubuntu 24.04 (it assumes Postgres
# 16). If you are running other versions of Ubuntu, it's likely to work just
# fine with newer or older Postgres versions as well: adjust the installed
# packages below.

# run runs a command tolerating benign failures (e.g. object does not exist)
# but aborting on fatal errors that indicate the environment is in a bad state
# we cannot recover from automatically.
run() {
  printf '+ %s\n' "$*"
  output=$("$@" 2>&1)
  rc=$?
  if [ $rc -ne 0 ]; then
    case "$output" in
      *"does not exist"*|*"already exists"*|*"not found"*)
        # Benign: trying to drop something that's gone or create something
        # that's already there. Expected during idempotent resets.
        printf '%s\n' "$output"
        printf '%s exit code %d (ignored)\n' '->' "$rc"
        return 0
        ;;
      *)
        printf '%s\n' "$output" >&2
        printf '%s exit code %d\n' '->' "$rc" >&2
        exit 1
        ;;
    esac
  fi
  printf '%s\n' "$output"
  printf '%s exit code %d\n' '->' "$rc"
  return 0
}

trap 'if [ $? -eq 0 ]; then printf "\nDB reset succeeded.\n"; else printf "\nDB reset failed.\n" >&2; fi' EXIT

# Set the postgres superuser password to match the local development
# credentials in .env, and ensure the postgres database exists.
run sudo -i -u postgres psql -c "alter user postgres with encrypted password 'postgres'"
