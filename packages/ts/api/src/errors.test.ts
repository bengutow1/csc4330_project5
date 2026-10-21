import { Code, ConnectError, createClient } from "@connectrpc/connect";
import { ScanService } from "@fries-in-the-bag/apigen/scan/v1/scan_pb";
import { describe, expect, it } from "vitest";
import {
  errorMessage,
  fieldErrors,
  isFailedPrecondition,
  isNotFound,
  isUnauthenticated,
  isUnavailable,
  isUnreachable,
} from "./errors.ts";
import { createApiTransport } from "./transport.ts";

const GENERIC = "Something went wrong. Try again.";

describe("errorMessage", () => {
  // Mirrors the arms of handlers.ConnectErr whose text is written for a person.
  const written: Code[] = [
    Code.Unauthenticated,
    Code.PermissionDenied,
    Code.InvalidArgument,
    Code.NotFound,
    Code.AlreadyExists,
    Code.FailedPrecondition,
    Code.ResourceExhausted,
  ];
  const redacted: Code[] = [Code.Internal, Code.Unknown, Code.Unavailable, Code.DeadlineExceeded];

  it.each(written)("shows the server's own words for code %s", (code) => {
    expect(errorMessage(new ConnectError("that combo is full", code))).toBe("that combo is full");
  });

  it.each(redacted)("says something generic for code %s, whatever the server wrote", (code) => {
    expect(errorMessage(new ConnectError("dial tcp 10.0.0.4:5432", code))).toBe(GENERIC);
  });

  it("says something generic for an error that did not come from the api", () => {
    expect(errorMessage(new Error("undefined is not a function"))).toBe(GENERIC);
    expect(errorMessage("a thrown string")).toBe(GENERIC);
  });
});

describe("fieldErrors", () => {
  it("is empty when there is nothing to put under an input", () => {
    expect(fieldErrors(new Error("boom"))).toEqual({});
    expect(fieldErrors(new ConnectError("that combo is full", Code.ResourceExhausted))).toEqual({});
  });
});

describe("predicates", () => {
  const cases = [
    { name: "isUnauthenticated", is: isUnauthenticated, code: Code.Unauthenticated },
    { name: "isNotFound", is: isNotFound, code: Code.NotFound },
    { name: "isFailedPrecondition", is: isFailedPrecondition, code: Code.FailedPrecondition },
    { name: "isUnavailable", is: isUnavailable, code: Code.Unavailable },
    { name: "isUnreachable", is: isUnreachable, code: Code.Unknown },
  ];

  it.each(cases)("$name matches its own code and nothing else", ({ is, code }) => {
    expect(is(new ConnectError("x", code))).toBe(true);
    expect(is(new ConnectError("x", Code.Internal))).toBe(false);
    expect(is(new Error("x"))).toBe(false);
  });
});

describe("isUnreachable", () => {
  // Each error comes out of a real transport, so the test sees what connect makes of a dead
  // network or a server's answer, not a code we assumed it would pick.
  async function failureFrom(fetch: typeof globalThis.fetch): Promise<unknown> {
    const transport = createApiTransport({ baseUrl: "http://api.test.invalid", fetch });
    try {
      await createClient(ScanService, transport).getLastScan({});
    } catch (err) {
      return err;
    }
    throw new Error("the call succeeded, so there is no failure to check");
  }

  it("is a request that never got an answer", async () => {
    const err = await failureFrom(() => Promise.reject(new TypeError("Network request failed")));

    expect(isUnreachable(err)).toBe(true);
  });

  it("is not a server that answered, even to say it cannot serve right now", async () => {
    const answer = JSON.stringify({ code: "unavailable", message: "The judge is out." });
    const err = await failureFrom(
      async () =>
        new Response(answer, { status: 503, headers: { "content-type": "application/json" } }),
    );

    expect(isUnavailable(err)).toBe(true);
    expect(isUnreachable(err)).toBe(false);
  });
});
