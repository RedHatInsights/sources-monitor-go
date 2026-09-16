#!/bin/bash

#
# Bash safety options:
#   - e is for exiting immediately when a command exits with a non-zero status.
#   - u is for treating unset variables as an error when substituting.
#   - o pipefail is for taking into account the exit status of the commands
#     that run on pipelines.
#
# Note: xtrace (-x) is deferred until after variable setup and suppressed
# around credential-bearing commands to prevent token exposure (CWE-532).
#
set -euo pipefail

#
# Copy the repository contents so we don't have to deal with changing
# permissions on the mounted volume.
#
source_dir=$(mktemp --directory)
readonly source_dir

cp --recursive "/repository" "${source_dir}"

# Enable xtrace for non-sensitive setup above
set -x

#
# Run the sonar scanner.
#
#
# On the main branch there is no need to give the pull request details.
#
# Suppress xtrace around sonar-scanner to avoid leaking SONARQUBE_TOKEN
# to build logs (CWE-532).
#
{ set +x; } 2>/dev/null
if [ -n "${GIT_BRANCH:-}" ] && { [ "${GIT_BRANCH}" == "main" ] || [ "${GIT_BRANCH}" == "origin/main" ]; }; then
  sonar-scanner \
    -Dsonar.exclusions="**/*.sql" \
    -Dsonar.host.url="${SONARQUBE_HOST_URL}" \
    -Dsonar.projectBaseDir="${source_dir}/repository" \
    -Dsonar.projectKey="${SONARQUBE_PROJECT_KEY}" \
    -Dsonar.projectVersion="${COMMIT_SHORT}" \
    -Dsonar.sourceEncoding="UTF-8" \
    -Dsonar.sources="${source_dir}/repository" \
    -Dsonar.token="${SONARQUBE_TOKEN}"
else
  sonar-scanner \
    -Dsonar.exclusions="**/*.sql" \
    -Dsonar.host.url="${SONARQUBE_HOST_URL}" \
    -Dsonar.projectBaseDir="${source_dir}/repository" \
    -Dsonar.projectKey="${SONARQUBE_PROJECT_KEY}" \
    -Dsonar.projectVersion="${COMMIT_SHORT}" \
    -Dsonar.pullrequest.base="main" \
    -Dsonar.pullrequest.branch="${GIT_BRANCH}" \
    -Dsonar.pullrequest.key="${GITHUB_PULL_REQUEST_ID}" \
    -Dsonar.sourceEncoding="UTF-8" \
    -Dsonar.sources="${source_dir}/repository" \
    -Dsonar.token="${SONARQUBE_TOKEN}"
fi
set -x

#
# Clean the directory of the repository.
#
rm --force --recursive "${source_dir}"
