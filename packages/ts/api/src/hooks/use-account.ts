import { createConnectQueryKey, skipToken, useMutation, useQuery } from "@connectrpc/connect-query";
import { AccountService } from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { CookService } from "@fries-in-the-bag/apigen/cook/v1/cook_pb";
import {
  type MutationFilters,
  type QueryClient,
  useIsMutating,
  useQueryClient,
} from "@tanstack/react-query";

// WhoAmI takes no input. A module constant keeps the query key the same on every render.
const WHO_AM_I_INPUT = {};

/**
 * WHO_AM_I_FAMILY is every cached WhoAmI, whichever transport asked it. A write to the account
 * invalidates it, and an unauthenticated answer resets it (query-client.ts).
 */
export const WHO_AM_I_FAMILY = createConnectQueryKey({
  schema: AccountService.method.whoAmI,
  cardinality: undefined,
});

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
 * CREATING_ACCOUNT finds a CreateAccount this device still has out, from whichever screen sent
 * it. A screen the player backs out of keeps its call running, and a second account would strand
 * the first.
 */
export const CREATING_ACCOUNT = {
  mutationKey: createConnectQueryKey({
    schema: AccountService.method.createAccount,
    cardinality: undefined,
  }),
} satisfies MutationFilters;

/**
 * useCreateAccount makes the device account and signs this device in. Success resets every
 * query rather than invalidating it, which would keep the previous session's answers on screen.
 */
export function useCreateAccount() {
  const queryClient = useQueryClient();
  return useMutation(AccountService.method.createAccount, createAccountOptions(queryClient));
}

/** createAccountOptions is what useCreateAccount sends with, CREATING_ACCOUNT's key included. */
export function createAccountOptions(queryClient: QueryClient) {
  return {
    mutationKey: CREATING_ACCOUNT.mutationKey,
    onSuccess: () => queryClient.resetQueries(),
  };
}

/** useCreatingAccount says whether a CreateAccount is still out (CREATING_ACCOUNT). */
export function useCreatingAccount() {
  return useIsMutating(CREATING_ACCOUNT) > 0;
}

/**
 * useDeleteAccount deletes the player's account and everything of theirs. Success resets every
 * query, as making one does, so WhoAmI asks again, finds no session and onboarding opens.
 */
export function useDeleteAccount() {
  const queryClient = useQueryClient();
  return useMutation(AccountService.method.deleteAccount, {
    onSuccess: () => queryClient.resetQueries(),
  });
}

// Every recipe read. The recipes leave out what the player can't eat or cook with, so shelves
// cached before an allergy was saved would still offer a recipe that contains it.
const RECIPE_FAMILY = createConnectQueryKey({ schema: CookService, cardinality: undefined });

/**
 * useInvalidateAccount marks the session's account stale, and every recipe read cut to its
 * kitchen, so every screen showing either refetches. It is the account family's one invalidator.
 */
export function useInvalidateAccount() {
  const queryClient = useQueryClient();
  return () => invalidateAccount(queryClient);
}

/** invalidateAccount is useInvalidateAccount's work, on whichever query client it is handed. */
export function invalidateAccount(queryClient: QueryClient) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: WHO_AM_I_FAMILY }),
    queryClient.invalidateQueries({ queryKey: RECIPE_FAMILY }),
  ]);
}

/**
 * useUpdateKitchen saves the player's whole kitchen. The account carries the kitchen and the
 * recipes are cut to it, so both refetch once the write settles, and the mutation finishes only
 * after they have.
 */
export function useUpdateKitchen() {
  const invalidate = useInvalidateAccount();
  return useMutation(AccountService.method.updateKitchen, {
    onSettled: () => invalidate(),
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
