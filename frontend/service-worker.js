const CACHE_NAME = 'program-checker-swr-v1';
const urlsToCache = [
    '/',
    '/index.html',
    '/manifest.json',
    '/icon.svg',
    '/icon-192.png',
    '/icon-512.png'
];

self.addEventListener('install', event => {
    self.skipWaiting(); // 強制讓新的 Service Worker 立即進入 active 狀態
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then(cache => cache.addAll(urlsToCache))
    );
});

self.addEventListener('activate', event => {
    // 清除舊版本的快取，確保使用者取得最新檔案
    event.waitUntil(
        caches.keys().then(cacheNames => {
            return Promise.all(
                cacheNames.map(cacheName => {
                    if (cacheName !== CACHE_NAME) {
                        return caches.delete(cacheName);
                    }
                })
            );
        }).then(() => self.clients.claim()) // 立即接管頁面控制權
    );
});

self.addEventListener('fetch', event => {
    // Stale-While-Revalidate 策略
    event.respondWith(
        caches.match(event.request)
            .then(cachedResponse => {
                // 1. 無論有沒有快取，都發起網路請求去抓最新版 (Revalidate)
                const fetchPromise = fetch(event.request).then(networkResponse => {
                    // 將最新的資源存入快取
                    const responseToCache = networkResponse.clone();
                    caches.open(CACHE_NAME).then(cache => cache.put(event.request, responseToCache));
                    return networkResponse;
                });

                // 2. 如果有快取就先回傳快取 (Stale)，沒有才回傳網路請求的結果
                return cachedResponse || fetchPromise;
            })
    );
});