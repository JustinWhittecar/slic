import { useState, useEffect, useRef } from 'react'
import { track } from '../analytics'
import { useAuth } from '../contexts/AuthContext'

interface FeedbackModalProps {
  onClose: () => void
}

const TYPES = [
  { value: 'bug', label: '🐛 Bug' },
  { value: 'feature', label: '💡 Feature' },
  { value: 'general', label: '💬 General' },
]

export function FeedbackModal({ onClose }: FeedbackModalProps) {
  const { user } = useAuth()
  const [type, setType] = useState('general')
  const [message, setMessage] = useState('')
  const [contact, setContact] = useState('')
  const [status, setStatus] = useState<'idle' | 'sending' | 'sent' | 'error'>('idle')
  const [visible, setVisible] = useState(false)
  const panelRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    requestAnimationFrame(() => setVisible(true))
    textareaRef.current?.focus()
  }, [])

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) onClose()
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [onClose])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  const submit = async () => {
    if (!message.trim()) return
    setStatus('sending')
    try {
      const res = await fetch('/api/feedback', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type, message: message.trim(), contact: user?.email || contact.trim() }),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text)
      }
      track('feedback_submit', { type })
      setStatus('sent')
    } catch {
      setStatus('error')
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: 'rgba(0,0,0,0.4)' }}>
      <div
        ref={panelRef}
        className="w-full max-w-md rounded-lg shadow-2xl transition-all duration-200"
        style={{
          background: 'var(--bg-page)',
          border: '1px solid var(--border-default)',
          transform: visible ? 'scale(1) translateY(0)' : 'scale(0.95) translateY(10px)',
          opacity: visible ? 1 : 0,
        }}
      >
        {status === 'sent' ? (
          <div className="p-6 text-center">
            <div className="text-3xl mb-3">✓</div>
            <h3 className="text-sm font-semibold mb-1" style={{ color: 'var(--text-primary)' }}>Thanks for the feedback!</h3>
            <p className="text-xs" style={{ color: 'var(--text-secondary)' }}>We'll take a look.</p>
            <button
              onClick={onClose}
              className="mt-4 text-xs px-4 py-2 rounded cursor-pointer"
              style={{ background: 'var(--accent)', color: '#fff' }}
            >
              Close
            </button>
          </div>
        ) : (
          <>
            <div className="p-4 pb-0 flex justify-between items-center">
              <h3 className="text-sm font-bold" style={{ color: 'var(--text-primary)' }}>Send Feedback</h3>
              <button
                onClick={onClose}
                className="text-lg cursor-pointer min-w-[44px] min-h-[44px] flex items-center justify-center"
                style={{ color: 'var(--text-tertiary)' }}
              >
                ✕
              </button>
            </div>

            <div className="p-4 space-y-3">
              {/* Type selector */}
              <div className="flex gap-1.5">
                {TYPES.map(t => (
                  <button
                    key={t.value}
                    onClick={() => setType(t.value)}
                    className="px-3 py-1.5 text-xs rounded cursor-pointer transition-colors"
                    style={{
                      background: type === t.value ? 'var(--accent)' : 'transparent',
                      color: type === t.value ? '#fff' : 'var(--text-secondary)',
                      border: `1px solid ${type === t.value ? 'var(--accent)' : 'var(--border-default)'}`,
                    }}
                  >
                    {t.label}
                  </button>
                ))}
              </div>

              {/* Message */}
              <textarea
                ref={textareaRef}
                value={message}
                onChange={e => setMessage(e.target.value)}
                placeholder="What's on your mind?"
                rows={4}
                maxLength={5000}
                className="w-full px-3 py-2 rounded text-sm outline-none resize-none"
                style={{
                  background: 'var(--bg-surface)',
                  border: '1px solid var(--border-default)',
                  color: 'var(--text-primary)',
                }}
              />

              {/* Contact (optional, hidden when logged in) */}
              {!user && (
                <input
                  type="text"
                  value={contact}
                  onChange={e => setContact(e.target.value)}
                  placeholder="Email or name (optional, for follow-up)"
                  className="w-full px-3 py-2 rounded text-xs outline-none"
                  style={{
                    background: 'var(--bg-surface)',
                    border: '1px solid var(--border-default)',
                    color: 'var(--text-primary)',
                  }}
                />
              )}

              {status === 'error' && (
                <p className="text-xs" style={{ color: '#ef4444' }}>
                  Something went wrong. Please try again.
                </p>
              )}

              {/* Submit */}
              <div className="flex justify-end">
                <button
                  onClick={submit}
                  disabled={!message.trim() || status === 'sending'}
                  className="text-xs px-4 py-2 rounded cursor-pointer font-medium disabled:opacity-50"
                  style={{ background: 'var(--accent)', color: '#fff' }}
                >
                  {status === 'sending' ? 'Sending…' : 'Submit'}
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
