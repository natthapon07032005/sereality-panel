#!/usr/bin/env bash
set -euo pipefail

SEREALITY_INSTALLER_TEST=1
source "$(dirname "${BASH_SOURCE[0]}")/../install.sh"

test_release_asset_url_uses_sereality_repository() {
    local got
    got="$(sereality_release_asset_url "v1.0.0" "amd64")"
    [[ "$got" == "https://github.com/natthapon07032005/sereality-panel/releases/download/v1.0.0/x-ui-linux-amd64.tar.gz" ]]
}

test_control_script_url_uses_sereality_repository() {
    local got
    got="$(sereality_raw_asset_url "x-ui.sh")"
    [[ "$got" == "https://raw.githubusercontent.com/natthapon07032005/sereality-panel/main/x-ui.sh" ]]
}

test_checksum_url_matches_release_asset() {
    local got
    got="$(sereality_release_checksum_url "v1.0.0" "arm64")"
    [[ "$got" == "https://github.com/natthapon07032005/sereality-panel/releases/download/v1.0.0/x-ui-linux-arm64.tar.gz.sha256" ]]
}

test_first_sereality_release_version_is_accepted() {
    local got
    got="$(sereality_normalize_version "1.0.0")"
    [[ "$got" == "v1.0.0" ]]
}

test_sereality_cli_uses_short_command_name() {
    [[ "${SEREALITY_CLI_NAME}" == "sp" ]]
}

test_management_script_points_to_sereality_repository() {
    local got
    got="$(bash "$(dirname "${BASH_SOURCE[0]}")/../x-ui.sh" --print-sereality-config)"
    [[ "$got" == *"https://raw.githubusercontent.com/natthapon07032005/sereality-panel/main/x-ui.sh"* ]]
    [[ "$got" == *"https://raw.githubusercontent.com/natthapon07032005/sereality-panel/v1.0.1/install.sh"* ]]
}

test_release_asset_url_uses_sereality_repository
test_control_script_url_uses_sereality_repository
test_checksum_url_matches_release_asset
test_first_sereality_release_version_is_accepted
test_sereality_cli_uses_short_command_name
test_management_script_points_to_sereality_repository
printf 'installer repository tests: ok\n'
