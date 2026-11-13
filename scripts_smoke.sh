#!/usr/bin/env sh
set -eu
BASE="${API_URL:-http://localhost:8080/api/v1}"
if ! curl --fail --silent "${BASE%/api/v1}/health"; then
 echo "API health check failed" >&2
 exit 1
fi
printf '\nAPI отвечает. Проверьте регистрацию и ежедневное задание через сайт.\n'
