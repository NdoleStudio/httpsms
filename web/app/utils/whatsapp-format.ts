/**
 * Returns true when the given phone notification token is a URL (HTTP notification
 * transport) instead of an FCM token.
 */
export function isUrlNotificationToken(
  fcmToken?: string | null | undefined,
): boolean {
  if (!fcmToken) {
    return false
  }
  return fcmToken.trim().toLowerCase().startsWith('https://')
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}

// Character class (as a regex source string) that is treated as a valid
// boundary around a formatting marker: whitespace, common punctuation, or
// the start/end of the string.
// Other formatting markers and generated tags (<>) also count so that nested
// formatting such as `_*text*_` works.
const BOUNDARY_CHARS = String.raw`\s.,!?;:'"()\[\]{}<>*_~-`

function withBoundaries(pattern: string): RegExp {
  return new RegExp(
    `(?<=^|[${BOUNDARY_CHARS}])${pattern}(?=$|[${BOUNDARY_CHARS}])`,
    'g',
  )
}

const CODE_BLOCK_PATTERN = withBoundaries('```([^`]+)```')
const INLINE_CODE_PATTERN = withBoundaries('`(\\S(?:[^`\\n]*\\S)?)`')
const BOLD_PATTERN = withBoundaries('\\*(\\S(?:[^*\\n]*\\S)?)\\*')
const ITALIC_PATTERN = withBoundaries('_(\\S(?:[^_\\n]*\\S)?)_')
const STRIKE_PATTERN = withBoundaries('~(\\S(?:[^~\\n]*\\S)?)~')

// Characters that are stripped off the end of a matched URL because they are
// Sentence punctuation that is always treated as trailing text, never part
// of a URL (balanced closing parentheses are handled separately below).
const URL_TRAILING_PUNCTUATION = new Set([
  ',',
  '.',
  ';',
  ':',
  '!',
  '?',
  "'",
  '"',
  ']',
  '}',
])

// WhatsApp formatting-marker characters. A trailing one of these is only
// stripped from a URL when it pairs with a matching opening delimiter
// immediately before the URL (e.g. the closing `*` in `*https://x.com*`),
// so legitimate URL characters like the trailing `_` in `.../file_name_`
// are preserved.
const URL_TRAILING_MARKERS = new Set(['*', '_', '~', '`'])

/**
 * Splits a raw URL match into the actual URL and any trailing characters
 * that aren't part of it. Closing parentheses are balance-checked so URLs
 * such as `https://example.com/page_(v2)` keep their closing parenthesis.
 */
function splitTrailingPunctuation(
  raw: string,
  precedingChar: string | undefined,
): {
  url: string
  trailing: string
} {
  let end = raw.length
  while (end > 0) {
    const char = raw[end - 1] as string
    if (char === ')') {
      const opens = (raw.slice(0, end).match(/\(/g) ?? []).length
      const closes = (raw.slice(0, end).match(/\)/g) ?? []).length
      if (closes <= opens) break
      end--
      continue
    }
    if (!URL_TRAILING_PUNCTUATION.has(char)) break
    end--
  }

  if (end > 0) {
    const char = raw[end - 1] as string
    if (URL_TRAILING_MARKERS.has(char) && precedingChar === char) {
      end--
    }
  }

  return { url: raw.slice(0, end), trailing: raw.slice(end) }
}

/**
 * Stores HTML fragments (links, code spans) behind unique placeholder
 * tokens so that the markdown replacements that run afterwards don't touch
 * them. Each store instance uses a random per-call nonce so that literal
 * text in the message can never collide with a placeholder token.
 */
function createPlaceholderStore() {
  const nonce = Math.random().toString(36).slice(2)
  const open = `\uE000${nonce}:`
  const close = `:${nonce}\uE000`
  const store: string[] = []
  const pattern = new RegExp(`${open}(\\d+)${close}`, 'g')

  return {
    protect(html: string): string {
      store.push(html)
      return `${open}${store.length - 1}${close}`
    },
    restore(text: string): string {
      return text.replace(pattern, (_, idx: string) => store[Number(idx)] ?? '')
    },
  }
}

type PlaceholderStore = ReturnType<typeof createPlaceholderStore>

/**
 * Replaces raw URLs with anchor tags, protecting them behind placeholder
 * tokens so that markdown characters are not misinterpreted as formatting.
 */
function linkifyUrls(text: string, placeholders: PlaceholderStore): string {
  return text.replace(
    /(https?:\/\/[^\s<]+|www\.[^\s<]+)/gi,
    (match, _group, offset: number, full: string) => {
      const precedingChar = offset > 0 ? full[offset - 1] : undefined
      const { url, trailing } = splitTrailingPunctuation(match, precedingChar)
      const href = /^www\./i.test(url) ? `https://${url}` : url
      const anchor = `<a href="${href}" target="_blank" rel="noopener noreferrer">${url}</a>`
      return placeholders.protect(anchor) + trailing
    },
  )
}

/**
 * Replaces code blocks/spans with placeholder tokens so that bold, italic
 * and strikethrough markers inside code are rendered literally instead of
 * being interpreted as formatting.
 */
function extractCodeSpans(
  text: string,
  placeholders: PlaceholderStore,
): string {
  return text
    .replace(CODE_BLOCK_PATTERN, (_, content: string) =>
      placeholders.protect(`<code>${content}</code>`),
    )
    .replace(INLINE_CODE_PATTERN, (_, content: string) =>
      placeholders.protect(`<code>${content}</code>`),
    )
}

/**
 * Converts WhatsApp-style formatted text into sanitized HTML.
 * Supported syntax (each marker pair must be surrounded by whitespace,
 * punctuation, or the start/end of the string, and must not have internal
 * leading/trailing whitespace, matching WhatsApp's own formatting rules):
 *  - *bold*
 *  - _italic_
 *  - ~strikethrough~
 *  - `code` / ```code block``` (code content is never re-formatted)
 *  - "> " quoted lines (the space after ">" is required)
 *  - raw URLs, turned into clickable links
 */
export function formatWhatsappText(text: string): string {
  const escaped = escapeHtml(text ?? '')
  const placeholders = createPlaceholderStore()

  const withLinks = linkifyUrls(escaped, placeholders)

  const withQuotes = withLinks
    .split('\n')
    .map((line) => {
      const quoteMatch = line.match(/^&gt; (.*)$/)
      if (quoteMatch) {
        return `<span class="d-block pl-2" style="border-left: 3px solid currentColor; opacity: 0.8;">${quoteMatch[1]}</span>`
      }
      return line
    })
    .join('\n')

  const withCode = extractCodeSpans(withQuotes, placeholders)

  const formatted = withCode
    .replace(BOLD_PATTERN, '<strong>$1</strong>')
    .replace(ITALIC_PATTERN, '<em>$1</em>')
    .replace(STRIKE_PATTERN, '<s>$1</s>')

  return placeholders.restore(formatted)
}
