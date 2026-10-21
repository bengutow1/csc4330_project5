import { create } from "@bufbuild/protobuf";
import { createConnectQueryKey, skipToken } from "@connectrpc/connect-query";
import {
  AccountService,
  CheckHandleResponseSchema,
  WhoAmIResponseSchema,
} from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { MutationObserver, QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { createApiTransport } from "../transport.ts";
import {
  CREATING_ACCOUNT,
  checkHandleInput,
  createAccountOptions,
  WHO_AM_I_FAMILY,
} from "./use-account.ts";

describe("checkHandleInput", () => {
  it("waits until the handle is long enough for the server to take", () => {
    expect(checkHandleInput("")).toBe(skipToken);
    expect(checkHandleInput("ab")).toBe(skipToken);
    expect(checkHandleInput("  ab  ")).toBe(skipToken);
  });

  it("asks from three characters on, sending what was typed for the server to normalize", () => {
    expect(checkHandleInput("abc")).toEqual({ handle: "abc" });
    expect(checkHandleInput(" Chef_Maya ")).toEqual({ handle: " Chef_Maya " });
  });
});

describe("CREATING_ACCOUNT", () => {
  // A screen the player backs out of keeps its CreateAccount running. If the next screen can't
  // see that call, its Next makes a second account and strands the first.
  it("still sees a CreateAccount once the screen that sent it is gone, and no other write", async () => {
    const client = new QueryClient();
    // Made up front: TanStack calls mutationFn a few ticks after mutate.
    let land = () => {};
    const held = new Promise<void>((resolve) => {
      land = resolve;
    });
    const create = new MutationObserver(client, {
      ...createAccountOptions(client),
      mutationFn: () => held,
    });
    const other = new MutationObserver(client, { mutationFn: () => new Promise<void>(() => {}) });
    const leave = create.subscribe(() => {});

    const sent = create.mutate();
    leave();
    void other.mutate();

    expect(client.isMutating(CREATING_ACCOUNT)).toBe(1);
    land();
    await sent;
    expect(client.isMutating(CREATING_ACCOUNT)).toBe(0);
    client.clear();
  });
});

describe("WHO_AM_I_FAMILY", () => {
  // A key that matched nothing would leave a saved kitchen stale on every screen showing it.
  it("reaches the WhoAmI a screen caches, and not the account's other queries", async () => {
    const client = new QueryClient();
    const transport = createApiTransport({ baseUrl: "http://api.test.invalid" });
    const whoAmI = createConnectQueryKey({
      schema: AccountService.method.whoAmI,
      input: {},
      transport,
      cardinality: "finite",
    });
    const checkHandle = createConnectQueryKey({
      schema: AccountService.method.checkHandle,
      input: { handle: "test_maya" },
      transport,
      cardinality: "finite",
    });
    client.setQueryData(whoAmI, create(WhoAmIResponseSchema, { account: { handle: "test_maya" } }));
    client.setQueryData(checkHandle, create(CheckHandleResponseSchema, { available: true }));

    await client.invalidateQueries({ queryKey: WHO_AM_I_FAMILY });

    expect(client.getQueryState(whoAmI)?.isInvalidated).toBe(true);
    expect(client.getQueryState(checkHandle)?.isInvalidated).toBe(false);
    client.clear();
  });
});
