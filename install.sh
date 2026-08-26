#!/bin/sh
# Downloads the latest webadelphos release for this machine and runs it in the background.
#   curl -sL https://raw.githubusercontent.com/json-bateman/webadelphos/main/install.sh | sh
set -e

REPO="json-bateman/webadelphos"

os=$(uname -s)
if [ "$os" != "Linux" ]; then
  echo "webadelphos only ships Linux binaries (detected: $os)" >&2
  exit 1
fi

arch=$(uname -m)
case "$arch" in
  x86_64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="webadelphos-linux-${arch}"
url=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep browser_download_url \
  | grep "$asset" \
  | cut -d '"' -f4)

if [ -z "$url" ]; then
  echo "could not find a release asset matching ${asset}" >&2
  exit 1
fi

echo "Downloading ${asset}..."
curl -sL -o webadelphos "$url"
chmod +x webadelphos

echo "Starting webadelphos in the background (logs: webadelphos.log)..."
nohup ./webadelphos > webadelphos.log 2>&1 &

echo "webadelphos is running (pid $!). Open http://localhost:44223"
echo "Note: running as $(id -un). For full visibility into root-owned processes/units, run instead: sudo ./webadelphos"
