// Author: Caden Lund
package vision

import "time"

// WithoutWaitForTest drops the mock's two-second pause, so a test that reads every fridge does
// not sit through it. Call it before the first read. It compiles into the tests only.
func (m *Mock) WithoutWaitForTest() *Mock {
	m.wait = 0
	return m
}

// WithTimeoutForTest shortens the client's HTTP timeout, so the timeout test does not sit through
// the real one. It compiles into the tests only.
func (c *Client) WithTimeoutForTest(d time.Duration) *Client {
	c.http.Timeout = d
	return c
}
