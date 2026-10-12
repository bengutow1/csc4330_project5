import { Code, ConnectError, createClient } from "@connectrpc/connect";
import { HealthService } from "@fries-in-the-bag/apigen/health/v1/health_pb";
import { describe, expect, it } from "vitest";
import { errorMessage, fieldErrors, isUnavailable } from "./errors.ts";
import { createApiTransport } from "./transport.ts";

// Captured from the Go api: connect-go's error writer, fed what handlers.ConnectErr returns for
// FieldErrors and for ErrNotReady. Real bytes, so a shape the server never sends cannot pass.
const INVALID_ARGUMENT_BODY =
  '{"code":"invalid_argument","message":"handle is required (and 1 more)","details":[{"type":"error.v1.BadRequest","value":"ChwKBmhhbmRsZRISaGFuZGxlIGlzIHJlcXVpcmVkChwKBmF2YXRhchISYXZhdGFyIGlzIHJlcXVpcmVk","debug":{"fieldViolations":[{"field":"handle","description":"handle is required"},{"field":"avatar","description":"avatar is required"}]}}]}';
const NOT_READY_BODY = '{"code":"unavailable","message":"not ready"}';

const BASE_URL = "https://api.test.invalid";

describe("createApiTransport", () => {
  it("sends an rpc to its own path on the api, carrying the session cookie", async () => {
    const api = fakeFetch(() => json(200, "{}"));
    const health = createClient(HealthService, createApiTransport({ baseUrl: BASE_URL, ...api }));

    await health.ready({});

    expect(api.sent).toHaveLength(1);
    expect(api.sent[0]?.url).toBe(`${BASE_URL}/health.v1.HealthService/Ready`);
    expect(api.sent[0]?.init?.method).toBe("POST");
    expect(api.sent[0]?.init?.credentials).toBe("include");
  });

  it("decodes every field violation the server sent", async () => {
    const api = fakeFetch(() => json(400, INVALID_ARGUMENT_BODY));
    const health = createClient(HealthService, createApiTransport({ baseUrl: BASE_URL, ...api }));

    const err = await health.ready({}).catch((e: unknown) => e);

    expect(fieldErrors(err)).toEqual({
      handle: "handle is required",
      avatar: "avatar is required",
    });
    expect(errorMessage(err)).toBe("handle is required (and 1 more)");
  });

  it("reads a failed readiness check as unavailable, in the server's own words", async () => {
    const api = fakeFetch(() => json(503, NOT_READY_BODY));
    const health = createClient(HealthService, createApiTransport({ baseUrl: BASE_URL, ...api }));

    const err = await health.ready({}).catch((e: unknown) => e);

    expect(isUnavailable(err)).toBe(true);
    expect(err instanceof ConnectError && err.rawMessage).toBe("not ready");
  });

  // The status screen tells "not ready" from "unreachable" by this difference.
  it("reports a server it never reached as unknown, not unavailable", async () => {
    const api = fakeFetch(() => Promise.reject(new TypeError("Network request failed")));
    const health = createClient(HealthService, createApiTransport({ baseUrl: BASE_URL, ...api }));

    const err = await health.ready({}).catch((e: unknown) => e);

    expect(err instanceof ConnectError && err.code).toBe(Code.Unknown);
    expect(isUnavailable(err)).toBe(false);
    expect(errorMessage(err)).toBe("Something went wrong. Try again.");
  });
});

/** fakeFetch answers every request with one canned response and records what was sent. */
function fakeFetch(respond: () => Response | Promise<Response>) {
  const sent: { url: string; init: RequestInit | undefined }[] = [];
  const fetch: typeof globalThis.fetch = async (input, init) => {
    sent.push({ url: String(input), init });
    return respond();
  };
  return { fetch, sent };
}

function json(status: number, body: string): Response {
  return new Response(body, { status, headers: { "content-type": "application/json" } });
}
