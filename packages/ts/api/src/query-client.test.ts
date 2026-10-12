import { MutationObserver } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import { makeQueryClient } from "./query-client.ts";

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
