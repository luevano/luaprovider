package http

// SetMaxBodySize sets maxBodySize until the returned func is called.
func SetMaxBodySize(n int64) (restore func()) {
	old := maxBodySize
	maxBodySize = n
	return func() { maxBodySize = old }
}
