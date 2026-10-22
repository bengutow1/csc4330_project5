import { createConnectQueryKey, skipToken } from "@connectrpc/connect-query";
import { AccountService } from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { BoardService } from "@fries-in-the-bag/apigen/board/v1/board_pb";
import { ScanService } from "@fries-in-the-bag/apigen/scan/v1/scan_pb";
import { partialMatchKey } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { createApiTransport } from "../transport.ts";
import { SCANNED, SCANS, scanInput } from "./use-scan.ts";

describe("scanInput", () => {
  it("waits until there is a scan to ask about", () => {
    expect(scanInput(undefined)).toBe(skipToken);
    expect(scanInput(0n)).toBe(skipToken);
  });

  it("asks for a real scan by its id", () => {
    expect(scanInput(42n)).toEqual({ scanId: 42n });
  });
});

describe("SCANS", () => {
  // Built as the hooks build them, through a real transport, so a key the family missed would
  // show here rather than as a verdict that never refreshes.
  const transport = createApiTransport({ baseUrl: "http://api.test.invalid" });
  const keyOf = (key: ReturnType<typeof createConnectQueryKey>) => partialMatchKey(key, SCANS);

  it("covers every scan read, so one invalidation refreshes them all", () => {
    const oneScan = createConnectQueryKey({
      schema: ScanService.method.getScan,
      input: { scanId: 42n },
      transport,
      cardinality: "finite",
    });
    const lastScan = createConnectQueryKey({
      schema: ScanService.method.getLastScan,
      input: {},
      transport,
      cardinality: "finite",
    });
    expect(keyOf(oneScan)).toBe(true);
    expect(keyOf(lastScan)).toBe(true);
  });

  it("leaves other services' reads alone", () => {
    const whoAmI = createConnectQueryKey({
      schema: AccountService.method.whoAmI,
      input: {},
      transport,
      cardinality: "finite",
    });
    expect(keyOf(whoAmI)).toBe(false);
  });
});

describe("SCANNED", () => {
  const transport = createApiTransport({ baseUrl: "http://api.test.invalid" });

  it("covers the scans and the board, whose fridge of the week a new scan can take", () => {
    const lastScan = createConnectQueryKey({
      schema: ScanService.method.getLastScan,
      input: {},
      transport,
      cardinality: "finite",
    });
    const fridge = createConnectQueryKey({
      schema: BoardService.method.getFridgeOfTheWeek,
      input: {},
      transport,
      cardinality: "finite",
    });
    expect(SCANNED.some((family) => partialMatchKey(lastScan, family))).toBe(true);
    expect(SCANNED.some((family) => partialMatchKey(fridge, family))).toBe(true);
  });

  it("leaves the account alone, which a scan does not change", () => {
    const whoAmI = createConnectQueryKey({
      schema: AccountService.method.whoAmI,
      input: {},
      transport,
      cardinality: "finite",
    });
    expect(SCANNED.some((family) => partialMatchKey(whoAmI, family))).toBe(false);
  });
});
