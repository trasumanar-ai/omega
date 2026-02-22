export class Rng {
  private state: number

  constructor(seed = Date.now()) {
    this.state = seed >>> 0
  }

  next(): number {
    this.state = (1664525 * this.state + 1013904223) >>> 0
    return this.state / 0x100000000
  }

  int(min: number, maxExclusive: number): number {
    const span = maxExclusive - min
    return min + Math.floor(this.next() * span)
  }

  shuffle<T>(arr: T[]): void {
    for (let i = arr.length - 1; i > 0; i -= 1) {
      const j = this.int(0, i + 1)
      ;[arr[i], arr[j]] = [arr[j], arr[i]]
    }
  }
}
