#!/usr/bin/env sh
# nerv-task installer: downloads the right release for this platform,
# installs the nerv-task binary, and runs `nerv-task install`.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/war-apps/nerv-task-by-gentle-ai/main/scripts/install.sh | sh
#
# Every option is an environment variable (this script is piped into
# sh, so it takes no arguments of its own):
#
#   NERV_TASK_VERSION       Exact tag to install (e.g. "v1.2.0"). Default:
#                            the latest stable release.
#   NERV_TASK_INSTALL_DIR    Where to put the binary. Default: $HOME/.local/bin.
#   NERV_TASK_NO_INSTALL     When set to any non-empty value, skip running
#                            `nerv-task install` after placing the binary.
#   NERV_TASK_TTY_DEVICE     A tty device (e.g. /dev/tty) to reattach to
#                            stdin before running `nerv-task install`, so
#                            its interactive prompts work even when this
#                            script itself was piped from curl. When unset
#                            or when stdin is already a terminal, this is
#                            skipped.

set -eu

REPO="war-apps/nerv-task-by-gentle-ai"
BINARY="nerv-task"
API_BASE="https://api.github.com/repos/${REPO}"
RELEASE_BASE="https://github.com/${REPO}/releases/download"

log() { printf '%s\n' "$*" >&2; }
die() { log "nerv-task install.sh: $*"; exit 1; }

need() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required but not found on PATH"
}

detect_platform() {
	os=$(uname -s)
	arch=$(uname -m)

	case "$os" in
	Linux) goos=linux ;;
	Darwin) goos=darwin ;;
	*) die "unsupported OS: $os (see https://github.com/${REPO}#install)" ;;
	esac

	case "$arch" in
	x86_64 | amd64) goarch=amd64 ;;
	arm64 | aarch64) goarch=arm64 ;;
	*) die "unsupported architecture: $arch (see https://github.com/${REPO}#install)" ;;
	esac
}

resolve_version() {
	if [ -n "${NERV_TASK_VERSION:-}" ]; then
		tag="$NERV_TASK_VERSION"
		return
	fi
	tag=$(curl -fsSL "${API_BASE}/releases/latest" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
	[ -n "$tag" ] || die "could not resolve the latest release from ${API_BASE}/releases/latest"
}

download() {
	version=${tag#v}
	ext=tar.gz
	archive_name="${BINARY}_${version}_${goos}_${goarch}.${ext}"
	base_url="${RELEASE_BASE}/${tag}"

	workdir=$(mktemp -d)
	trap 'rm -rf "$workdir"' EXIT

	log "Downloading ${archive_name} (${tag})..."
	curl -fsSL -o "${workdir}/${archive_name}" "${base_url}/${archive_name}" || die "download failed: ${base_url}/${archive_name}"
	curl -fsSL -o "${workdir}/checksums.txt" "${base_url}/checksums.txt" || die "download failed: ${base_url}/checksums.txt"

	verify_checksum "${workdir}/${archive_name}" "${workdir}/checksums.txt"

	tar -xzf "${workdir}/${archive_name}" -C "$workdir" "$BINARY"
	extracted="${workdir}/${BINARY}"
}

verify_checksum() {
	archive="$1"
	checksums="$2"
	name=$(basename "$archive")
	expected=$(grep " ${name}\$" "$checksums" | awk '{print $1}')
	[ -n "$expected" ] || die "no checksum entry for ${name} in checksums.txt"

	if command -v sha256sum >/dev/null 2>&1; then
		actual=$(sha256sum "$archive" | awk '{print $1}')
	elif command -v shasum >/dev/null 2>&1; then
		actual=$(shasum -a 256 "$archive" | awk '{print $1}')
	else
		die "neither sha256sum nor shasum is available to verify the download"
	fi
	[ "$expected" = "$actual" ] || die "checksum mismatch for ${name}: expected ${expected}, got ${actual}"
}

install_binary() {
	install_dir="${NERV_TASK_INSTALL_DIR:-$HOME/.local/bin}"
	mkdir -p "$install_dir"
	install -m 0755 "$extracted" "${install_dir}/${BINARY}"
	log "Installed ${BINARY} to ${install_dir}/${BINARY}"

	case ":$PATH:" in
	*":${install_dir}:"*) ;;
	*) log "Warning: ${install_dir} is not on your PATH. Add it, e.g.: export PATH=\"${install_dir}:\$PATH\"" ;;
	esac
}

run_nerv_task_install() {
	if [ -n "${NERV_TASK_NO_INSTALL:-}" ]; then
		log "NERV_TASK_NO_INSTALL is set: skipping nerv-task install."
		return
	fi

	bin="${NERV_TASK_INSTALL_DIR:-$HOME/.local/bin}/${BINARY}"
	if [ -t 0 ]; then
		"$bin" install
	elif [ -n "${NERV_TASK_TTY_DEVICE:-}" ] && [ -r "${NERV_TASK_TTY_DEVICE}" ]; then
		"$bin" install <"${NERV_TASK_TTY_DEVICE}"
	else
		"$bin" install --no-configure
	fi
}

main() {
	need curl
	need tar
	detect_platform
	resolve_version
	download
	install_binary
	run_nerv_task_install
}

main "$@"
