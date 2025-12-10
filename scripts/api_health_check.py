#!/usr/bin/env python3
"""
Utility script to check API health endpoints and optional Flour vendor endpoint.

Endpoints checked:
- /healthz
- /readyz
- /api/flour/vendors/license/{licenseID}/ (optional, requires token)
"""

import argparse
import os
import sys
from typing import Optional, Tuple

import requests

DEFAULT_TIMEOUT = 5


def _normalize_base_url(base_url: str) -> str:
    return base_url.rstrip('/')


def _get(session: requests.Session, url: str, token: Optional[str] = None) -> requests.Response:
    headers = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    return session.get(url, headers=headers, timeout=DEFAULT_TIMEOUT)


def check_health(session: requests.Session, base_url: str) -> Tuple[bool, str]:
    url = f"{_normalize_base_url(base_url)}/healthz"
    resp = _get(session, url)
    if resp.status_code == 200:
        return True, "OK"
    return False, f"{resp.status_code} {resp.text[:100]}"


def check_ready(session: requests.Session, base_url: str) -> Tuple[bool, str]:
    url = f"{_normalize_base_url(base_url)}/readyz"
    resp = _get(session, url)
    if resp.status_code == 200:
        return True, "OK"
    return False, f"{resp.status_code} {resp.text[:100]}"


def check_flour_vendor(
    session: requests.Session,
    base_url: str,
    license_id: str,
    token: Optional[str]
) -> Tuple[bool, str]:
    url = f"{_normalize_base_url(base_url)}/api/flour/vendors/license/{license_id}/"
    resp = _get(session, url, token)
    if resp.status_code == 200:
        return True, "OK"
    if resp.status_code == 401:
        return False, "Unauthorized"
    if resp.status_code == 404:
        return False, "Not Found"
    return False, f"{resp.status_code} {resp.text[:100]}"


def main() -> None:
    parser = argparse.ArgumentParser(description="Check API health and Flour endpoints")
    parser.add_argument("--base-url", default="http://localhost:3000", help="Base API URL")
    parser.add_argument("--token", help="Bearer token for authenticated endpoints")
    parser.add_argument(
        "--flour-license-id",
        help="License ID to query Flour vendor endpoint (optional)",
    )
    parser.add_argument(
        "--skip-flour",
        action="store_true",
        help="Skip Flour endpoint check",
    )
    args = parser.parse_args()

    session = requests.Session()
    base_url = args.base_url
    token = args.token or os.environ.get("AUTH_TOKEN")

    print(f"Checking health at {base_url}/healthz ...", end=" ")
    ok, msg = check_health(session, base_url)
    print("OK" if ok else f"FAIL ({msg})")

    print(f"Checking ready at {base_url}/readyz ...", end=" ")
    ok, msg = check_ready(session, base_url)
    print("OK" if ok else f"FAIL ({msg})")

    if not args.skip_flour and args.flour_license_id:
        print(
            f"Checking flour vendor {args.flour_license_id} at {base_url}/api/flour/vendors/license/{{id}}/ ...",
            end=" ",
        )
        ok, msg = check_flour_vendor(session, base_url, args.flour_license_id, token)
        print("OK" if ok else f"FAIL ({msg})")
    elif not args.skip_flour:
        print("Skipping Flour check: no license ID provided. Use --flour-license-id to enable.")


if __name__ == "__main__":
    try:
        main()
    except requests.exceptions.RequestException as exc:
        print(f"Request failed: {exc}")
        sys.exit(1)
