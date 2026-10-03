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

// Matches the \uE000<index>\uE000 placeholders (private-use Unicode area,
// unlikely to appear in real text) used to protect linkified URLs from the
// markdown replacements that run afterwards.
const LINK_PLACEHOLDER = '\uE000'
const LINK_PLACEHOLDER_PATTERN = /\uE000(\d+)\uE000/g

/**
 * Replaces raw URLs with anchor tags, protecting them behind placeholder
 * tokens so that markdown characters inside the URL (e.g. `_`) are not
 * misinterpreted as formatting by later replacements.
 */
function linkifyUrls(text: string, links: string[]): string {
  return text.replace(/(https?:\/\/[^\s<]+|www\.[^\s<]+)/gi, (match) => {
    const trailing = match.match(/[),.;:!?'"]+$/)?.[0] ?? ''
    const urlText = trailing ? match.slice(0, -trailing.length) : match
    const href = /^www\./i.test(urlText) ? `https://${urlText}` : urlText
    links.push(
      `<a href="${href}" target="_blank" rel="noopener noreferrer">${urlText}</a>${trailing}`,
    )
    return `${LINK_PLACEHOLDER}${links.length - 1}${LINK_PLACEHOLDER}`
  })
}

/**
 * Converts WhatsApp-style formatted text into sanitized HTML.
 * Supported syntax (each marker pair must be surrounded by whitespace,
 * punctuation, or the start/end of the string, and must not have internal
 * leading/trailing whitespace, matching WhatsApp's own formatting rules):
 *  - *bold*
 *  - _italic_
 *  - ~strikethrough~
 *  - `code` / ```code block```
 *  - "> " quoted lines
 *  - raw URLs, turned into clickable links
 */
export function formatWhatsappText(text: string): string {
  const escaped = escapeHtml(text ?? '')

  const links: string[] = []
  const withPlaceholders = linkifyUrls(escaped, links)

  const lines = withPlaceholders.split('\n').map((line) => {
    const quoteMatch = line.match(/^&gt;\s?(.*)$/)
    if (quoteMatch) {
      return `<span class="d-block pl-2" style="border-left: 3px solid currentColor; opacity: 0.8;">${quoteMatch[1]}</span>`
    }
    return line
  })

  const formatted = lines
    .join('\n')
    .replace(
      /(?<=^|[\s.,!?;:'"()[\]{}-])```(\S(?:[^`]*\S)?)```(?=$|[\s.,!?;:'"()[\]{}-])/g,
      '<code>$1</code>',
    )
    .replace(
      /(?<=^|[\s.,!?;:'"()[\]{}-])`(\S(?:[^`\n]*\S)?)`(?=$|[\s.,!?;:'"()[\]{}-])/g,
      '<code>$1</code>',
    )
    .replace(
      /(?<=^|[\s.,!?;:'"()[\]{}-])\*(\S(?:[^*\n]*\S)?)\*(?=$|[\s.,!?;:'"()[\]{}-])/g,
      '<strong>$1</strong>',
    )
    .replace(
      /(?<=^|[\s.,!?;:'"()[\]{}-])_(\S(?:[^_\n]*\S)?)_(?=$|[\s.,!?;:'"()[\]{}-])/g,
      '<em>$1</em>',
    )
    .replace(
      /(?<=^|[\s.,!?;:'"()[\]{}-])~(\S(?:[^~\n]*\S)?)~(?=$|[\s.,!?;:'"()[\]{}-])/g,
      '<s>$1</s>',
    )

  return formatted.replace(
    LINK_PLACEHOLDER_PATTERN,
    (_, idx: string) => links[Number(idx)] ?? '',
  )
}
