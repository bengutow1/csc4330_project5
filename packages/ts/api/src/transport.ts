import type { Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

/** What an app's root hands createApiTransport: the platform differences, as arguments. */
export interface ApiTransportOptions {
  /** Where the api lives: an absolute url on native (a phone has no origin), "/" on web. */
  baseUrl: string;
  /** The fetch to send with, when the global one will not do: a test's fake, or a platform's own. */
  fetch?: typeof globalThis.fetch;
}

/**
 * createApiTransport is a factory, not a module singleton, so each app builds one at its root
 * with its own base url and fetch, and this package stays free of platform code.
 */
export function createApiTransport(opts: ApiTransportOptions): Transport {
  return createConnectTransport({
    baseUrl: opts.baseUrl,
    // Sessions are http-only cookies, so every request carries credentials. connect-web has no
    // option for that, so it goes on the fetch.
    fetch: (input, init) =>
      (opts.fetch ?? globalThis.fetch)(input, { ...init, credentials: "include" }),
  });
}
