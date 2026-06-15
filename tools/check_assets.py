#!/usr/bin/env python3
"""Find missing icons/assets in the running SMF Go port.

Methodology: log in, BFS-crawl the forum following its own index.php links
(skipping destructive GET actions), and for every page collect referenced
assets -- <img src>, stylesheet <link href>, <script src>, and url(...) refs
inside the fetched CSS. Each unique asset URL is requested once; anything that
is not HTTP 200 is reported along with the page(s) that referenced it.

Usage:
    python3 check_assets.py [base_url] [user] [pass]
Defaults: http://127.0.0.1:8080  admin  admin
"""
import sys, re, collections
from urllib.parse import urljoin, urldefrag, urlsplit
from urllib.request import Request, urlopen, HTTPCookieProcessor, build_opener
from urllib.error import HTTPError, URLError
from http.cookiejar import CookieJar

BASE = (sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:8080").rstrip("/")
USER = sys.argv[2] if len(sys.argv) > 2 else "admin"
PASS = sys.argv[3] if len(sys.argv) > 3 else "admin"
HOST = urlsplit(BASE).netloc
MAX_PAGES = 400

opener = build_opener(HTTPCookieProcessor(CookieJar()))
opener.addheaders = [("User-Agent", "asset-check/1.0")]

# Don't follow GET links that could mutate state under an admin session.
DESTRUCTIVE = re.compile(
    r"(action=logout|action=theme|[?;]sa=pick|[?;]th=|[?;]vt=|"
    r"[?;](sa|area)=[^;&]*?(delete|remove|kill|prune|empty|clear|"
    r"ban|approve|reject|deactivate|merge|split|lock|sticky|makeadmin|restore|"
    r"maintain|dumpdb|backup|repair)|;do=|delete=|remove=|;save)", re.I)

def fetch(url):
    try:
        with opener.open(Request(url), timeout=15) as r:
            ct = r.headers.get_content_type()
            body = r.read() if ct.startswith(("text/", "application/")) else b""
            return r.status, ct, body.decode("utf-8", "replace")
    except HTTPError as e:
        return e.code, "", ""
    except (URLError, Exception) as e:
        return 0, str(e), ""

def login():
    fetch(f"{BASE}/index.php?action=login")  # prime session cookie
    from urllib.parse import urlencode
    data = urlencode({"user": USER, "passwrd": PASS,
                      "cookielength": "60", "hash_passwrd": ""}).encode()
    try:
        opener.open(Request(f"{BASE}/index.php?action=login2", data=data), timeout=15)
    except HTTPError:
        pass  # 302 is success

IMG  = re.compile(r"""<img\b[^>]*?\bsrc\s*=\s*["']([^"']+)["']""", re.I)
CSS  = re.compile(r"""<link\b[^>]*?\bhref\s*=\s*["']([^"']+\.css[^"']*)["']""", re.I)
JS   = re.compile(r"""<script\b[^>]*?\bsrc\s*=\s*["']([^"']+)["']""", re.I)
BG   = re.compile(r"""\bbackground(?:-image)?\s*[:=]\s*["']?[^;"'>]*url\(['"]?([^)'"]+)""", re.I)
HREF = re.compile(r"""\bhref\s*=\s*["']([^"']+)["']""", re.I)
CSSURL = re.compile(r"""url\(\s*['"]?([^)'"]+)""", re.I)

def same_host(u):
    return urlsplit(u).netloc in ("", HOST)

# ---- crawl pages, gathering asset refs (asset_url -> set of referrer pages) ----
assets = collections.defaultdict(set)
seen_pages, queue = set(), collections.deque([f"{BASE}/index.php"])
css_files = set()

login()
pages = 0
while queue and pages < MAX_PAGES:
    page = queue.popleft()
    key = urldefrag(page)[0]
    if key in seen_pages:
        continue
    seen_pages.add(key)
    status, ct, html = fetch(page)
    if "html" not in ct:
        continue
    pages += 1
    for rx in (IMG, CSS, JS, BG):
        for m in rx.finditer(html):
            a = urldefrag(urljoin(page, m.group(1)))[0]
            if same_host(a):
                assets[a].add(page)
                if a.endswith(".css") or ".css?" in a:
                    css_files.add(a)
    for m in HREF.finditer(html):
        link = urljoin(page, m.group(1))
        lk = urldefrag(link)[0]
        if (same_host(link) and "index.php" in lk and lk not in seen_pages
                and not DESTRUCTIVE.search(link)):
            queue.append(link)

# ---- also mine url() refs inside every stylesheet we saw ----
for css in css_files:
    status, ct, body = fetch(css)
    for m in CSSURL.finditer(body):
        ref = m.group(1).strip()
        if ref.startswith(("data:", "http")) and not ref.startswith(BASE):
            continue
        a = urldefrag(urljoin(css, ref))[0]
        if same_host(a):
            assets[a].add(css)

# ---- check every unique asset once ----
missing = {}
for url in sorted(assets):
    status, ct, _ = fetch(url)
    if status != 200:
        missing[url] = (status, assets[url])

print(f"crawled {pages} pages, checked {len(assets)} unique assets, "
      f"{len(missing)} missing\n")
if missing:
    for url in sorted(missing):
        status, refs = missing[url]
        ref = sorted(refs)[0].replace(BASE, "")
        print(f"  [{status}] {url.replace(BASE,'')}")
        print(f"          first referenced by: {ref}  (+{len(refs)-1} more)")
    sys.exit(1)
else:
    print("No missing assets. ✓")
