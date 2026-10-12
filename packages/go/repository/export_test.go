// Author: Caden Lund
package repository

import "context"

// ExecForTest runs a statement on whichever handle r holds (the pool, or the transaction InTx
// opened), so the external tests can write through InTx before the package has a write method of
// its own. It compiles into the tests only. Once a real write exists, test InTx through it and
// delete this file.
func (r *Repository) ExecForTest(ctx context.Context, sql string, args ...any) error {
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}
