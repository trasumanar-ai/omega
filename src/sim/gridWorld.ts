import type { Coord } from './types'

export class GridWorld {
  readonly width: number
  readonly height: number
  readonly occupancy: Int32Array
  private readonly positions: Coord[]

  constructor(width: number, height: number, agentCount: number) {
    this.width = width
    this.height = height
    this.occupancy = new Int32Array(width * height)
    this.occupancy.fill(-1)
    this.positions = Array.from({ length: agentCount }, () => ({ x: -1, y: -1 }))
  }

  getPosition(agentId: number): Coord {
    return this.positions[agentId]
  }

  place(agentId: number, coord: Coord): void {
    const idx = this.index(coord)
    if (this.occupancy[idx] !== -1) {
      throw new Error(`Cell is occupied at ${coord.x},${coord.y}`)
    }
    this.occupancy[idx] = agentId
    this.positions[agentId] = coord
  }

  isEmpty(coord: Coord): boolean {
    return this.occupancy[this.index(coord)] === -1
  }

  move(agentId: number, target: Coord): boolean {
    if (!this.inBounds(target)) {
      return false
    }
    const targetIndex = this.index(target)
    if (this.occupancy[targetIndex] !== -1) {
      return false
    }

    const current = this.positions[agentId]
    this.occupancy[this.index(current)] = -1
    this.occupancy[targetIndex] = agentId
    this.positions[agentId] = target
    return true
  }

  neighbors4(coord: Coord): Coord[] {
    const candidates = [
      { x: coord.x + 1, y: coord.y },
      { x: coord.x - 1, y: coord.y },
      { x: coord.x, y: coord.y + 1 },
      { x: coord.x, y: coord.y - 1 },
    ]
    return candidates.filter((c) => this.inBounds(c))
  }

  inBounds(coord: Coord): boolean {
    return coord.x >= 0 && coord.x < this.width && coord.y >= 0 && coord.y < this.height
  }

  private index(coord: Coord): number {
    return coord.y * this.width + coord.x
  }
}
