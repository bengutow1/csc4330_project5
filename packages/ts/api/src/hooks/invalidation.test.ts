import { createConnectQueryKey } from "@connectrpc/connect-query";
import { BoardService } from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { MutationObserver, QueryClient, QueryObserver } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { createApiTransport } from "../transport.ts";
import { invalidatePlated } from "./use-cook.ts";
import { invalidateFryRuns } from "./use-fries.ts";
import { invalidateScanned } from "./use-scan.ts";

// How long a write may take to settle once its server call is done. A write that waited on a
// refetch which never lands would still be waiting.
const SETTLE_MS = 50;

/** boardOnScreen is a client with the board in view, whose next refetch never lands. */
function boardOnScreen() {
  const client = new QueryClient();
  const queryKey = createConnectQueryKey({
    schema: BoardService.method.getFridgeOfTheWeek,
    input: {},
    transport: createApiTransport({ baseUrl: "http://api.test.invalid" }),
    cardinality: "finite",
  });
  const board = new QueryObserver(client, {
    queryKey,
    queryFn: () => new Promise<never>(() => undefined),
    initialData: {},
    staleTime: Number.POSITIVE_INFINITY,
  });
  return { client, queryKey, stop: board.subscribe(() => undefined) };
}

describe("a write's invalidation", () => {
  it.each([
    { write: "a scan", invalidate: invalidateScanned },
    { write: "a plate", invalidate: invalidatePlated },
    { write: "a fry run", invalidate: invalidateFryRuns },
  ])("lets $write settle while the board refetches behind it", async ({ invalidate }) => {
    const { client, queryKey, stop } = boardOnScreen();
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
      // The board was still told: it is refetching.
      expect(client.getQueryState(queryKey)?.fetchStatus).toBe("fetching");
    } finally {
      stop();
      client.clear();
    }
  });
});
