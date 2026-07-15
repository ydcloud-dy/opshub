<template>
  <div class="markdown-view" v-html="html"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  content?: string
}>()

const escapeHtml = (value: string) => value
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&#39;')

const escapeAttribute = (value: string) => escapeHtml(value).replace(/`/g, '&#96;')

const isSafeLink = (value: string) => {
  const trimmed = value.trim().toLowerCase()
  return trimmed.startsWith('http://') || trimmed.startsWith('https://') || trimmed.startsWith('mailto:')
}

const inlineMarkdown = (value: string) => {
  const codeTokens: string[] = []
  const withTokens = value.replace(/`([^`]+)`/g, (_, code) => {
    const token = `@@CODE_${codeTokens.length}@@`
    codeTokens.push(`<code>${escapeHtml(code)}</code>`)
    return token
  })
  let html = escapeHtml(withTokens)
    .replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_, label, href) => {
      const rawHref = String(href).replace(/&amp;/g, '&')
      if (!isSafeLink(rawHref)) {
        return label
      }
      return `<a href="${escapeAttribute(rawHref)}" target="_blank" rel="noopener noreferrer">${label}</a>`
    })
    .replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
    .replace(/\*([^*]+)\*/g, '<em>$1</em>')
  codeTokens.forEach((token, index) => {
    html = html.replace(`@@CODE_${index}@@`, token)
  })
  return html
}

const splitTableRow = (line: string) => {
  const normalized = line.trim().replace(/^\|/, '').replace(/\|$/, '')
  return normalized.split('|').map(cell => cell.trim())
}

const isTableSeparator = (line: string) => /^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$/.test(line)

const tableAlignments = (line: string) => splitTableRow(line).map(cell => {
  const value = cell.trim()
  if (value.startsWith(':') && value.endsWith(':')) return 'center'
  if (value.endsWith(':')) return 'right'
  return 'left'
})

const renderTable = (lines: string[]) => {
  const headers = splitTableRow(lines[0])
  const alignments = tableAlignments(lines[1])
  const rows = lines.slice(2).filter(line => line.trim()).map(splitTableRow)
  return `<div class="table-wrap"><table><thead><tr>${headers.map((cell, index) => `<th style="text-align:${alignments[index] || 'left'}">${inlineMarkdown(cell)}</th>`).join('')}</tr></thead><tbody>${rows.map(row => `<tr>${headers.map((_, index) => `<td style="text-align:${alignments[index] || 'left'}">${inlineMarkdown(row[index] || '')}</td>`).join('')}</tr>`).join('')}</tbody></table></div>`
}

const parseOrderedLine = (line: string) => {
  const match = line.trim().match(/^(\d+)\.\s+(.+)$/)
  if (!match) return null
  return {
    number: Number(match[1]) || 1,
    text: match[2]
  }
}

const renderBlock = (block: string, options?: { orderedStart?: number; numberedHeading?: boolean }) => {
  const lines = block.split('\n').filter(line => line.trim())
  if (lines.length >= 2 && lines[0].includes('|') && isTableSeparator(lines[1])) {
    return renderTable(lines)
  }
  if (lines.every(line => /^[-*+]\s+\[[ xX]\]\s+/.test(line.trim()))) {
    return `<ul class="task-list">${lines.map(line => {
      const checked = /\[[xX]\]/.test(line)
      const text = line.trim().replace(/^[-*+]\s+\[[ xX]\]\s+/, '')
      return `<li><input type="checkbox" disabled ${checked ? 'checked' : ''}>${inlineMarkdown(text)}</li>`
    }).join('')}</ul>`
  }
  if (lines.every(line => /^[-*+]\s+/.test(line.trim()))) {
    return `<ul>${lines.map(line => `<li>${inlineMarkdown(line.trim().replace(/^[-*+]\s+/, ''))}</li>`).join('')}</ul>`
  }
  if (lines.every(line => /^\d+\.\s+/.test(line.trim()))) {
    const items = lines.map(line => parseOrderedLine(line)).filter(Boolean) as Array<{ number: number; text: string }>
    const start = options?.orderedStart || items[0]?.number || 1
    if (options?.numberedHeading && items.length === 1) {
      return `<h4 class="numbered-heading"><span class="number-badge">${start}</span><span class="number-title">${inlineMarkdown(items[0].text)}</span></h4>`
    }
    return `<ol start="${start}">${items.map(item => `<li>${inlineMarkdown(item.text)}</li>`).join('')}</ol>`
  }
  if (lines.every(line => /^>\s?/.test(line.trim()))) {
    return `<blockquote>${lines.map(line => inlineMarkdown(line.trim().replace(/^>\s?/, ''))).join('<br>')}</blockquote>`
  }
  if (lines.length === 1) {
    const line = lines[0].trim()
    if (/^---+$/.test(line)) {
      return '<hr>'
    }
    if (/^#{1,4}\s+/.test(line)) {
      const level = Math.min((line.match(/^#+/)?.[0].length || 2), 4)
      return `<h${level}>${inlineMarkdown(line.replace(/^#{1,4}\s+/, ''))}</h${level}>`
    }
  }
  return `<p>${lines.map(inlineMarkdown).join('<br>')}</p>`
}

const isTableStart = (lines: string[], index: number) => {
  const current = lines[index] || ''
  const next = lines[index + 1] || ''
  return current.includes('|') && isTableSeparator(next)
}

const isSpecialBlockStart = (lines: string[], index: number) => {
  const trimmed = (lines[index] || '').trim()
  return !trimmed ||
    isTableStart(lines, index) ||
    /^#{1,4}\s+/.test(trimmed) ||
    /^---+$/.test(trimmed) ||
    /^[-*+]\s+/.test(trimmed) ||
    /^\d+\.\s+/.test(trimmed) ||
    /^>\s?/.test(trimmed)
}

const renderTextSegment = (segment: string) => {
  const lines = segment.replace(/\r\n/g, '\n').split('\n')
  const parts: string[] = []
  let paragraph: string[] = []
  let nextSingleOrderedStart = 1

  const flushParagraph = () => {
    const text = paragraph.map(line => line.trim()).filter(Boolean)
    if (text.length) {
      parts.push(renderBlock(text.join('\n')))
    }
    paragraph = []
  }

  for (let index = 0; index < lines.length;) {
    const line = lines[index]
    const trimmed = line.trim()
    if (!trimmed) {
      flushParagraph()
      index += 1
      continue
    }

    if (isTableStart(lines, index)) {
      flushParagraph()
      const tableLines = [lines[index], lines[index + 1]]
      index += 2
      while (index < lines.length) {
        const row = lines[index]
        const rowTrimmed = row.trim()
        if (!rowTrimmed || !row.includes('|') || isSpecialBlockStart(lines, index)) {
          break
        }
        tableLines.push(row)
        index += 1
      }
      parts.push(renderTable(tableLines))
      continue
    }

    if (/^#{1,4}\s+/.test(trimmed) || /^---+$/.test(trimmed)) {
      flushParagraph()
      parts.push(renderBlock(trimmed))
      nextSingleOrderedStart = 1
      index += 1
      continue
    }

    if (/^[-*+]\s+/.test(trimmed) || /^\d+\.\s+/.test(trimmed) || /^>\s?/.test(trimmed)) {
      flushParagraph()
      const group: string[] = []
      const matcher = /^[-*+]\s+/.test(trimmed)
        ? /^[-*+]\s+/
        : /^\d+\.\s+/.test(trimmed)
          ? /^\d+\.\s+/
          : /^>\s?/
      while (index < lines.length && matcher.test(lines[index].trim())) {
        group.push(lines[index])
        index += 1
      }
      if (/^\d+\.\s+/.test(trimmed)) {
        const parsed = parseOrderedLine(trimmed)
        const sourceStart = parsed?.number || 1
        const orderedStart = sourceStart > 1 ? sourceStart : nextSingleOrderedStart
        parts.push(renderBlock(group.join('\n'), {
          orderedStart,
          numberedHeading: group.length === 1
        }))
        nextSingleOrderedStart = orderedStart + group.length
      } else {
        parts.push(renderBlock(group.join('\n')))
      }
      continue
    }

    paragraph.push(line)
    index += 1
  }

  flushParagraph()
  return parts.join('')
}

const renderMarkdown = (raw?: string) => {
  const text = String(raw || '').trim()
  if (!text) return ''
  const parts: string[] = []
  const codeFence = /```([\w-]*)[ \t]*\n?([\s\S]*?)(```|$)/g
  let lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = codeFence.exec(text)) !== null) {
    const before = text.slice(lastIndex, match.index).trim()
    if (before) {
      parts.push(renderTextSegment(before))
    }
    const lang = match[1] ? `<span>${escapeHtml(match[1])}</span>` : ''
    parts.push(`<pre><div class="code-head">${lang}</div><code>${escapeHtml(match[2].trim())}</code></pre>`)
    lastIndex = match.index + match[0].length
    if (match[3] !== '```') {
      break
    }
  }
  const tail = text.slice(lastIndex).trim()
  if (tail) {
    parts.push(renderTextSegment(tail))
  }
  return parts.join('')
}

const html = computed(() => renderMarkdown(props.content))
</script>

<style scoped>
.markdown-view {
  color: #111827;
  font-size: 14px;
  line-height: 1.75;
  word-break: break-word;
}

.markdown-view :deep(h1),
.markdown-view :deep(h2),
.markdown-view :deep(h3),
.markdown-view :deep(h4) {
  margin: 14px 0 10px;
  color: #0f172a;
  font-weight: 850;
  line-height: 1.35;
}

.markdown-view :deep(h1) {
  font-size: 22px;
}

.markdown-view :deep(h2) {
  font-size: 19px;
}

.markdown-view :deep(h3) {
  font-size: 17px;
}

.markdown-view :deep(h4) {
  font-size: 15px;
}

.markdown-view :deep(.numbered-heading) {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin: 18px 0 8px;
  padding: 0;
  color: #0f172a;
  font-size: 15px;
  font-weight: 850;
  line-height: 1.55;
}

.markdown-view :deep(.number-badge) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  margin-top: 0;
  border-radius: 9px;
  background: #111827;
  color: #fff;
  font-size: 12px;
  font-weight: 900;
  line-height: 1;
  flex: 0 0 auto;
  box-shadow: 0 8px 18px rgba(17, 24, 39, 0.12);
}

.markdown-view :deep(.number-title) {
  min-width: 0;
}

.markdown-view :deep(p) {
  margin: 0 0 12px;
}

.markdown-view :deep(ul),
.markdown-view :deep(ol) {
  margin: 0 0 12px 20px;
  padding: 0;
}

.markdown-view :deep(li) {
  margin: 5px 0;
}

.markdown-view :deep(.task-list) {
  list-style: none;
  margin-left: 0;
}

.markdown-view :deep(.task-list li) {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}

.markdown-view :deep(.task-list input) {
  margin-top: 6px;
}

.markdown-view :deep(blockquote) {
  margin: 0 0 12px;
  padding: 10px 12px;
  border-left: 3px solid #111827;
  background: #f8fafc;
  color: #475569;
  border-radius: 10px;
}

.markdown-view :deep(a) {
  color: #2563eb;
  font-weight: 700;
  text-decoration: none;
}

.markdown-view :deep(a:hover) {
  text-decoration: underline;
}

.markdown-view :deep(hr) {
  margin: 14px 0;
  border: 0;
  border-top: 1px solid #e5e7eb;
}

.markdown-view :deep(.table-wrap) {
  width: 100%;
  overflow: auto;
  margin: 12px 0;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
}

.markdown-view :deep(table) {
  width: 100%;
  border-collapse: collapse;
  min-width: 420px;
}

.markdown-view :deep(th),
.markdown-view :deep(td) {
  padding: 10px 12px;
  border-bottom: 1px solid #e5e7eb;
  text-align: left;
  vertical-align: top;
}

.markdown-view :deep(th) {
  background: #f8fafc;
  color: #475569;
  font-weight: 800;
}

.markdown-view :deep(tr:last-child td) {
  border-bottom: 0;
}

.markdown-view :deep(code) {
  padding: 2px 6px;
  border-radius: 7px;
  background: #eef2f7;
  color: #0f172a;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}

.markdown-view :deep(pre) {
  margin: 12px 0;
  overflow: auto;
  border-radius: 14px;
  background: #0f172a;
  color: #f8d675;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.06);
}

.markdown-view :deep(pre code) {
  display: block;
  padding: 14px 16px 16px;
  background: transparent;
  color: inherit;
  white-space: pre;
  line-height: 1.65;
}

.markdown-view :deep(.code-head) {
  min-height: 8px;
  padding: 8px 12px 0;
  color: #94a3b8;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
</style>
