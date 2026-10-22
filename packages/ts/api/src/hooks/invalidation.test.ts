import { createConnectQueryKey } from "@connectrpc/connect-query";
import { BoardService } from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { ProfileService } from "@fries-in-the-bag/apigen/profile/v1/profile_pb";
import { MutationObserver, QueryClient, QueryObserver } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { createApiTransport } from "../transport.ts";
import { invalidatePlated } from "./use-cook.ts";
import { invalidateFryRuns } from "./use-fries.ts";
import { invalidateScanned } from "./use-scan.ts";

// How long a write may take to settle once its server call is done. A write that waited on a
// refetch which never lands would still be waiting.
const SETTLE_MS = 50;

/** tabsOnScreen is a client with the board and the Me tab in view, whose next refetches never land. */
function tabsOnScreen() {
  const client = new QueryClient();
  const transport = createApiTransport({ baseUrl: "http://api.test.invalid" });
  const keys = [
    createConnectQueryKey({
      schema: BoardService.method.getFridgeOfTheWeek,
      input: {},
      transport,
      cardinality: "finite",
    }),
    createConnectQueryKey({
      schema: ProfileService.method.getProfile,
      input: {},
      transport,
      cardinality: "finite",
    }),
  ];
  const stops = keys.map((queryKey) =>
    new QueryObserver(client, {
      queryKey,
      queryFn: () => new Promise<never>(() => undefined),
      initialData: {},
      staleTime: Number.POSITIVE_INFINITY,
    }).subscribe(() => undefined),
  );
  return {
    client,
    keys,
    stop: () => {
      for (const stop of stops) {
        stop();
      }
    },
  };
}

describe("a write's invalidation", () => {
  it.each([
    { write: "a scan", invalidate: invalidateScanned },
    { write: "a plate", invalidate: invalidatePlated },
    { write: "a fry run", invalidate: invalidateFryRuns },
  ])(
    "lets $write settle while the board and the Me tab refetch behind it",
    async ({ invalidate }) => {
      const { client, keys, stop } = tabsOnScreen();
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
        // Both tabs were still told: they are refetching.
        for (const queryKey of keys) {
          expect(client.getQueryState(queryKey)?.fetchStatus).toBe("fetching");
        }
      } finally {
        stop();
        client.clear();
      }
    },
  );
});
