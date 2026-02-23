package sim

import "fmt"

type GridWorld struct {
	Width     int
	Height    int
	Occupancy []int
	positions []Coord
}

func NewGridWorld(width, height, capacity int) *GridWorld {
	occupancy := make([]int, width*height)
	for i := range occupancy {
		occupancy[i] = -1
	}
	positions := make([]Coord, capacity)
	for i := range positions {
		positions[i] = Coord{X: -1, Y: -1}
	}
	return &GridWorld{
		Width:     width,
		Height:    height,
		Occupancy: occupancy,
		positions: positions,
	}
}

func (g *GridWorld) GetPosition(agentID int) Coord {
	if agentID < 0 || agentID >= len(g.positions) {
		return Coord{X: -1, Y: -1}
	}
	return g.positions[agentID]
}

func (g *GridWorld) Place(agentID int, coord Coord) error {
	if !g.InBounds(coord) {
		return fmt.Errorf("place out of bounds: %d,%d", coord.X, coord.Y)
	}
	idx := g.index(coord)
	if g.Occupancy[idx] != -1 {
		return fmt.Errorf("cell is occupied at %d,%d", coord.X, coord.Y)
	}
	g.Occupancy[idx] = agentID
	g.positions[agentID] = coord
	return nil
}

func (g *GridWorld) IsEmpty(coord Coord) bool {
	if !g.InBounds(coord) {
		return false
	}
	return g.Occupancy[g.index(coord)] == -1
}

func (g *GridWorld) Move(agentID int, target Coord) bool {
	if !g.InBounds(target) {
		return false
	}
	targetIndex := g.index(target)
	if g.Occupancy[targetIndex] != -1 {
		return false
	}
	current := g.positions[agentID]
	if !g.InBounds(current) {
		return false
	}
	g.Occupancy[g.index(current)] = -1
	g.Occupancy[targetIndex] = agentID
	g.positions[agentID] = target
	return true
}

func (g *GridWorld) Remove(agentID int) {
	if agentID < 0 || agentID >= len(g.positions) {
		return
	}
	current := g.positions[agentID]
	if !g.InBounds(current) {
		g.positions[agentID] = Coord{X: -1, Y: -1}
		return
	}
	g.Occupancy[g.index(current)] = -1
	g.positions[agentID] = Coord{X: -1, Y: -1}
}

func (g *GridWorld) Neighbors4(coord Coord) []Coord {
	candidates := [...]Coord{
		{X: coord.X + 1, Y: coord.Y},
		{X: coord.X - 1, Y: coord.Y},
		{X: coord.X, Y: coord.Y + 1},
		{X: coord.X, Y: coord.Y - 1},
	}
	out := make([]Coord, 0, 4)
	for _, c := range candidates {
		if g.InBounds(c) {
			out = append(out, c)
		}
	}
	return out
}

func (g *GridWorld) InBounds(coord Coord) bool {
	return coord.X >= 0 && coord.X < g.Width && coord.Y >= 0 && coord.Y < g.Height
}

func (g *GridWorld) index(coord Coord) int {
	return coord.Y*g.Width + coord.X
}
