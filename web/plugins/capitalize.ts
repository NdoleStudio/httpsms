export default function (value: string | null) {
  if (!value) {
    return ''
  }

  value = value.toString()

  return value.charAt(0).toUpperCase() + value.slice(1)
}
