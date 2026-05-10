var CACHE = 'hn-digest-v1';
var STATIC_URLS = [
  '/static/css/style.css',
  '/static/css/bootstrap.min.css',
  '/static/css/font-awesome.min.css',
  '/static/js/hn.js',
  '/static/js/jquery.scrollUp.min.js',
  '/static/js/jquery.lazyload.min.js',
  '/static/js/html5shiv.min.js',
  '/static/js/respond.min.js',
  '/static/favicon.ico',
  '/static/apple-touch-icon.png',
  '/static/icon-200.png',
  '/static/manifest.json'
];

self.addEventListener('install', function(e) {
  e.waitUntil(
    caches.open(CACHE).then(function(cache) {
      return cache.addAll(STATIC_URLS);
    })
  );
  self.skipWaiting();
});

self.addEventListener('activate', function(e) {
  e.waitUntil(
    caches.keys().then(function(keys) {
      return Promise.all(
        keys.filter(function(k) { return k !== CACHE; }).map(function(k) { return caches.delete(k); })
      );
    })
  );
  self.clients.claim();
});

self.addEventListener('fetch', function(e) {
  var url = new URL(e.request.url);
  if (url.origin !== location.origin) return;
  if (url.pathname === '/' || url.pathname.startsWith('/daily/') || url.pathname.startsWith('/image/')) {
    e.respondWith(
      fetch(e.request).catch(function() { return caches.match(e.request); })
    );
    return;
  }
  e.respondWith(
    caches.match(e.request).then(function(r) { return r || fetch(e.request).catch(function() {}); })
  );
});
