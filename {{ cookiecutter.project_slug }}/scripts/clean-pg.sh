#!/bin/sh
# Remove embedded-Postgres temp directories left behind by runs that were
# killed before their Stop could run. A directory whose postmaster is still
# alive is reported and kept, because a dev server may have adopted it via
# the port-reuse path. FORCE=1 stops those first.
set -u

tmp="${TMPDIR:-/tmp}"
removed=0
live=0
freed_kb=0

for d in "$tmp"/embeddedpg-data-* "$tmp"/embeddedpg-rt-*; do
    [ -d "$d" ] || continue

    pid=''
    if [ -f "$d/postmaster.pid" ]; then
        pid=$(head -1 "$d/postmaster.pid" 2>/dev/null | tr -dc '0-9')
    fi

    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
        if [ "${FORCE:-0}" != "1" ]; then
            echo "  in use, keeping: $d (postmaster $pid)"
            live=$((live + 1))
            continue
        fi
        echo "  stopping postmaster $pid"
        kill -TERM "$pid" 2>/dev/null || true
        n=0
        while [ "$n" -lt 20 ] && kill -0 "$pid" 2>/dev/null; do
            sleep 0.5
            n=$((n + 1))
        done
        if kill -0 "$pid" 2>/dev/null; then
            kill -KILL "$pid" 2>/dev/null || true
            sleep 0.5
        fi
    fi

    # A runtime directory still holding a socket belongs to a live instance.
    if [ "${FORCE:-0}" != "1" ] && ls "$d"/.s.PGSQL.* >/dev/null 2>&1; then
        echo "  in use, keeping: $d (socket)"
        live=$((live + 1))
        continue
    fi

    kb=$(du -sk "$d" 2>/dev/null | cut -f1)
    case "$kb" in ''|*[!0-9]*) kb=0 ;; esac
    if rm -rf "$d"; then
        removed=$((removed + 1))
        freed_kb=$((freed_kb + kb))
    fi
done

echo "removed $removed dir(s), ~$((freed_kb / 1024)) MB; $live left in use"
if [ "$live" -gt 0 ]; then
    echo "re-run as 'task clean:pg FORCE=1' to stop those and reclaim them"
fi
