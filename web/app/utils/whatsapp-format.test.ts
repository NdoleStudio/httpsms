import assert from 'node:assert/strict'
import { test } from 'node:test'

import { formatWhatsappText, isUrlNotificationToken } from './whatsapp-format'

test('isUrlNotificationToken', async (t) => {
  await t.test('returns true for an https:// token', () => {
    assert.equal(isUrlNotificationToken('https://example.com/notify'), true)
  })

  await t.test('returns false for an FCM token', () => {
    assert.equal(isUrlNotificationToken('some-fcm-token'), false)
  })

  await t.test('returns false for http:// (non-https) tokens', () => {
    assert.equal(isUrlNotificationToken('http://example.com/notify'), false)
  })

  await t.test('returns false for null/undefined/empty tokens', () => {
    assert.equal(isUrlNotificationToken(undefined), false)
    assert.equal(isUrlNotificationToken(null), false)
    assert.equal(isUrlNotificationToken(''), false)
  })

  await t.test('trims whitespace and is case-insensitive', () => {
    assert.equal(isUrlNotificationToken('  HTTPS://example.com  '), true)
  })
})

test('formatWhatsappText - boundaries', async (t) => {
  await t.test('bolds text surrounded by whitespace', () => {
    assert.equal(
      formatWhatsappText('*boldme* test'),
      '<strong>boldme</strong> test',
    )
  })

  await t.test('does not bold a marker with no boundary at the end', () => {
    assert.equal(
      formatWhatsappText('*donotboldme*no space oe end of string'),
      '*donotboldme*no space oe end of string',
    )
  })

  await t.test('bolds text bounded by punctuation', () => {
    assert.equal(
      formatWhatsappText('your *IUC/Smartcard number*.'),
      'your <strong>IUC/Smartcard number</strong>.',
    )
  })

  await t.test('does not format a marker embedded mid-word', () => {
    assert.equal(
      formatWhatsappText('(parenthesis*bold*test)'),
      '(parenthesis*bold*test)',
    )
  })

  await t.test('formats italic and strikethrough', () => {
    assert.equal(formatWhatsappText('_it_ and ~s~'), '<em>it</em> and <s>s</s>')
  })
})

test('formatWhatsappText - quotes', async (t) => {
  await t.test('wraps "> " prefixed lines in a quote span', () => {
    assert.equal(
      formatWhatsappText('> quoted text'),
      '<span class="d-block pl-2" style="border-left: 3px solid currentColor; opacity: 0.8;">quoted text</span>',
    )
  })

  await t.test('leaves a bare ">" with no following space untouched', () => {
    assert.equal(formatWhatsappText('>threshold'), '&gt;threshold')
  })
})

test('formatWhatsappText - code', async (t) => {
  await t.test('wraps inline code', () => {
    assert.equal(formatWhatsappText('`code`'), '<code>code</code>')
  })

  await t.test('wraps multiline triple-backtick code blocks', () => {
    assert.equal(
      formatWhatsappText('```\nline one\nline two\n```'),
      '<code>\nline one\nline two\n</code>',
    )
  })

  await t.test('does not reformat markdown characters inside code', () => {
    assert.equal(formatWhatsappText('`a *b* c`'), '<code>a *b* c</code>')
  })
})

test('formatWhatsappText - links', async (t) => {
  await t.test('linkifies a raw URL', () => {
    assert.equal(
      formatWhatsappText('visit https://example.com today'),
      'visit <a href="https://example.com" target="_blank" rel="noopener noreferrer">https://example.com</a> today',
    )
  })

  await t.test('strips unbalanced trailing punctuation from a URL', () => {
    assert.equal(
      formatWhatsappText('Check this out (https://example.com).'),
      'Check this out (<a href="https://example.com" target="_blank" rel="noopener noreferrer">https://example.com</a>).',
    )
  })

  await t.test(
    'keeps a balanced trailing parenthesis as part of the URL',
    () => {
      assert.equal(
        formatWhatsappText('See https://example.com/page_(v2) for details.'),
        'See <a href="https://example.com/page_(v2)" target="_blank" rel="noopener noreferrer">https://example.com/page_(v2)</a> for details.',
      )
    },
  )

  await t.test('bolds a URL wrapped in asterisks', () => {
    assert.equal(
      formatWhatsappText('*https://example.com*'),
      '<strong><a href="https://example.com" target="_blank" rel="noopener noreferrer">https://example.com</a></strong>',
    )
  })
})

test('formatWhatsappText - escaping and placeholders', async (t) => {
  await t.test('escapes HTML special characters', () => {
    assert.equal(
      formatWhatsappText('<script>alert(1)</script>'),
      '&lt;script&gt;alert(1)&lt;/script&gt;',
    )
  })

  await t.test(
    'does not let literal placeholder-like text leak a link into the output',
    () => {
      const input = 'no links here, just text'
      assert.equal(formatWhatsappText(input), input)
    },
  )

  await t.test('two calls do not interfere with each other', () => {
    const first = formatWhatsappText('visit https://a.example today')
    const second = formatWhatsappText('visit https://b.example today')
    assert.ok(first.includes('https://a.example'))
    assert.ok(second.includes('https://b.example'))
    assert.ok(!first.includes('https://b.example'))
  })
})
