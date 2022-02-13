package core

type Host struct {
	// the host id: ip:port
	Name string

	// the load bound of the host
	LoadBound uint64
}
