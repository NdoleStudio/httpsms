export default class Bag<T> {
  private items = new Map<string, Array<T>>()

  serialize(): Record<string, Array<T>> {
    const result: Record<string, Array<T>> = {}
    this.items.forEach((value: T[], key) => {
      result[key] = value
    })
    return result
  }

  static fromObject<T>(items: Record<string, Array<T>>): Bag<T> {
    const result = new Bag<T>()
    Object.keys(items).forEach((key) => {
      result.addMany(key, items[key])
    })
    return result
  }

  add(key: string, value: T): this {
    let messages: Array<T> | undefined = this.items.get(key)
    if (messages === undefined) {
      messages = []
    }

    if (!messages.includes(value)) {
      messages.push(value)
    }

    this.items.set(key, messages)
    return this
  }

  addMany(key: string, values: Array<T>): this {
    values.forEach((value: T) => {
      this.add(key, value)
    })
    return this
  }

  has(key: string): boolean {
    return this.items.has(key)
  }

  first(key: string): T | undefined {
    if (this.has(key)) {
      return this.get(key)[0] ?? undefined
    }
    return undefined
  }

  get(key: string): Array<T> {
    const result = this.items.get(key)
    if (result === undefined) {
      return []
    }
    return result
  }

  size(): number {
    return this.items.size
  }
}
