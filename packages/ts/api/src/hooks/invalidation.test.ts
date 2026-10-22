import { createConnectQueryKey } from "@connectrpc/connect-query";
import { Diet } from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { BoardService } from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { CookService } from "@fries-in-the-bag/apigen/cook/v1/cook_pb";
import { MutationObserver, QueryClient, type QueryKey, QueryObserver } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { createApiTransport } from "../transport.ts";
import { invalidatePlated, recipesInput } from "./use-cook.ts";
import { invalidateFryRuns } from "./use-fries.ts";
import { invalidateScanned } from "./use-scan.ts";

// How long a write may take to settle once its server call is done. A write that waited on a
// refetch which never lands would still be waiting.
const SETTLE_MS = 50;

const transport = createApiTransport({ baseUrl: "http://api.test.invalid" });

// Reads that other services make from what a write changed: the board's fridge of the week, and
// a scan's shelves, whose new recipes after a fixed list come from the model.
const BOARD = createConnectQueryKey({
  schema: BoardService.method.getFridgeOfTheWeek,
  input: {},
  transport,
  cardinality: "finite",
});
const SHELVES = createConnectQueryKey({
  schema: CookService.method.listRecipes,
  input: recipesInput(42n, { maxMinutes: 0, equipment: [], diet: Diet.NONE }),
  transport,
  cardinality: "finite",
});

/** onScreen is a client with one read in view, whose next refetch never lands. */
function onScreen(queryKey: QueryKey) {
  const client = new QueryClient();
  const read = new QueryObserver(client, {
    queryKey,
    queryFn: () => new Promise<never>(() => undefined),
    initialData: {},
    staleTime: Number.POSITIVE_INFINITY,
  });
  return { client, stop: read.subscribe(() => undefined) };
}

describe("a write's invalidation", () => {
  it.each([
    { write: "a scan", read: "the board", queryKey: BOARD, invalidate: invalidateScanned },
    { write: "a plate", read: "the board", queryKey: BOARD, invalidate: invalidatePlated },
    { write: "a fry run", read: "the board", queryKey: BOARD, invalidate: invalidateFryRuns },
    {
      write: "a fixed list",
      read: "the shelves",
      queryKey: SHELVES,
      invalidate: invalidateScanned,
    },
  ])("lets $write settle, with $read refetching behind it", async ({ queryKey, invalidate }) => {
    const { client, stop } = onScreen(queryKey);
    try {
      const write = new MutationObserver(client, {
        mutationFn: () => Promise.resolve("done"),
        onSettled: () => invalidate(client),
      });
      const settled = await Promise.race([
        write.mutate().then(() => "settled"),
        new Promise((resolve) => setTimeout(() => resolve("waiting"), SETTLE_MS)),
      ]);
      expect(settled).toBe("settled");
      // The read was still told: it is refetching.
      expect(client.getQueryState(queryKey)?.fetchStatus).toBe("fetching");
    } finally {
      stop();
      client.clear();
    }
  });
});
