"""Validate the administrative permission seed before it is imported."""

import argparse
import json
from pathlib import Path


def validate_seed(seed):
    """Reject ambiguous routes, malformed trees, and invalid role grants."""
    if seed.get("schemaVersion") != 1:
        raise ValueError("unsupported seed schema version")
    nodes = seed["nodes"]
    by_code = {}
    permissions = set()
    routes = set()
    endpoints = set()
    for node in nodes:
        code = node["code"]
        if not code or code in by_code:
            raise ValueError(f"duplicate or empty node code: {code}")
        by_code[code] = node
        kind = node["type"]
        if kind not in {"directory", "menu", "action"}:
            raise ValueError(f"invalid node type: {code}")
        permission = node.get("permissionCode")
        if kind == "directory" and (permission or node.get("route") or node["apis"]):
            raise ValueError(f"directory carries executable metadata: {code}")
        if kind == "action" and (not permission or node.get("route")):
            raise ValueError(f"invalid action metadata: {code}")
        if kind == "menu" and not node.get("route"):
            raise ValueError(f"menu has no route: {code}")
        if permission:
            if permission in permissions:
                raise ValueError(f"duplicate permission: {permission}")
            permissions.add(permission)
        elif kind == "menu" and node.get("visibility") not in {
            "authenticated", "authenticated_self_service"
        }:
            raise ValueError(f"menu has no access rule: {code}")
        route = node.get("route")
        if route:
            if not route.startswith("/") or route.startswith("//") or route in routes:
                raise ValueError(f"invalid or duplicate menu route: {code}")
            routes.add(route)
        if node["apis"] and not permission:
            raise ValueError(f"business API has no permission: {code}")
        for endpoint in node["apis"]:
            _register_endpoint(endpoints, endpoint)
    for node in nodes:
        parent = node["parentCode"]
        if node["type"] == "action" and (
            parent not in by_code or by_code[parent]["type"] != "menu"
        ):
            raise ValueError(f"action must belong to menu: {node['code']}")
        seen = {node["code"]}
        while parent is not None:
            if parent not in by_code:
                raise ValueError(f"missing parent: {parent}")
            if parent in seen:
                raise ValueError(f"permission tree cycle: {parent}")
            if by_code[parent]["type"] == "action":
                raise ValueError(f"action cannot have children: {parent}")
            seen.add(parent)
            parent = by_code[parent]["parentCode"]
    roles = {role["code"]: role for role in seed["roles"]}
    if len(roles) != len(seed["roles"]) or set(roles) != {
        "super_admin", "content_admin", "viewer"
    }:
        raise ValueError("default roles missing or duplicated")
    for role in roles.values():
        grants = role["permissions"]
        if len(set(grants)) != len(grants) or not set(grants) <= permissions:
            raise ValueError(f"invalid role grants: {role['code']}")
        if role["isSuper"] != (role["code"] == "super_admin"):
            raise ValueError("invalid super administrator role")
    if set(roles["super_admin"]["permissions"]) != permissions:
        raise ValueError("super administrator grants are incomplete")
    if any(not p.endswith(".read") for p in roles["viewer"]["permissions"]):
        raise ValueError("viewer has write permission")
    self_codes = set()
    for entry in seed["selfService"]:
        if entry["code"] in self_codes:
            raise ValueError("duplicate self service entry")
        self_codes.add(entry["code"])
        scopes = entry["allowedScopes"]
        if not scopes or not set(scopes) <= {"full", "onboarding", "password_change"}:
            raise ValueError("invalid self service scope")
        for endpoint in entry["apis"]:
            _register_endpoint(endpoints, endpoint)
    anonymous = set()
    for endpoint in seed["anonymousApis"]:
        _register_endpoint(endpoints, endpoint, public=True)
        anonymous.add((endpoint["method"], endpoint["path"]))
    if anonymous != {
        ("POST", "/admin/api/v1/login"),
        ("POST", "/admin/api/v1/login/totp"),
        ("GET", "/api/v1/contact"),
        ("GET", "/api/v1/whatsapp-status"),
    }:
        raise ValueError("unexpected anonymous administrative route")
    return len(nodes), len(permissions), len(roles)


def _register_endpoint(endpoints, endpoint, public=False):
    """Enforce one administrative authorization owner per method and path."""
    method, path = endpoint["method"], endpoint["path"]
    if method not in {"GET", "POST", "PUT", "PATCH", "DELETE"}:
        raise ValueError(f"invalid HTTP method: {method}")
    valid_prefix = path.startswith("/admin/api/v1/") or (
        public and method == "GET" and path.startswith("/api/v1/")
    )
    if not valid_prefix or ".." in path or "?" in path:
        raise ValueError(f"invalid administrative API: {path}")
    key = method, path
    if key in endpoints:
        raise ValueError(f"duplicate administrative API: {method} {path}")
    endpoints.add(key)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "path", nargs="?", type=Path,
        default=Path(__file__).resolve().parents[1]
        / "backend/internal/data/seed/admin_permissions.json",
    )
    args = parser.parse_args()
    try:
        counts = validate_seed(json.loads(args.path.read_text()))
    except (OSError, ValueError, KeyError, TypeError) as error:
        parser.exit(1, f"Invalid permission seed: {error}\n")
    print(f"Permission seed valid: {counts[0]} nodes, {counts[1]} permissions, {counts[2]} roles")


if __name__ == "__main__":
    main()
