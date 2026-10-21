import { createConnectQueryKey, skipToken, useMutation, useQuery } from "@connectrpc/connect-query";
import { ScanService } from "@fries-in-the-bag/apigen/scan/v1/scan_pb";
import { useQueryClient } from "@tanstack/react-query";

// GetLastScan takes no input. A module constant keeps the query key the same on every render.
const LAST_SCAN_INPUT = {};

/** SCANS is every scan read, finite or infinite: the one key the scan family invalidates. */
export const SCANS = createConnectQueryKey({ schema: ScanService, cardinality: undefined });

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
 * and the score it was judged with.
 */
export function useEditIngredients() {
  const invalidateScans = useInvalidateScans();
  return useMutation(ScanService.method.editIngredients, { onSettled: invalidateScans });
}

/** useInvalidateScans marks every scan read stale, after a write changed one. */
export function useInvalidateScans() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: SCANS });
}
