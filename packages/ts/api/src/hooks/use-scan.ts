import type { DescService } from "@bufbuild/protobuf";
import { createConnectQueryKey, skipToken, useMutation, useQuery } from "@connectrpc/connect-query";
import { BoardService } from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { CookService } from "@fries-in-the-bag/apigen/cook/v1/cook_pb";
import { ProfileService } from "@fries-in-the-bag/apigen/profile/v1/profile_pb";
import { ScanService } from "@fries-in-the-bag/apigen/scan/v1/scan_pb";
import { type QueryClient, useQueryClient } from "@tanstack/react-query";

// GetLastScan takes no input. A module constant keeps the query key the same on every render.
const LAST_SCAN_INPUT = {};

/** SCANS is every scan read, finite or infinite: the one key the scan family invalidates. */
export const SCANS = createConnectQueryKey({ schema: ScanService, cardinality: undefined });

// The other services whose queries read the player's scans, so a new or fixed scan stales them
// too: the board's Fridge of the Week, the recipes, which the server makes from a scan's list, and
// the profile's best score and badges.
const SCAN_READERS: readonly DescService[] = [BoardService, CookService, ProfileService];
const SCAN_READER_KEYS = SCAN_READERS.map((schema) =>
  createConnectQueryKey({ schema, cardinality: undefined }),
);

/** SCANNED is the family key of every query a scan changes: the scans' own, then each reader's. */
export const SCANNED = [SCANS, ...SCAN_READER_KEYS];

/** useScan reads one of the player's verdicts, and waits until there is a scan to ask about. */
export function useScan(scanId: bigint | undefined) {
  return useQuery(ScanService.method.getScan, scanInput(scanId));
}

/** scanInput is what useScan sends: skipToken until the id is a real one. */
export function scanInput(scanId: bigint | undefined) {
  return scanId === undefined || scanId <= 0n ? skipToken : { scanId };
}

/**
 * useLastScan reads the player's newest verdict. Before their first scan it comes back unset,
 * which is an answer, not an error.
 */
export function useLastScan() {
  return useQuery(ScanService.method.getLastScan, LAST_SCAN_INPUT);
}

/**
 * useCreateScan sends the photos to be judged. A failed scan is never retried on its own: each
 * try spends one of the player's scans, so the player taps again instead.
 */
export function useCreateScan() {
  const invalidateScans = useInvalidateScans();
  return useMutation(ScanService.method.createScan, { onSettled: invalidateScans });
}

/**
 * useEditIngredients saves the player's corrected list. The verdict comes back with the new list
 * and the score it was judged with. The save settles once the verdict is fresh, and the recipes
 * made from the old list refetch behind it: the model writes the new ones, which takes seconds.
 */
export function useEditIngredients() {
  const invalidateScans = useInvalidateScans();
  return useMutation(ScanService.method.editIngredients, { onSettled: invalidateScans });
}

/** useInvalidateScans marks every read a scan changes stale (SCANNED), after a write changed one. */
export function useInvalidateScans() {
  const queryClient = useQueryClient();
  return () => invalidateScanned(queryClient);
}

/**
 * invalidateScanned stales SCANNED, and its promise waits for the scans' own reads alone. The
 * readers refetch behind it, so a verdict never waits on another tab's pages, and a fixed list's
 * save never waits on the model writing its new recipes.
 */
export function invalidateScanned(queryClient: QueryClient) {
  for (const queryKey of SCAN_READER_KEYS) {
    void queryClient.invalidateQueries({ queryKey });
  }
  return queryClient.invalidateQueries({ queryKey: SCANS });
}
