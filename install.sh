#!/bin/sh
# Downloads the latest weblet release for this machine and runs it in the background.
#   curl -sL https://raw.githubusercontent.com/json-bateman/weblet/main/install.sh | sh
set -e

REPO="json-bateman/weblet"

os=$(uname -s)
if [ "$os" != "Linux" ]; then
  echo "weblet only ships Linux binaries (detected: $os)" >&2
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

asset="weblet-linux-${arch}"
url=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep browser_download_url \
  | grep "$asset" \
  | cut -d '"' -f4)

if [ -z "$url" ]; then
  echo "could not find a release asset matching ${asset}" >&2
  exit 1
fi

echo "Downloading ${asset}..."
curl -sL -o weblet "$url"
chmod +x weblet

echo "Starting weblet in the background (logs: weblet.log)..."
nohup ./weblet > weblet.log 2>&1 &

echo "weblet is running (pid $!). Open http://localhost:44223"
echo "Note: running as $(id -un). For full visibility into root-owned processes/units, run instead: sudo ./weblet"
