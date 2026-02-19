export interface BlogPost {
  title: string
  slug: string
  date: string
  description: string
  tags: string[]
  author: string
  image?: string
  content: string
}

function parseFrontmatter(raw: string): BlogPost {
  const match = raw.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/)
  if (!match) throw new Error('Invalid frontmatter')
  const [, meta, content] = match
  const obj: Record<string, string> = {}
  for (const line of meta.split('\n')) {
    const m = line.match(/^(\w+):\s*(.+)$/)
    if (m) obj[m[1]] = m[2].replace(/^["']|["']$/g, '')
  }
  const tags = obj.tags
    ? JSON.parse(obj.tags.replace(/'/g, '"'))
    : []
  return {
    title: obj.title || '',
    slug: obj.slug || '',
    date: obj.date || '',
    description: obj.description || '',
    tags,
    author: obj.author || '',
    image: obj.image || undefined,
    content: content.trim(),
  }
}

// Vite glob import of all .md files
const modules = import.meta.glob('./posts/*.md', { query: '?raw', eager: true, import: 'default' }) as Record<string, string>

export const posts: BlogPost[] = Object.values(modules)
  .map(raw => parseFrontmatter(raw))
  .sort((a, b) => b.date.localeCompare(a.date))

export function getPostBySlug(slug: string): BlogPost | undefined {
  return posts.find(p => p.slug === slug)
}
