import { createConnectQueryKey, useInfiniteQuery, useQuery } from "@connectrpc/connect-query";
import {
  type Board,
  BoardService,
  type GetBoardResponse,
  type Range,
} from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { keepPreviousData, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

// GetFridgeOfTheWeek takes no input. A module constant keeps the query key the same on every render.
const FRIDGE_INPUT = {};

/**
 * useBoard reads one board a page at a time, ranked by the server, with the caller's own row on
 * every page. With a crewId it ranks only that crew's members, among themselves; 0n is the whole
 * campus. Switching board, range or crew keeps the last rows on screen, marked
 * isPlaceholderData, until the new ones land.
 */
export function useBoard(board: Board, range: Range, crewId = 0n) {
  return useInfiniteQuery(
    BoardService.method.getBoard,
    { board, range, crewId, cursor: "" },
    {
      pageParamKey: "cursor",
      getNextPageParam: nextBoardCursor,
      placeholderData: keepPreviousData,
    },
  );
}

/** nextBoardCursor is the cursor for the page after this one; undefined ends the board. */
export function nextBoardCursor(page: GetBoardResponse) {
  return page.nextCursor === "" ? undefined : page.nextCursor;
}

/** useFridgeOfTheWeek reads the week's best fridge. Before anyone scans this week it is unset, which is an answer. */
export function useFridgeOfTheWeek() {
  return useQuery(BoardService.method.getFridgeOfTheWeek, FRIDGE_INPUT);
}

/** useInvalidateBoard stales every board query, so the board's next look asks the server again. */
export function useInvalidateBoard() {
  const queryClient = useQueryClient();
  return useCallback(
    () =>
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({ schema: BoardService, cardinality: undefined }),
      }),
    [queryClient],
  );
}
