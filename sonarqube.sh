#!/bin/bash
# Scanner CLI 8+ docker image maps SONAR_TOKEN -> -Dsonar.token, which SonarQube 9.9
# ignores. SQ 9.9 still authenticates via sonar.login (see scanner error text).
# Do not set SONAR_TOKEN here: the image would inject sonar.token and 9.9 would
# still report "Not authorized".

set -euo pipefail

SONARQUBE_URL=10.2.150.68:19000
YOUR_PROJECT_KEY=ram-work
SONAR_LOGIN=sqp_a9c91fa76aa997c418b3ace01cc525dfd48c6b84
YOUR_REPO=/tmp/ram-work
mkdir -p "${YOUR_REPO}/logs"
LOCK_FILE="${YOUR_REPO}/logs/.sonarqube-scan.lock"
exec 9>"${LOCK_FILE}"
if ! flock -n 9; then
  echo "sonarqube scan already running; skip overlapping invocation"
  exit 0
fi
# MySQL datadir is 0750 and owned by the container user; sensors still walk
# projectBaseDir and hit AccessDeniedException. Overlay with an empty dir.
SONAR_EMPTY_OVERLAY="${YOUR_REPO}/.runall/sonar-empty-overlay"
mkdir -p "${SONAR_EMPTY_OVERLAY}"

docker run \
    --rm \
    -e SONAR_HOST_URL="http://${SONARQUBE_URL}" \
    -e SONAR_SCANNER_OPTS="-Dsonar.projectKey=${YOUR_PROJECT_KEY} -Dsonar.login=${SONAR_LOGIN}" \
    -v "${YOUR_REPO}:/usr/src" \
    -v "${SONAR_EMPTY_OVERLAY}:/usr/src/dockerInfra:ro" \
    sonarsource/sonar-scanner-cli
