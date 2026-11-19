#!/usr/bin/env bash
# Records the README screenshots and demo GIFs, in dark and light, with VHS
# (https://github.com/charmbracelet/vhs) against the fake limactl in this
# folder, so no real VM is touched. Run from anywhere: make demo
set -euo pipefail

demo=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$demo/../.." && pwd)
assets="$repo/assets"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

command -v vhs >/dev/null || { echo "record.sh: needs vhs (brew install vhs)" >&2; exit 1; }

version=$(git -C "$repo" describe --tags --abbrev=0 2>/dev/null || echo dev)
go build -C "$repo" -ldflags "-X main.version=$version" -o "$work/lima-tui" .

# The VMs the demo starts with: status sshPort cpus memoryGiB diskGiB
reset_vms() {
	rm -rf "$work/state" && mkdir -p "$work/state"
	# Listed in this order; demo.tape.in moves by row.
	echo "Running 60301 8 16 200" >"$work/state/builder"
	echo "Running 60022 4 4 100" >"$work/state/default"
	echo "Stopped 0 2 4 50" >"$work/state/k3s"
	echo "Running 60118 4 8 100" >"$work/state/postgres"
	echo "Stopped 0 2 2 20" >"$work/state/test-vm"
}

# An empty config folder, so your own config.yaml and skins stay out of it.
mkdir -p "$work/config"

for scheme in dark light; do
	theme="Builtin Dark"
	[[ $scheme == light ]] && theme="Builtin Light"
	reset_vms
	sed -e "s|@OUT@|$work/$scheme|g" \
		-e "s|@VHS_THEME@|$theme|g" \
		-e "s|@SKIN@|$scheme|g" \
		-e "s|@DEMO_DIR@|$demo|g" \
		-e "s|@STATE@|$work/state|g" \
		-e "s|@CONFIG@|$work/config|g" \
		-e "s|@BIN@|$work/lima-tui|g" \
		"$demo/demo.tape.in" >"$work/$scheme.tape"
	vhs "$work/$scheme.tape"
	mv "$work/$scheme.png" "$assets/interface-$scheme.png"
	mv "$work/$scheme.gif" "$assets/demo-$scheme.gif"
done

ls -lh "$assets"/interface-*.png "$assets"/demo-*.gif
