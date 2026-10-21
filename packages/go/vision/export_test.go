// Author: Caden Lund
package vision

// WithoutWaitForTest drops the mock's two-second pause, so a test that reads every fridge does
// not sit through it. Call it before the first read. It compiles into the tests only.
func (m *Mock) WithoutWaitForTest() *Mock {
	m.wait = 0
	return m
}
