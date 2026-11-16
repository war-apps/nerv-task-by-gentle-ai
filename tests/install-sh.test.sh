#!/usr/bin/env bash
# Tests scripts/install.sh against a stub curl (no real network) and a
# fixture release archive, on bash (the test harness itself needs
# bash's arrays/functions; install.sh itself stays POSIX sh).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_SH="$REPO_ROOT/scripts/install.sh"

pass=0
fail=0

note() { printf '%s\n' "$*" >&2; }
ok() { pass=$((pass + 1)); note "ok - $*"; }
bad() {
	fail=$((fail + 1))
	note "FAIL - $*"
}

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

# build_fixture creates, under $1 (a directory), a release archive and a
# matching checksums.txt for tag v9.9.9, linux/amd64, whose "nerv-task"
# binary is a tiny shell script that prints a recognizable marker.
build_fixture() {
	fixture_dir="$1"
	mkdir -p "$fixture_dir/payload"
	cat >"$fixture_dir/payload/nerv-task" <<'EOF'
#!/usr/bin/env sh
echo "fixture nerv-task $*"
EOF
	chmod +x "$fixture_dir/payload/nerv-task"

	archive="$fixture_dir/nerv-task_9.9.9_linux_amd64.tar.gz"
	(cd "$fixture_dir/payload" && tar -czf "$archive" nerv-task)

	sum=$(sha256_of "$archive")
	printf '%s  nerv-task_9.9.9_linux_amd64.tar.gz\n' "$sum" >"$fixture_dir/checksums.txt"
}

# install_stub_curl writes a curl stub at $1/curl serving the fixture
# under $2 for any URL ending in the archive or checksums.txt name, and
# a canned "latest release" JSON for the GitHub API URL. Any other URL
# makes it fail loudly, so a wrong URL never silently "succeeds".
install_stub_curl() {
	stub_dir="$1"
	fixture_dir="$2"
	mkdir -p "$stub_dir"
	cat >"$stub_dir/curl" <<EOF
#!/usr/bin/env bash
set -euo pipefail
out=""
url=""
args=("\$@")
for ((i = 0; i < \${#args[@]}; i++)); do
	case "\${args[i]}" in
	-o) out="\${args[i+1]}"; i=\$((i + 1)) ;;
	http*) url="\${args[i]}" ;;
	esac
done

case "\$url" in
*/releases/latest)
	echo '{"tag_name":"v9.9.9"}'
	;;
*nerv-task_9.9.9_linux_amd64.tar.gz)
	cp "$fixture_dir/nerv-task_9.9.9_linux_amd64.tar.gz" "\$out"
	;;
*/checksums.txt)
	cp "$fixture_dir/checksums.txt" "\$out"
	;;
*)
	echo "stub curl: unexpected URL: \$url" >&2
	exit 1
	;;
esac
EOF
	chmod +x "$stub_dir/curl"
}

# install_stub_uname writes a uname stub at $1/uname reporting a fixed
# Linux/x86_64 platform, so this harness exercises install.sh's Linux
# code path deterministically, whatever host (including Windows Git
# Bash, whose uname reports MINGW64_NT-...) actually runs it.
install_stub_uname() {
	stub_dir="$1"
	mkdir -p "$stub_dir"
	cat >"$stub_dir/uname" <<'EOF'
#!/usr/bin/env bash
case "$1" in
-s) echo "Linux" ;;
-m) echo "x86_64" ;;
*) echo "Linux" ;;
esac
EOF
	chmod +x "$stub_dir/uname"
}

test_successful_install() {
	work=$(mktemp -d)
	trap 'rm -rf "$work"' RETURN

	fixture_dir="$work/fixture"
	stub_dir="$work/stub"
	install_dir="$work/install"
	mkdir -p "$fixture_dir" "$install_dir"
	build_fixture "$fixture_dir"
	install_stub_curl "$stub_dir" "$fixture_dir"
	install_stub_uname "$stub_dir"

	if PATH="$stub_dir:$PATH" \
		NERV_TASK_INSTALL_DIR="$install_dir" \
		NERV_TASK_NO_INSTALL=1 \
		sh "$INSTALL_SH" >"$work/stdout.log" 2>"$work/stderr.log"; then
		if [ -x "$install_dir/nerv-task" ] && grep -q "fixture nerv-task" "$install_dir/nerv-task"; then
			ok "successful install places the fixture binary, executable"
		else
			bad "successful install did not place the expected binary (see $work)"
		fi
	else
		bad "install.sh exited non-zero on a valid fixture: $(cat "$work/stderr.log")"
	fi
}

test_checksum_mismatch_is_refused() {
	work=$(mktemp -d)
	trap 'rm -rf "$work"' RETURN

	fixture_dir="$work/fixture"
	stub_dir="$work/stub"
	install_dir="$work/install"
	mkdir -p "$fixture_dir" "$install_dir"
	build_fixture "$fixture_dir"
	# Corrupt the checksum so it no longer matches the archive.
	printf 'deadbeef  nerv-task_9.9.9_linux_amd64.tar.gz\n' >"$fixture_dir/checksums.txt"
	install_stub_curl "$stub_dir" "$fixture_dir"
	install_stub_uname "$stub_dir"

	if PATH="$stub_dir:$PATH" \
		NERV_TASK_INSTALL_DIR="$install_dir" \
		NERV_TASK_NO_INSTALL=1 \
		sh "$INSTALL_SH" >"$work/stdout.log" 2>"$work/stderr.log"; then
		bad "install.sh accepted a corrupted archive (checksum mismatch not caught)"
	else
		if grep -qi "checksum mismatch" "$work/stderr.log"; then
			ok "a checksum mismatch is refused with a clear message"
		else
			bad "install.sh refused for the wrong reason: $(cat "$work/stderr.log")"
		fi
	fi
	if [ -e "$install_dir/nerv-task" ]; then
		bad "a corrupted archive must never be installed"
	fi
}

test_explicit_version_skips_latest_lookup() {
	work=$(mktemp -d)
	trap 'rm -rf "$work"' RETURN

	fixture_dir="$work/fixture"
	stub_dir="$work/stub"
	install_dir="$work/install"
	mkdir -p "$fixture_dir" "$install_dir"
	build_fixture "$fixture_dir"
	# A stub curl with NO "releases/latest" case: if install.sh still
	# tries to resolve "latest" despite NERV_TASK_VERSION being set,
	# this stub makes that failure visible instead of silently passing.
	mkdir -p "$stub_dir"
	cat >"$stub_dir/curl" <<EOF
#!/usr/bin/env bash
set -euo pipefail
out=""
url=""
args=("\$@")
for ((i = 0; i < \${#args[@]}; i++)); do
	case "\${args[i]}" in
	-o) out="\${args[i+1]}"; i=\$((i + 1)) ;;
	http*) url="\${args[i]}" ;;
	esac
done
case "\$url" in
*nerv-task_9.9.9_linux_amd64.tar.gz) cp "$fixture_dir/nerv-task_9.9.9_linux_amd64.tar.gz" "\$out" ;;
*/checksums.txt) cp "$fixture_dir/checksums.txt" "\$out" ;;
*) echo "stub curl: unexpected URL (latest lookup should have been skipped): \$url" >&2; exit 1 ;;
esac
EOF
	chmod +x "$stub_dir/curl"
	install_stub_uname "$stub_dir"

	if PATH="$stub_dir:$PATH" \
		NERV_TASK_VERSION="v9.9.9" \
		NERV_TASK_INSTALL_DIR="$install_dir" \
		NERV_TASK_NO_INSTALL=1 \
		sh "$INSTALL_SH" >"$work/stdout.log" 2>"$work/stderr.log"; then
		ok "NERV_TASK_VERSION skips the latest-release lookup"
	else
		bad "install.sh failed with NERV_TASK_VERSION set: $(cat "$work/stderr.log")"
	fi
}

test_successful_install
test_checksum_mismatch_is_refused
test_explicit_version_skips_latest_lookup

note ""
note "install-sh.test.sh: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
