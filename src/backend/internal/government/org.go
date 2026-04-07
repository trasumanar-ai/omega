package government

// Organization is a module that plugs into a government.
// Each organization manages its own domain (banking, identity, contracts, etc.)
type Organization interface {
	Name() string
}
