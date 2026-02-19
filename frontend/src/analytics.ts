const FLUSH_INTERVAL = 5000
const BATCH_SIZE = 10

interface QueuedEvent {
  name: string
  properties?: Record<string, any>
  page_url: string
  referrer: string
}

let queue: QueuedEvent[] = []
let timer: ReturnType<typeof setTimeout> | null = null

function getSessionId(): string {
  let sid = localStorage.getItem('slic_sid')
  if (!sid) {
    sid = crypto.randomUUID()
    localStorage.setItem('slic_sid', sid)
  }
  return sid
}

function flush() {
  if (queue.length === 0) return
  const batch = queue.splice(0, 50)
  const body = JSON.stringify({ events: batch })

  // Use sendBeacon if available (works during unload), fall back to fetch
  if (navigator.sendBeacon) {
    navigator.sendBeacon('/api/events', new Blob([body], { type: 'application/json' }))
  } else {
    fetch('/api/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
      credentials: 'include',
      keepalive: true,
    }).catch(() => {})
  }
}

function scheduleFlush() {
  if (timer) return
  timer = setTimeout(() => {
    timer = null
    flush()
  }, FLUSH_INTERVAL)
}

export function track(eventName: string, properties?: Record<string, any>) {
  queue.push({
    name: eventName,
    properties,
    page_url: window.location.href,
    referrer: document.referrer,
  })

  if (queue.length >= BATCH_SIZE) {
    if (timer) { clearTimeout(timer); timer = null }
    flush()
  } else {
    scheduleFlush()
  }
}

export function trackPageView(page: string) {
  track('page_view', { page })
}

// Flush on page unload
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => flush())
  // Also ensure session ID cookie exists for server-side correlation
  getSessionId()
}
