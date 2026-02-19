import { posts } from './posts'

interface Props {
  onNavigate: (slug: string) => void
  onBack: () => void
}

export function BlogIndex({ onNavigate, onBack }: Props) {
  return (
    <div className="min-h-screen" style={{ background: 'var(--bg-page)', color: 'var(--text-primary)' }}>
      <div className="max-w-3xl mx-auto px-4 py-8">
        <button
          onClick={onBack}
          className="text-xs mb-6 cursor-pointer hover:underline"
          style={{ color: 'var(--accent)' }}
        >
          ← Back to SLIC
        </button>
        <h1 className="text-2xl font-bold mb-1">SLIC Blog</h1>
        <p className="text-sm mb-8" style={{ color: 'var(--text-secondary)' }}>
          Data-driven BattleTech analysis and meta discussion.
        </p>
        <div className="flex flex-col gap-6">
          {posts.map(post => (
            <article
              key={post.slug}
              className="rounded-lg p-5 cursor-pointer transition-colors"
              style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-default)' }}
              onClick={() => onNavigate(post.slug)}
              onMouseEnter={e => (e.currentTarget.style.borderColor = 'var(--accent)')}
              onMouseLeave={e => (e.currentTarget.style.borderColor = 'var(--border-default)')}
            >
              <div className="flex items-center gap-2 mb-2">
                <time className="text-xs" style={{ color: 'var(--text-tertiary)' }}>
                  {new Date(post.date + 'T00:00:00').toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })}
                </time>
                <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>·</span>
                <span className="text-xs" style={{ color: 'var(--text-tertiary)' }}>{post.author}</span>
              </div>
              <h2 className="text-lg font-semibold mb-2" style={{ color: 'var(--text-primary)' }}>
                {post.title}
              </h2>
              <p className="text-sm mb-3" style={{ color: 'var(--text-secondary)' }}>
                {post.description}
              </p>
              <div className="flex gap-2">
                {post.tags.map(tag => (
                  <span
                    key={tag}
                    className="text-xs px-2 py-0.5 rounded"
                    style={{ background: 'var(--bg-elevated)', color: 'var(--text-tertiary)' }}
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </article>
          ))}
        </div>
      </div>
    </div>
  )
}
