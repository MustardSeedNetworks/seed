#!/bin/bash
#
# sync-dev-env.sh
#
# This script automates the process of keeping the development environments
# on both the local and remote systems up-to-date, and verifies the versions.
#
# It handles:
# - Go, Node.js, and Python versions
# - NPM and Go dependencies
#
# Usage:
# ./scripts/sync-dev-env.sh [remote_host1] [remote_host2] ...
#

# --- Configuration ---
REMOTE_USER="krisarmstrong"
REMOTE_PROJECT_PATH="/home/krisarmstrong/seed"
PYTHON_VERSION="3.14.2"
NODE_VERSION="25.2.1"
GO_VERSION="1.25.5"

# --- Helper Functions ---
print_info() {
  echo "INFO: $1"
}

print_error() {
  echo "ERROR: $1" >&2
}

# --- Verification Function ---
verify_system() {
  local system_name=$1
  print_info "Verifying system: $system_name"

  print_info "Go version:"
  go version
  print_info "Node version:"
  node --version
  print_info "npm version:"
  npm --version
  print_info "Python version:"
  python --version

  print_info "Checking for outdated Go dependencies..."
  go list -u -m all

  print_info "Checking for outdated npm dependencies in web project..."
  (cd web && npm outdated)
  
  print_info "Checking for outdated npm dependencies in root project..."
  npm outdated
}


# --- Local System Update & Verification ---
update_and_verify_local() {
  print_info "Updating local system..."

  # Update Homebrew and packages
  print_info "Updating Homebrew and upgrading go and node..."
  brew update
  brew upgrade go node

  # Update global npm
  print_info "Updating global npm packages..."
  npm install -g npm@latest

  # Update project dependencies
  print_info "Updating Go dependencies..."
  go get -u ./...
  print_info "Updating web project npm dependencies..."
  (cd web && npm update)

  # Check and install Python
  print_info "Checking Python version..."
  if ! command -v pyenv &> /dev/null;
    then
    print_info "pyenv not found. Installing..."
    brew install pyenv
  fi
  
  eval "$(pyenv init -)"
  if ! pyenv versions --bare | grep -q "$PYTHON_VERSION"; then
    print_info "Installing Python $PYTHON_VERSION..."
    pyenv install "$PYTHON_VERSION"
  fi
  pyenv global "$PYTHON_VERSION"
  
  verify_system "local"
}

# --- Remote System Update & Verification ---
update_and_verify_remote() {
  local remote_host=$1
  print_info "Updating remote system: $remote_host..."

  ssh "$REMOTE_USER@$remote_host" << EOF
    # --- Remote Helper Functions ---
    print_info() {
      echo "REMOTE INFO: $1"
    }

    print_error() {
      echo "REMOTE ERROR: $1" >&2
    }

    # --- OS Detection ---
    OS=$(uname -s)
    print_info "Detected OS: $OS"

    # --- Install Dependencies ---
    install_deps() {
      print_info "Installing dependencies..."
      if [ "$OS" == "Linux" ]; then
        if command -v apt-get &> /dev/null;
          then
          sudo apt-get update
          sudo apt-get install -y make build-essential libssl-dev zlib1g-dev \
            libbz2-dev libreadline-dev libsqlite3-dev wget curl llvm \
            libncurses5-dev libncursesw5-dev xz-utils tk-dev libffi-dev liblzma-dev mercurial
        elif command -v dnf &> /dev/null;
          then
          sudo dnf install -y make automake gcc gcc-c++ kernel-devel \
            zlib-devel bzip2 bzip2-devel readline-devel sqlite sqlite-devel \
            openssl-devel tk-devel libffi-devel xz-devel mercurial
        else
          print_error "Unsupported Linux distribution."
          exit 1
        fi
      elif [ "$OS" == "Darwin" ]; then
        # Assuming Homebrew is installed on remote macOS
        brew update
        brew install mercurial
      else
        print_error "Unsupported OS: $OS"
        exit 1
      fi
    }

    # --- Main Remote Logic ---
    install_deps

    print_info "Updating Node.js and npm..."
    source ~/.nvm/nvm.sh
    if ! nvm --version | grep -q '0.39.7'; then
        curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
        source ~/.nvm/nvm.sh
    fi
    nvm install "$NODE_VERSION"
    nvm use "$NODE_VERSION"
    nvm alias default "$NODE_VERSION"
    npm install -g npm@latest

    print_info "Updating Go..."
    if ! command -v gvm &> /dev/null;
      then
        bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
    fi
    source ~/.gvm/scripts/gvm
    gvm install "$GO_VERSION" -B
    gvm use "$GO_VERSION" --default

    print_info "Updating Go dependencies..."
    cd "$REMOTE_PROJECT_PATH"
    go get -u ./...

    print_info "Updating web project npm dependencies..."
    cd "$REMOTE_PROJECT_PATH/web"
    npm update

    print_info "Checking Python version..."
    if ! command -v pyenv &> /dev/null;
      then
      print_info "pyenv not found. Installing..."
      curl https://pyenv.run | bash
    fi
    export PATH="$HOME/.pyenv/bin:$PATH"
    eval "$(pyenv init -)"
    
    if ! pyenv versions --bare | grep -q "$PYTHON_VERSION"; then
      print_info "Installing Python $PYTHON_VERSION..."
      pyenv install "$PYTHON_VERSION"
    fi
    pyenv global "$PYTHON_VERSION"

    # --- Verification ---
    print_info "Verifying remote system: $remote_host"

    print_info "Go version:"
    go version
    print_info "Node version:"
    node --version
    print_info "npm version:"
    npm --version
    print_info "Python version:"
    python --version

    print_info "Checking for outdated Go dependencies..."
    cd "$REMOTE_PROJECT_PATH"
    go list -u -m all

    print_info "Checking for outdated npm dependencies in web project..."
    cd "$REMOTE_PROJECT_PATH/web"
    npm outdated
    
    print_info "Checking for outdated npm dependencies in root project..."
    npm outdated
EOF
}

# --- Main ---
main() {
  if [ "$#" -eq 0 ]; then
    print_info "No remote hosts provided. Only updating and verifying local system."
    update_and_verify_local
    print_info "Local update and verification complete!"
    exit 0
  fi

  update_and_verify_local

  for remote_host in "$@"; do
    update_and_verify_remote "$remote_host"
  done

  print_info "Sync and verification complete!"
}

main "$@"
