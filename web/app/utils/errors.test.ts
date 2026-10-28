import assert from 'node:assert/strict'
import { test } from 'node:test'
import { getErrorMessages } from './errors'

const validationData = {
  messages_per_minute: [
    'The messages_per_minute field value can not be greater than 30',
  ],
}

test('formats fetch validation errors while retaining field keys', () => {
  const errors = getErrorMessages({
    status: 422,
    data: { data: validationData },
    response: new Response(null, { status: 422 }),
  })

  assert.deepEqual(errors.get('messages_per_minute'), [
    'The messages per minute value can not be greater than 30',
  ])
})

test('still formats Axios validation errors', () => {
  const errors = getErrorMessages({
    response: { status: 422, data: { data: validationData } },
  })

  assert.deepEqual(errors.get('messages_per_minute'), [
    'The messages per minute value can not be greater than 30',
  ])
})

test('does not turn non-validation failures into field errors', () => {
  for (const error of [
    null,
    undefined,
    new Error('Network error'),
    { status: 500, data: { data: validationData } },
    { status: 422, data: { message: 'Validation failed' } },
  ]) {
    assert.equal(getErrorMessages(error).size(), 0)
  }
})
