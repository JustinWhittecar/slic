import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { getPostBySlug } from './posts'

interface Props {
  slug: string
  onBack: () => void
  onBackToIndex: () => void
}

export function BlogPost({ slug, onBack, onBackToIndex }: Props) {
  const post = getPostBySlug(slug)

  if (!post) {
    return (
      <div className="min-h-screen flex items-center justify-center" style={{ background: 'var(--bg-page)', color: 'var(--text-primary)' }}>
        <div className="text-center">
          <h1 className="text-xl font-bold mb-2">Post not found</h1>
          <button onClick={onBackToIndex} className="text-sm cursor-pointer" style={{ color: 'var(--accent)' }}>← Back to blog</button>
        </div>
      </div>
    )
  }

  // Set OG meta tags
  if (typeof document !== 'undefined') {
    document.title = `${post.title} — SLIC Blog`
    const setMeta = (property: string, content: string) => {
      let el = document.querySelector(`meta[property="${property}"]`) as HTMLMetaElement | null
      if (!el) {
        el = document.createElement('meta')
        el.setAttribute('property', property)
        document.head.appendChild(el)
      }
      el.setAttribute('content', content)
    }
    const setMetaName = (name: string, content: string) => {
      let el = document.querySelector(`meta[name="${name}"]`) as HTMLMetaElement | null
      if (!el) {
        el = document.createElement('meta')
        el.setAttribute('name', name)
        document.head.appendChild(el)
      }
      el.setAttribute('content', content)
    }
    setMeta('og:title', post.title)
    setMeta('og:description', post.description)
    setMeta('og:type', 'article')
    setMeta('og:url', `https://slic.dev/blog/${post.slug}`)
    if (post.image) setMeta('og:image', post.image)
    setMetaName('description', post.description)
    setMeta('twitter:card', 'summary_large_image')
    setMeta('twitter:title', post.title)
    setMeta('twitter:description', post.description)
  }

  return (
    <div className="min-h-screen" style={{ background: 'var(--bg-page)', color: 'var(--text-primary)' }}>
      <div className="max-w-3xl mx-auto px-4 py-8">
        <div className="flex gap-3 mb-6">
          <button onClick={onBack} className="text-xs cursor-pointer hover:underline" style={{ color: 'var(--accent)' }}>← SLIC</button>
          <button onClick={onBackToIndex} className="text-xs cursor-pointer hover:underline" style={{ color: 'var(--accent)' }}>← Blog</button>
        </div>

        <article>
          <header className="mb-8">
            <h1 className="text-2xl font-bold mb-3 leading-tight">{post.title}</h1>
            <div className="flex items-center gap-3">
              <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>{post.author}</span>
              <span style={{ color: 'var(--text-tertiary)' }}>·</span>
              <time className="text-sm" style={{ color: 'var(--text-tertiary)' }}>
                {new Date(post.date + 'T00:00:00').toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })}
              </time>
            </div>
            <div className="flex gap-2 mt-3">
              {post.tags.map(tag => (
                <span key={tag} className="text-xs px-2 py-0.5 rounded" style={{ background: 'var(--bg-elevated)', color: 'var(--text-tertiary)' }}>
                  {tag}
                </span>
              ))}
            </div>
          </header>

          <div className="blog-content">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {post.content}
            </ReactMarkdown>
          </div>
        </article>
      </div>
    </div>
  )
}
