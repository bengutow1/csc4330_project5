// Author: Caden Lund
package board

import (
	boardv1 "github.com/bengutow1/csc4330_project5/packages/go/apigen/board/v1"
	"github.com/bengutow1/csc4330_project5/packages/go/repository"
)

// toProtoBoard maps a page onto the wire. rows may hold one row past the page, which only says
// there is more.
func toProtoBoard(in getBoardInput, rows []repository.BoardRow, mine repository.BoardRow, ranked bool) *boardv1.GetBoardResponse {
	out := &boardv1.GetBoardResponse{}
	if len(rows) > in.limit {
		last := rows[in.limit-1]
		out.NextCursor = encodeCursor(repository.BoardKey{Value: last.Value, Handle: last.Handle})
		rows = rows[:in.limit]
	}
	out.Rows = make([]*boardv1.Row, 0, len(rows))
	for _, r := range rows {
		out.Rows = append(out.Rows, toProtoRow(in.board, r))
	}
	if ranked {
		out.MyRow = toProtoRow(in.board, mine)
	}
	return out
}

// toProtoRow maps one row. The Hall of Fries leader, every rank-1 row there, wears the crown.
func toProtoRow(board boardv1.Board, r repository.BoardRow) *boardv1.Row {
	return &boardv1.Row{
		// A rank is at most the number of players, well inside int32.
		Rank:   int32(r.Rank),
		Handle: r.Handle,
		Avatar: r.Avatar,
		Value:  r.Value,
		Crown:  board == boardv1.Board_BOARD_HALL_OF_FRIES && r.Rank == 1,
	}
}
