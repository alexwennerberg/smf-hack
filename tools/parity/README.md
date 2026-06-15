# PHP reference parity harness

Byte-diffs the Go port's committed goldens (`internal/app/testdata/golden/`)
against a live **SMF 1.1.21 on php:5.6-apache + mysql:5.5** instance. This is
verification strategy #1/#2 from the plan: a real PHP oracle, the same
goldenApp seed, the same URL crawl, the same normalizers.

Result at last run: **59/64 pages byte-identical**; the other 5 are documented
design decisions / harness artifacts (see PORTING.md "Phase 8 — PHP REFERENCE
PARITY").

## One-time setup

Requires docker (the daemon must be running). All commands assume a scratch
dir `/tmp/smf-php` with `www/` = a writable copy of the SMF repo root.

```sh
# 1. network + MySQL 5.5
docker network create smfnet
docker run -d --name smf-mysql --network smfnet -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=smf mysql:5.5

# 2. PHP 5.6 image with the mysql + gd extensions (see Dockerfile)
mkdir -p /tmp/smf-php/www && cp -a <repo>/{SSI.php,Settings.php,Smileys,Sources,Themes,agreement.txt,attachments,avatars,index.php,install.php,install_1-1.sql,Packages,...} /tmp/smf-php/www/
chmod -R 0777 /tmp/smf-php/www
cp Dockerfile /tmp/smf-php/ && (cd /tmp/smf-php && docker build -t smf-php:5.6 .)
docker run -d --name smf-web --network smfnet -p 8080:80 -v /tmp/smf-php/www:/var/www/html smf-php:5.6

# 3. drive the real installer (boardurl MUST match the Go test: 127.0.0.1:8080)
curl -s -c jar "http://127.0.0.1:8080/install.php?lang_file=Install.english.php" -o /dev/null
curl -s -b jar -c jar "http://127.0.0.1:8080/install.php?step=1" \
  --data-urlencode boardurl=http://127.0.0.1:8080 --data-urlencode db_server=smf-mysql \
  --data-urlencode db_name=smf --data-urlencode db_user=root --data-urlencode db_passwd=root \
  --data-urlencode db_prefix=smf_ --data-urlencode "mbname=My Community" -o /dev/null
curl -s -b jar "http://127.0.0.1:8080/install.php?step=2" \
  --data-urlencode username=admin --data-urlencode password1=testpass --data-urlencode password2=testpass \
  --data-urlencode email=a@b.com --data-urlencode password3=root -o /dev/null

# 4. post-install: remove installer, align cookiename to the Go default, apply seed,
#    copy in the parity helpers, set period-correct PHP + skip the /e E_DEPRECATED
rm -f /tmp/smf-php/www/install.php /tmp/smf-php/www/install_1-1.sql
sed -i "s/\$cookiename = '[^']*'/\$cookiename = 'SMFCookie11'/" /tmp/smf-php/www/Settings.php
docker exec -i smf-mysql mysql -uroot -proot smf < seed.sql
cp mkcookie.php clearonline.php /tmp/smf-php/www/
docker exec smf-web bash -c 'printf "display_errors = Off\nlog_errors = On\nerror_reporting = E_ALL & ~E_DEPRECATED & ~E_STRICT & ~E_NOTICE\n" > /usr/local/etc/php/conf.d/zz-parity.ini && apachectl -k graceful'
# Patch the container's Sources/Errors.php error_handler to also early-return on
# E_DEPRECATED (SMF 1.1.21 predates the PHP 5.5 /e-modifier deprecation, so
# period-correct PHP would not have logged it). Add to the guard at the top of
# error_handler():  || (defined('E_DEPRECATED') && $error_level == E_DEPRECATED)

# 5. mint per-role login cookies (urlencoded) used by the crawler
for id in 1 2 3; do
  curl -s "http://127.0.0.1:8080/mkcookie.php?id=$id" | python3 -c 'import urllib.parse,sys;print(urllib.parse.quote(sys.stdin.read()))' > /tmp/smf-php/cookie_$id.txt
done
```

## Running

```sh
cp crawl.go reset.sql run.sh /tmp/smf-php/   # crawl.go/run.sh expect /tmp/smf-php paths
/tmp/smf-php/run.sh                          # resets volatile state, then crawls + diffs
```

`reset.sql` restores the goldenApp baseline (numViews, most-online, per-user
read/notify logs, error log) so every run is idempotent. `crawl.go` normalizes
the same volatile bytes as `golden_test.go` plus the docker client IP
(172.18.0.1 → the Go httptest RemoteAddr 192.0.2.1).

## Files

- `Dockerfile` — php:5.6-apache + mysql & gd exts (archive.debian.org apt).
- `seed.sql` — goldenApp's seed, as MySQL inserts (applied once after install).
- `reset.sql` — per-run reset of volatile state to the seed baseline.
- `crawl.go` — fetch the 64-URL golden set, normalize, diff vs committed goldens.
- `run.sh` — reset + crawl.
- `mkcookie.php` / `clearonline.php` — in-container helpers (login cookie mint,
  log_online clear).

## Write-flow parity

`writeflow_crawl.go` (run from /tmp/smf-php via `go run writeflow_crawl.go`)
performs mutations on the live PHP instance and diffs the rendered result
against `testdata/golden_writeflow/`. It restores a clean seed snapshot before
each scenario; generate the snapshot once after seeding + a baseline reset:

```sh
docker exec -i smf-mysql mysql -uroot -proot smf < reset.sql
docker exec smf-mysql sh -c 'mysqldump -uroot -proot smf' > /tmp/smf-php/snapshot.sql
```

The read crawl (`crawl.go`) also restores this snapshot before its run so
write-flow rows (e.g. a sent PM) don't leak into read goldens.
