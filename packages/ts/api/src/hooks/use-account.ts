import { skipToken, useMutation, useQuery } from "@connectrpc/connect-query";
import { AccountService } from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { useQueryClient } from "@tanstack/react-query";

// WhoAmI takes no input. A module constant keeps the query key the same on every render.
const WHO_AM_I_INPUT = {};

// The shortest handle the server takes. Anything shorter is not worth a round trip.
const MIN_HANDLE_LENGTH = 3;

/**
 * useWhoAmI asks whose session this device holds. Retry is off: unauthenticated is an answer
 * (this device has no account yet), not a blip to hide.
 */
export function useWhoAmI() {
  return useQuery(AccountService.method.whoAmI, WHO_AM_I_INPUT, { retry: false });
}

/**
 * useCreateAccount makes the device account and signs this device in. Success resets every
 * query rather than invalidating it, which would keep the previous session's answers on screen.
 */
export function useCreateAccount() {
  const queryClient = useQueryClient();
  return useMutation(AccountService.method.createAccount, {
    onSuccess: () => queryClient.resetQueries(),
  });
}

/**
 * useCheckHandle says whether a handle is free while the player types it, and waits until it is
 * long enough to ask about. Retry is off: a malformed handle is an answer too.
 */
export function useCheckHandle(handle: string) {
  return useQuery(AccountService.method.checkHandle, checkHandleInput(handle), { retry: false });
}

/** checkHandleInput is what useCheckHandle sends for what was typed: skipToken until 3 characters. */
export function checkHandleInput(handle: string) {
  return handle.trim().length < MIN_HANDLE_LENGTH ? skipToken : { handle };
}
