#!/usr/bin/env bash

# make sure we are in the right folder
scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "${scriptdir}"

set -euo pipefail
trap 'exit' SIGINT SIGQUIT SIGTSTP SIGTERM SIGHUP

# Load up .env
set -o allexport
[[ -f ../.env ]] && source ../.env
[[ -f .env ]] && source .env
set +o allexport

: "${GITLAB_TOKEN:?variable not set or empty}"

bold=$(tput bold)
green=$(tput setaf 2)
sgr0=$(tput sgr0)

while IFS=' ' read -r repo file; do
	[[ $repo =~ ^#.* ]] && continue

	printf "Downloading ${bold}%s:%s${sgr0} ... " $repo $file
	repo=${repo//\//%2F}
	f=${file//\//%2F}
	url="https://gitlab.com/api/v4/projects/$repo/repository/files/$f/raw?ref=master"
	curl -fsSL --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" -o "$file" --create-dirs "$url"
	printf "${green}done${sgr0}\n"
done < <(cat sources.list | grep -v '^[[:space:]]*$')

