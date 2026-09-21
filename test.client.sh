#!/usr/bin/env bash

set -e


run_once() {
  client_message_id="test.$(openssl rand -hex 20)"
  curl -fsSL -X POST \
    --header 'Authorization: example' \
    --data '{
    "client_message_id": "'"${client_message_id}"'",
    "title": "Hello",
    "content": "World."
  }' \
    http://127.0.0.1:39498/messages
  echo ''
}

ops="${1:-"once"}"
case ${ops} in
once)
  run_once
  ;;
loop)
  while true; do
    run_once
  done
  ;;
esac