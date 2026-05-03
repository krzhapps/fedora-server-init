#!/usr/bin/env python3
"""Download all ebooks from a Humble Bundle library using a session cookie."""

import json
import os
import re
import subprocess
import sys
import time

COOKIE_FILE = os.path.join(os.path.dirname(__file__), "cookie")
DOWNLOAD_DIR = os.path.join(os.path.dirname(__file__), "downloads")
API_BASE = "https://www.humblebundle.com/api/v1"

# Download preference order; all others (Supplement, ZIP) are appended after
FORMAT_PRIORITY = ["EPUB", "PDF", "MOBI"]


def read_cookie():
    with open(COOKIE_FILE) as f:
        return f.readline().strip()


def api_get(path, cookie):
    url = f"{API_BASE}{path}"
    result = subprocess.run(
        [
            "curl", "-sf", "-4",
            "-H", f"Cookie: _simpleauth_sess={cookie}",
            "-H", "Accept: application/json",
            "-H", "User-Agent: humble-downloader/1.0",
            url,
        ],
        capture_output=True,
        check=True,
    )
    return json.loads(result.stdout)


def slugify(name):
    """Turn a human name into a safe filename component."""
    name = re.sub(r'[<>:"/\\|?*]', "", name)
    name = re.sub(r"\s+", " ", name).strip()
    return name[:120]


def pick_formats(download_struct):
    """Return download_struct entries sorted by FORMAT_PRIORITY, others last."""
    priority = {fmt: i for i, fmt in enumerate(FORMAT_PRIORITY)}
    return sorted(
        download_struct,
        key=lambda d: priority.get(d["name"], len(FORMAT_PRIORITY)),
    )


def download_file(url, dest_path):
    # Use curl: avoids Python's 60s IPv6-first connection delay on this host
    tmp_path = dest_path + ".part"
    try:
        result = subprocess.run(
            [
                "curl", "-f", "-L", "--progress-bar",
                "-4",                   # force IPv4, avoids 60s IPv6 timeout
                "-o", tmp_path,
                url,
            ],
            check=True,
        )
        os.rename(tmp_path, dest_path)
    except subprocess.CalledProcessError as e:
        if os.path.exists(tmp_path):
            os.remove(tmp_path)
        raise RuntimeError(f"curl exited with {e.returncode}") from e
    except Exception:
        if os.path.exists(tmp_path):
            os.remove(tmp_path)
        raise


def ext_for(format_name):
    return {
        "EPUB": "epub",
        "PDF": "pdf",
        "MOBI": "mobi",
        "ZIP": "zip",
        "Supplement": "zip",
    }.get(format_name, format_name.lower())


def main():
    only_formats = set(sys.argv[1:]) if sys.argv[1:] else None
    cookie = read_cookie()

    print("Fetching order list…")
    orders = api_get("/user/order", cookie)
    print(f"Found {len(orders)} orders.\n")

    total_downloaded = 0
    total_skipped = 0
    total_errors = 0

    for order_info in orders:
        gamekey = order_info["gamekey"]
        time.sleep(0.5)

        try:
            order = api_get(f"/order/{gamekey}", cookie)
        except Exception as e:
            print(f"[ERROR] Could not fetch order {gamekey}: {e}")
            total_errors += 1
            continue

        bundle_name = slugify(order["product"]["human_name"])
        bundle_dir = os.path.join(DOWNLOAD_DIR, bundle_name)
        os.makedirs(bundle_dir, exist_ok=True)

        print(f"Bundle: {bundle_name}")

        for subproduct in order.get("subproducts", []):
            book_title = slugify(subproduct["human_name"])

            for dl in subproduct.get("downloads", []):
                if dl.get("platform") not in ("ebook", None) and dl.get("platform", "ebook") != "ebook":
                    continue

                structs = pick_formats(dl.get("download_struct", []))

                for ds in structs:
                    fmt = ds["name"]
                    if only_formats and fmt not in only_formats:
                        continue

                    web_url = ds.get("url", {}).get("web")
                    if not web_url:
                        continue

                    ext = ext_for(fmt)
                    filename = f"{book_title}.{ext}"
                    dest = os.path.join(bundle_dir, filename)

                    if os.path.exists(dest):
                        print(f"  [skip] {book_title} ({fmt}): already downloaded")
                        total_skipped += 1
                        # Only skip the first matching preferred format
                        break

                    print(f"  [dl]   {book_title} ({fmt})")
                    try:
                        download_file(web_url, dest)
                        total_downloaded += 1
                    except Exception as e:
                        print(f"    [ERROR] {e}")
                        total_errors += 1
                        continue

                    time.sleep(0.3)
                    # Download only the highest-priority available format
                    break

        print()

    print(f"Done. Downloaded: {total_downloaded}  Skipped: {total_skipped}  Errors: {total_errors}")


if __name__ == "__main__":
    main()
