#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT="${PORT:-8090}"
DATA_DIR="${PB_DATA_DIR:-$ROOT_DIR/pb_data}"
PUBLIC_DIR="$ROOT_DIR/pocketbase/pb_public"

command -v npm >/dev/null 2>&1 || { echo "Error: npm no está instalado." >&2; exit 1; }
command -v go >/dev/null 2>&1 || { echo "Error: Go no está instalado." >&2; exit 1; }

cd "$ROOT_DIR"
echo "Compilando frontend en $PUBLIC_DIR..."
npm run build

echo "Arrancando PocketBase en http://127.0.0.1:$PORT"
echo "Base de datos: $DATA_DIR"
exec env PAY_PUBLIC_DIR="$PUBLIC_DIR" go -C "$ROOT_DIR/pocketbase" run . serve \
  --http="127.0.0.1:$PORT" \
  --dir="$DATA_DIR"
