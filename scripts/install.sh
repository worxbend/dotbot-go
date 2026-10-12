#!/bin/sh
# Installs the dotbot-go CLI from GitHub Releases.
#
#   curl --proto '=https' --tlsv1.2 -sSf https://github.com/worxbend/dotbot-go/releases/latest/download/install.sh | sh
#
# Env vars:
#   DOTBOT_VERSION      release tag to install, e.g. "v0.4.1" (default: latest)
#   DOTBOT_INSTALL_DIR  install directory (default: "$HOME/.local/bin")
set -eu

REPO="worxbend/dotbot-go"
BIN_NAME="dotbot"
INSTALL_DIR="${DOTBOT_INSTALL_DIR:-$HOME/.local/bin}"

err() {
  echo "error: $1" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || err "'$1' is required but was not found"
}

os=$(uname -s)
arch=$(uname -m)

case "$os" in
  Linux) asset_os="linux" ;;
  Darwin) asset_os="darwin" ;;
  *) err "unsupported OS: $os (dotbot-go currently ships Linux and macOS binaries only)" ;;
esac

case "$arch" in
  x86_64 | amd64) asset_arch="amd64" ;;
  arm64 | aarch64) asset_arch="arm64" ;;
  *) err "unsupported architecture: $arch" ;;
esac

need_cmd curl
need_cmd tar

sha256_check() {
  file="$1"
  expected=$(awk '{print $1}' "$file.sha256")

  if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$file" | awk '{print $1}')
  elif command -v shasum >/dev/null 2>&1; then
    actual=$(shasum -a 256 "$file" | awk '{print $1}')
  else
    err "'sha256sum' or 'shasum' is required but was not found"
  fi

  [ "$expected" = "$actual" ] || err "checksum mismatch for $file"
}

need_cmd mktemp

version="${DOTBOT_VERSION:-}"
if [ -z "$version" ]; then
  latest_url=$(curl --proto '=https' --tlsv1.2 -fsSL -o /dev/null -w '%{url_effective}' \
    "https://github.com/${REPO}/releases/latest")
  version="${latest_url##*/}"
fi
[ -n "$version" ] || err "could not determine the latest release version"

asset="dotbot-${asset_os}-${asset_arch}.tar.gz"
base_url="https://github.com/${REPO}/releases/download/${version}"

tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

echo "Downloading dotbot-go ${version} (${asset_os}-${asset_arch})..."
curl --proto '=https' --tlsv1.2 -fsSL -o "$tmp_dir/$asset" "$base_url/$asset" \
  || err "failed to download $asset from $base_url"
curl --proto '=https' --tlsv1.2 -fsSL -o "$tmp_dir/$asset.sha256" "$base_url/$asset.sha256" \
  || err "failed to download $asset.sha256 from $base_url"

(cd "$tmp_dir" && sha256_check "$asset") || err "checksum verification failed"

tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"

mkdir -p "$INSTALL_DIR"
cp "$tmp_dir/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
chmod +x "$INSTALL_DIR/$BIN_NAME"

echo "Installed $BIN_NAME to $INSTALL_DIR/$BIN_NAME"

add_path_line() {
  rc_file="$1"
  marker="# added by dotbot-go installer"

  if [ -f "$rc_file" ] && grep -qF "$marker" "$rc_file" 2>/dev/null; then
    # Rewrite through the original file to preserve permissions and symlinks.
    awk -v marker="$marker" 'index($0, marker) == 0' "$rc_file" > "$tmp_dir/rc"
    cat "$tmp_dir/rc" > "$rc_file"
  else
    case ":$PATH:" in
      *":$INSTALL_DIR:"*) return 0 ;;
    esac
  fi
  escaped_install_dir=$(printf '%s' "$INSTALL_DIR" | sed 's/[\\`"$]/\\&/g')
  printf '\nexport PATH="%s:$PATH" %s\n' "$escaped_install_dir" "$marker" >> "$rc_file"
  echo "Updated $INSTALL_DIR in PATH in $rc_file"
}

add_path_line "$HOME/.bashrc"
add_path_line "$HOME/.zshrc"

if [ "$os" = Darwin ]; then
  # Bash reads only the first existing login profile in this order.
  bash_profile="$HOME/.bash_profile"
  if [ ! -f "$bash_profile" ]; then
    if [ -f "$HOME/.bash_login" ]; then
      bash_profile="$HOME/.bash_login"
    elif [ -f "$HOME/.profile" ]; then
      bash_profile="$HOME/.profile"
    fi
  fi
  add_path_line "$bash_profile"
fi

echo "Restart your shell to update PATH."

echo "Run '$BIN_NAME --help' to get started."
