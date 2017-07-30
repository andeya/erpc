package utils

// Counter count size
type Counter struct {
	count int
}

// Write count size of writed.
func (c *Counter) Write(b []byte) (int, error) {
	cnt := len(b)
	c.count += cnt
	return cnt, nil
}

// Count returns count.
func (c *Counter) Count() int {
	return c.count
}

// Reset clear count.
func (c *Counter) Reset() {
	c.count = 0
}
