#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BACKEND_FILE="$ROOT_DIR/internal/service/ai/routes.go"
FRONTEND_FILE="$ROOT_DIR/web/src/api/modules/ai.ts"

usage() {
  cat <<'EOF'
Usage: script/ai/check_contract_parity.sh [--backend FILE] [--frontend FILE]

Compares registered backend AI routes with implemented frontend aiApi endpoints.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --backend)
      BACKEND_FILE="${2:-}"
      shift 2
      ;;
    --frontend)
      FRONTEND_FILE="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

python3 - "$BACKEND_FILE" "$FRONTEND_FILE" <<'PY'
import re
import sys
from pathlib import Path

backend_path = Path(sys.argv[1])
frontend_path = Path(sys.argv[2])


def die(message: str, code: int = 1) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(code)


def normalize_frontend_path(raw: str) -> str:
    path = raw
    path = path.replace("${base}", "")
    path = re.sub(r"\$\{([A-Za-z_][A-Za-z0-9_]*)\}", lambda m: f":{m.group(1)}", path)
    path = re.sub(r"\$\{[^}]+\}", ":param", path)
    return path


def join_route(prefix: str, suffix: str) -> str:
    if not prefix.startswith("/"):
        prefix = "/" + prefix
    if suffix.startswith("/"):
        return prefix + suffix
    return prefix + "/" + suffix


def extract_backend_routes(text: str) -> set[str]:
    group_match = re.search(r'\.Group\("([^"]+)"', text)
    if not group_match:
        die(f"unable to find backend route group in {backend_path}")

    base_prefix = "/api/v1" + group_match.group(1)
    routes: set[str] = set()

    for method, path in re.findall(r'\b(?:[A-Za-z_][A-Za-z0-9_]*)\.(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|ANY)\("([^"]+)"', text):
        if method == "ANY":
            routes.add(f"ANY {join_route(base_prefix, path)}")
            continue
        routes.add(f"{method} {join_route(base_prefix, path)}")

    if not routes:
        die(f"no backend AI routes found in {backend_path}")

    return routes


def extract_frontend_routes(text: str) -> set[str]:
    routes: set[str] = set()

    for method, _, path in re.findall(r'apiService\.(get|post|put|delete|patch)\(\s*([`\'"])(.*?)\2', text):
        routes.add(f"{method.upper()} {normalize_frontend_path(path)}")

    if re.search(r'fetch\(\s*`(?:\$\{base\})?/ai/chat`', text):
        routes.add("POST /ai/chat")

    if not routes:
        die(f"no frontend aiApi routes found in {frontend_path}")

    return routes


backend_routes = extract_backend_routes(backend_path.read_text(encoding="utf-8"))
frontend_routes = extract_frontend_routes(frontend_path.read_text(encoding="utf-8"))

def normalize_route(route: str) -> str:
    return re.sub(r"^([A-Z]+) /api/v1", r"\1 ", route)


normalized_backend = {normalize_route(route) for route in backend_routes}
normalized_frontend = {normalize_route(route) for route in frontend_routes}

backend_only = sorted(normalized_backend - normalized_frontend)
frontend_only = sorted(normalized_frontend - normalized_backend)

if backend_only or frontend_only:
    print("AI contract parity check failed.", file=sys.stderr)
    print(f"Backend file: {backend_path}", file=sys.stderr)
    print(f"Frontend file: {frontend_path}", file=sys.stderr)
    print("", file=sys.stderr)
    if backend_only:
        print("Routes registered in backend but missing from frontend:", file=sys.stderr)
        for route in backend_only:
            print(f"  {route}", file=sys.stderr)
    if frontend_only:
        print("Routes implemented in frontend but missing from backend:", file=sys.stderr)
        for route in frontend_only:
            print(f"  {route}", file=sys.stderr)
    raise SystemExit(1)

print(f"AI contract parity check passed: {len(normalized_backend)} backend routes match {len(normalized_frontend)} frontend endpoints.")
PY
