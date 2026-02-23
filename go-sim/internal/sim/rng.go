package sim

type RNG struct {
	state uint32
}

func NewRNG(seed uint32) *RNG {
	if seed == 0 {
		seed = 1
	}
	return &RNG{state: seed}
}

func (r *RNG) Next() float64 {
	r.state = 1664525*r.state + 1013904223
	return float64(r.state) / float64(uint64(1)<<32)
}

func (r *RNG) Int(min, maxExclusive int) int {
	if maxExclusive <= min {
		return min
	}
	span := maxExclusive - min
	return min + int(r.Next()*float64(span))
}

func (r *RNG) Shuffle(ids []int) {
	for i := len(ids) - 1; i > 0; i-- {
		j := r.Int(0, i+1)
		ids[i], ids[j] = ids[j], ids[i]
	}
}
