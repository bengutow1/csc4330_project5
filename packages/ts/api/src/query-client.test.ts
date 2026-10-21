import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import {
  AccountService,
  WhoAmIResponseSchema,
} from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { MutationObserver, QueryObserver } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { makeQueryClient } from "./query-client.ts";
import { createApiTransport } from "./transport.ts";

describe("makeQueryClient", () => {
  it("hands every failed query to the app's error seam", async () => {
    const errors: unknown[] = [];
    const client = makeQueryClient({ error: (e) => errors.push(e) });
    const boom = new Error("boom");

    await client
      .fetchQuery({ queryKey: ["probe"], queryFn: () => Promise.reject(boom), retry: false })
      .catch(() => undefined);

    expect(errors).toEqual([boom]);
    client.clear();
  });

  it("hands every failed mutation to the app's error seam", async () => {
    const errors: unknown[] = [];
    const client = makeQueryClient({ error: (e) => errors.push(e) });
    const boom = new Error("boom");
    const save = new MutationObserver(client, { mutationFn: () => Promise.reject(boom) });

    await save.mutate().catch(() => undefined);

    expect(errors).toEqual([boom]);
    client.clear();
  });

  it("toasts a success only for a mutation that declares one", async () => {
    const said: string[] = [];
    const client = makeQueryClient({ success: (m) => said.push(m) });
    const quiet = new MutationObserver(client, { mutationFn: () => Promise.resolve("ok") });
    const loud = new MutationObserver(client, {
      mutationFn: (n: number) => Promise.resolve(n * 2),
      meta: {
        successToast: (data, variables) => `Doubled ${String(variables)} to ${String(data)}`,
      },
    });

    await quiet.mutate();
    await loud.mutate(21);

    expect(said).toEqual(["Doubled 21 to 42"]);
    client.clear();
  });
});

describe("makeQueryClient once accounts exist", () => {
  // The key useWhoAmI caches under, built the way connect-query builds it for a screen.
  const transport = createApiTransport({ baseUrl: "http://api.test.invalid" });
  const whoAmIKey = createConnectQueryKey({
    schema: AccountService.method.whoAmI,
    input: {},
    transport,
    cardinality: "finite",
  });
  const signedIn = create(WhoAmIResponseSchema, { account: { handle: "test_maya", avatar: "🍟" } });

  function failWith(client: ReturnType<typeof makeQueryClient>, code: Code) {
    return client
      .fetchQuery({
        queryKey: ["scans"],
        queryFn: () => Promise.reject(new ConnectError("refused", code)),
        retry: false,
      })
      .catch(() => undefined);
  }

  it("forgets who is signed in when any call comes back unauthenticated, without a toast", async () => {
    const errors: unknown[] = [];
    const client = makeQueryClient({ error: (e) => errors.push(e) });
    client.setQueryData(whoAmIKey, signedIn);

    await failWith(client, Code.Unauthenticated);

    expect(client.getQueryData(whoAmIKey)).toBeUndefined();
    expect(errors).toEqual([]);
    client.clear();
  });

  it("does the same for a mutation", async () => {
    const client = makeQueryClient();
    client.setQueryData(whoAmIKey, signedIn);
    const save = new MutationObserver(client, {
      mutationFn: () => Promise.reject(new ConnectError("refused", Code.Unauthenticated)),
    });

    await save.mutate().catch(() => undefined);

    expect(client.getQueryData(whoAmIKey)).toBeUndefined();
    client.clear();
  });

  it("keeps the account through any other failure, and toasts it", async () => {
    const errors: unknown[] = [];
    const client = makeQueryClient({ error: (e) => errors.push(e) });
    client.setQueryData(whoAmIKey, signedIn);

    await failWith(client, Code.Unavailable);

    expect(client.getQueryData(whoAmIKey)).toEqual(signedIn);
    expect(errors).toHaveLength(1);
    client.clear();
  });

  it("asks WhoAmI once when it is the call that says unauthenticated", async () => {
    const client = makeQueryClient();
    let asks = 0;
    const observer = new QueryObserver(client, {
      queryKey: whoAmIKey,
      queryFn: () => {
        asks += 1;
        // A reset loop would ask forever. Answering the third ask ends one, so the test fails
        // instead of hanging.
        if (asks >= 3) {
          return Promise.resolve(signedIn);
        }
        return Promise.reject(new ConnectError("not signed in", Code.Unauthenticated));
      },
      retry: false,
    });
    const stop = observer.subscribe(() => {});

    await vi.waitFor(() => expect(observer.getCurrentResult().isFetching).toBe(false));
    // Nothing should ask again; give a loop the time it would need to show itself.
    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(asks).toBe(1);
    stop();
    client.clear();
  });
});
