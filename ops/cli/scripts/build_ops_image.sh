#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
IMAGE_NAME="${1:-flashsale-app-ops-control}"
BUILD_DIR="$REPO_ROOT/.memory/ops-build"
FRONTEND_DIR="$REPO_ROOT/frontend/ops"
EMBED_DIR="$REPO_ROOT/ops/web"

if command -v go.exe >/dev/null 2>&1; then
    GO_BIN="go.exe"
else
    GO_BIN="go"
fi

require_cmd() {
    local name="$1"
    if ! command -v "$name" >/dev/null 2>&1; then
        echo "missing required command: $name" >&2
        exit 1
    fi
}

require_cmd docker
require_cmd bun
require_cmd "$GO_BIN"

to_windows_path() {
    local path="$1"
    if command -v wslpath >/dev/null 2>&1; then
        wslpath -w "$path"
        return 0
    fi
    if command -v cygpath >/dev/null 2>&1; then
        cygpath -w "$path"
        return 0
    fi

    echo "missing path conversion tool: wslpath/cygpath" >&2
    exit 1
}

if ! docker image inspect flashsale-backend:local >/dev/null 2>&1; then
    echo "missing required local image: flashsale-backend:local" >&2
    exit 1
fi

mkdir -p "$BUILD_DIR"

if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
    (
        cd "$FRONTEND_DIR"
        bun install --frozen-lockfile
    )
fi

(
    cd "$FRONTEND_DIR"
    bun x vite build
)

rm -rf "$EMBED_DIR"
mkdir -p "$EMBED_DIR"
cp -R "$FRONTEND_DIR/dist/." "$EMBED_DIR/"
cat > "$EMBED_DIR/.gitignore" <<'EOF'
*
!.gitignore
EOF

if [[ "$GO_BIN" == "go.exe" ]] && command -v pwsh.exe >/dev/null 2>&1; then
    REPO_ROOT_WIN="$(to_windows_path "$REPO_ROOT")"
    BUILD_OUT_WIN="$(to_windows_path "$BUILD_DIR/fs")"
    BUILD_PS1="$BUILD_DIR/build_fs.ps1"
    cat > "$BUILD_PS1" <<EOF
\$ErrorActionPreference = 'Stop'
\$env:GOTOOLCHAIN = 'local'
\$env:GOOS = 'linux'
\$env:GOARCH = 'amd64'
\$env:GOPROXY = '${GOPROXY:-https://goproxy.cn|https://proxy.golang.org|direct}'
\$env:CGO_ENABLED = '0'
Set-Location '$REPO_ROOT_WIN'
go.exe build -buildvcs=false -trimpath -o '$BUILD_OUT_WIN' ./ops/cmd
EOF
    pwsh.exe -NoProfile -File "$(to_windows_path "$BUILD_PS1")"
else
    (
        cd "$REPO_ROOT"
        GOTOOLCHAIN=local \
        GOOS=linux \
        GOARCH=amd64 \
        GOPROXY="${GOPROXY:-https://goproxy.cn|https://proxy.golang.org|direct}" \
        CGO_ENABLED=0 \
        "$GO_BIN" build -buildvcs=false -trimpath -o "$BUILD_DIR/fs" ./ops/cmd
    )
fi

(
    cd "$REPO_ROOT"
    docker buildx build \
        --file deploy/docker/ops.local.Dockerfile \
        --tag "$IMAGE_NAME" \
        --progress=plain \
        --pull=false \
        --load \
        .
)
