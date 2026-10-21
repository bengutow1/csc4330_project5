// @fries-in-the-bag/api, the client data layer. Apps import hooks and helpers from here, and
// @fries-in-the-bag/apigen never leaks past this package, so the schema can change shape without
// every screen knowing.
//
// Universal by contract: no react-native, no react-dom, no DOM globals.

export {
  errorMessage,
  fieldErrors,
  isFailedPrecondition,
  isNotFound,
  isUnauthenticated,
  isUnavailable,
} from "./errors.ts";
export type { AppStateEvents } from "./focus.ts";
export { useCheckHandle, useCreateAccount, useWhoAmI } from "./hooks/use-account.ts";
export { useReady } from "./hooks/use-health.ts";
export { ApiProvider } from "./provider.tsx";
export { makeQueryClient, type ToastHandlers } from "./query-client.ts";
export { type ApiTransportOptions, createApiTransport } from "./transport.ts";
