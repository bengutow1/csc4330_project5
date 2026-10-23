import { skipToken, useQuery } from "@connectrpc/connect-query";
import { type Combo, ComboService, ComboState } from "@fries-in-the-bag/apigen/combo/v1/combo_pb";

// How often a waiting combo asks again: the lobby and the reveal both see the other phone's scan
// within this long, which is what lets the two phones reveal together.
const WAITING_POLL_MS = 2000;

/**
 * useCombo reads one combo. While it waits for a scan it asks again every 2 s, the one polling
 * this app does (ts-api.md); once both fridges are in, or the combo ends, it stops.
 */
export function useCombo(comboId: bigint | undefined) {
  return useQuery(ComboService.method.getCombo, comboInput(comboId), {
    refetchInterval: (query) => (comboWaiting(query.state.data?.combo) ? WAITING_POLL_MS : false),
  });
}

/** comboInput is what useCombo sends: skipToken until the id is a real one. */
export function comboInput(comboId: bigint | undefined) {
  return comboId === undefined || comboId <= 0n ? skipToken : { comboId };
}

/**
 * comboWaiting says whether a combo is still waiting on a scan: open, with no combined verdict
 * yet. Nothing has loaded counts as not waiting, so a read that failed is not polled.
 */
export function comboWaiting(combo: Combo | undefined): boolean {
  return combo !== undefined && combo.state === ComboState.OPEN && combo.verdict === undefined;
}
