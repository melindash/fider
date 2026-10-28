package web

// RemoteAddr is the TCP peer address, before any proxy headers. Mage-OS fork.
func (r *Request) RemoteAddr() string {
	return r.instance.RemoteAddr
}
