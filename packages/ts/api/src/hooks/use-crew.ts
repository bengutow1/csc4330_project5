import type { DescService } from "@bufbuild/protobuf";
import { createConnectQueryKey, useMutation, useQuery } from "@connectrpc/connect-query";
import { BoardService } from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { CrewService } from "@fries-in-the-bag/apigen/crew/v1/crew_pb";
import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

// ListMyCrews takes no input. A module constant keeps the query key the same on every render.
const LIST_INPUT = {};

// The services whose answers change when a player makes, joins or leaves a crew: their list of
// crews, and the boards, whose crew scopes rank its members.
const CREW_READERS: readonly DescService[] = [CrewService, BoardService];

/** useMyCrews reads the caller's crews, the one they joined first first. It is one short list: a player is in at most 10. */
export function useMyCrews() {
  return useQuery(CrewService.method.listMyCrews, LIST_INPUT);
}

/** useCreateCrew makes a crew with the caller in it, and a code to share. */
export function useCreateCrew() {
  const invalidateCrews = useInvalidateCrews();
  return useMutation(CrewService.method.createCrew, { onSettled: () => invalidateCrews() });
}

/** useJoinCrew joins the crew a code names. A wrong code is not found, in words the screen shows. */
export function useJoinCrew() {
  const invalidateCrews = useInvalidateCrews();
  return useMutation(CrewService.method.joinCrew, { onSettled: () => invalidateCrews() });
}

/** useLeaveCrew takes the caller out of a crew. */
export function useLeaveCrew() {
  const invalidateCrews = useInvalidateCrews();
  return useMutation(CrewService.method.leaveCrew, { onSettled: () => invalidateCrews() });
}

/** useInvalidateCrews stales the crew list and the boards, so both ask the server again. */
export function useInvalidateCrews() {
  const queryClient = useQueryClient();
  return useCallback(() => invalidateCrews(queryClient), [queryClient]);
}

/**
 * invalidateCrews stales every query a crew change moves, without waiting for them, so a write
 * never waits on a board page refetching behind it.
 */
export function invalidateCrews(queryClient: QueryClient) {
  for (const schema of CREW_READERS) {
    void queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema, cardinality: undefined }),
    });
  }
}
