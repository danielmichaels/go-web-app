// Service worker: a deliberately conservative starting point.
//
// It caches the static shell only. HTML and anything under /app are always
// fetched from the network, because a Datastar app patches live state and a
// cached page would show a stale view with no way to know it.
//
// Bump CACHE_VERSION whenever the precache list changes; the old cache is
// dropped on activate.
const CACHE_VERSION = "v1";
const CACHE_NAME = `static-${CACHE_VERSION}`;

const PRECACHE = ["/static/css/main.css", "/static/js/datastar.js"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(CACHE_NAME)
      // Individually, so one missing file does not fail the whole install.
      .then((cache) => Promise.allSettled(PRECACHE.map((url) => cache.add(url))))
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k))),
      )
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const { request } = event;

  // Only same-origin GETs of static assets are served from the cache.
  if (request.method !== "GET") return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  if (!url.pathname.startsWith("/static/")) return;

  event.respondWith(
    caches.match(request).then(
      (hit) =>
        hit ||
        fetch(request).then((response) => {
          if (response.ok) {
            const copy = response.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(request, copy));
          }
          return response;
        }),
    ),
  );
});
