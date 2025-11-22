#!/bin/sh
set -e

# harific installer script
# https://pb33f.io/harific
#
# Designed for quick installs over the network and CI/CD
#   sh -c "$(curl -sSL https://pb33f.io/scripts/install_harific.sh)"

INSTALL_DIR=${INSTALL_DIR:-"/usr/local/bin"}
BINARY_NAME=${BINARY_NAME:-"harific"}

REPO_NAME="pb33f/harific"
ISSUE_URL="https://github.com/pb33f/harific/issues/new"

# get_latest_release "pb33f/harific"
get_latest_release() {
  local response
  local retries=3
  local delay=2

  for i in $(seq 1 $retries); do
    if [ -n "$GITHUB_TOKEN" ]; then
      response=$(curl --retry 5 --silent -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$1/releases/latest" 2>/dev/null)
    else
      response=$(curl --retry 5 --silent "https://api.github.com/repos/$1/releases/latest" 2>/dev/null)
    fi

    if echo "$response" | grep -q '"tag_name"'; then
      echo "$response" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
      return 0
    fi

    echo "API request failed (attempt $i/$retries), retrying in ${delay}s..." >&2
    sleep $delay
    delay=$((delay * 2))
  done

  echo "Failed to get latest release after $retries attempts" >&2
  return 1
}

# Fallback version detection
get_version_with_fallback() {
  # Try API first
  local api_version
  api_version=$(get_latest_release $REPO_NAME 2>/dev/null)
  if [ $? -eq 0 ] && [ -n "$api_version" ]; then
    echo "$api_version"
    return 0
  fi

  # Fallback to parsing GitHub releases page
  local web_version
  web_version=$(curl -sL https://github.com/pb33f/harific/releases/latest 2>/dev/null | \
    grep -o 'releases/tag/v[0-9][^"]*' | head -1 | sed 's/releases\/tag\///')

  if [ -n "$web_version" ]; then
    echo "$web_version"
    return 0
  fi

  # Ultimate fallback - use a known stable version
  echo "v0.1.0"  # Update this with actual first release
}

get_asset_name() {
  echo "harific_$1_$2_$3.tar.gz"
}

get_download_url() {
  local asset_name=$(get_asset_name $1 $2 $3)
  echo "https://github.com/pb33f/harific/releases/download/v$1/${asset_name}"
}

get_checksum_url() {
  echo "https://github.com/pb33f/harific/releases/download/v$1/checksums.txt"
}

command_exists() {
  command -v "$@" >/dev/null 2>&1
}

fmt_error() {
  echo ${RED}"Error: $@"${RESET} >&2
}

fmt_warning() {
  echo ${YELLOW}"Warning: $@"${RESET} >&2
}

fmt_underline() {
  echo "$(printf '\033[4m')$@$(printf '\033[24m')"
}

fmt_code() {
  echo "\`$(printf '\033[38;5;247m')$@${RESET}\`"
}

setup_color() {
  # Only use colors if connected to a terminal
  if [ -t 1 ]; then
    RED=$(printf '\033[31m')
    GREEN=$(printf '\033[32m')
    YELLOW=$(printf '\033[33m')
    BLUE=$(printf '\033[34m')
    MAGENTA=$(printf '\033[35m')
    BOLD=$(printf '\033[1m')
    RESET=$(printf '\033[m')
  else
    RED=""
    GREEN=""
    YELLOW=""
    BLUE=""
    MAGENTA=""
    BOLD=""
    RESET=""
  fi
}

get_os() {
  case "$(uname -s)" in
    *linux* ) echo "linux" ;;
    *Linux* ) echo "linux" ;;
    *darwin* ) echo "darwin" ;;
    *Darwin* ) echo "darwin" ;;
  esac
}

get_machine() {
  case "$(uname -m)" in
    "x86_64"|"amd64"|"x64")
      echo "x86_64" ;;
    "i386"|"i86pc"|"x86"|"i686")
      echo "i386" ;;
    "arm64"|"armv6l"|"aarch64")
      echo "arm64"
  esac
}

get_tmp_dir() {
  echo $(mktemp -d)
}

do_checksum() {
  checksum_url=$(get_checksum_url $version)
  expected_checksum=$(curl -sL $checksum_url | grep $asset_name | awk '{print $1}')

  if [ -z "$expected_checksum" ]; then
    fmt_error "Failed to retrieve checksum for $asset_name"
    exit 1
  fi

  if command_exists sha256sum; then
    checksum=$(sha256sum $asset_name | awk '{print $1}')
  elif command_exists shasum; then
    checksum=$(shasum -a 256 $asset_name | awk '{print $1}')
  else
    fmt_warning "Could not find a checksum program. Install shasum or sha256sum to validate checksum."
    return 0
  fi

  if [ "$checksum" != "$expected_checksum" ]; then
    fmt_error "Checksums do not match"
    fmt_error "Expected: $expected_checksum"
    fmt_error "Got:      $checksum"
    exit 1
  fi

  echo "Checksum verified successfully"
}

do_install_binary() {
  asset_name=$(get_asset_name $version $os $machine)
  download_url=$(get_download_url $version $os $machine)

  command_exists curl || {
    fmt_error "curl is not installed"
    exit 1
  }

  command_exists tar || {
    fmt_error "tar is not installed"
    exit 1
  }

  local tmp_dir=$(get_tmp_dir)

  # Download tar.gz to tmp directory
  echo "Downloading $download_url"
  (cd $tmp_dir && curl -sL -O "$download_url") || {
    fmt_error "Failed to download $download_url"
    exit 1
  }

  (cd $tmp_dir && do_checksum) || {
    fmt_error "Checksum verification failed"
    exit 1
  }

  # Extract download
  (cd $tmp_dir && tar -xzf "$asset_name") || {
    fmt_error "Failed to extract $asset_name"
    exit 1
  }

  # Install binary
  mv "$tmp_dir/$BINARY_NAME" $INSTALL_DIR
  echo "Installed harific to $INSTALL_DIR"

  # Cleanup
  rm -rf $tmp_dir
}

install_termux() {
  echo "Installing harific, this may take a few minutes..."
  pkg upgrade && pkg install golang git -y && git clone https://github.com/pb33f/harific.git && cd harific/ && go build -o $PREFIX/bin/harific ./cmd/harific
}

main() {
  setup_color

  latest_tag=$(get_version_with_fallback)
  if [ -z "$latest_tag" ]; then
    fmt_error "Failed to determine latest version"
    exit 1
  fi

  latest_version=$(echo $latest_tag | sed 's/v//')
  version=${VERSION:-$latest_version}

  if [ -z "$version" ]; then
    fmt_error "Version could not be determined"
    exit 1
  fi

  echo "Installing harific version: $version"

  os=$(get_os)
  if test -z "$os"; then
    fmt_error "$(uname -s) os type is not supported"
    echo "Please create an issue so we can add support. $ISSUE_URL"
    exit 1
  fi

  machine=$(get_machine)
  if test -z "$machine"; then
    fmt_error "$(uname -m) machine type is not supported"
    echo "Please create an issue so we can add support. $ISSUE_URL"
    exit 1
  fi
  if [ ${TERMUX_VERSION} ] ; then
    install_termux
  else
    do_install_binary
  fi

  printf "$MAGENTA"
  cat <<'EOF'

██╗  ██╗ █████╗ ██████╗ ██╗███████╗██╗ ██████╗
██║  ██║██╔══██╗██╔══██╗██║██╔════╝██║██╔════╝
███████║███████║██████╔╝██║█████╗  ██║██║
██╔══██║██╔══██║██╔══██╗██║██╔══╝  ██║██║
██║  ██║██║  ██║██║  ██║██║██║     ██║╚██████╗
╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚═╝     ╚═╝ ╚═════╝

  harific is now installed, see available commands with:

  harific --help

EOF
  printf "$RESET"

}

main
