import { createConnectQueryKey } from "@connectrpc/connect-query";
import { AccountService } from "@fries-in-the-bag/apigen/account/v1/account_pb";
import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import { isUnauthenticated } from "./errors.ts";

// Every cached WhoAmI, whichever transport asked it.
const WHO_AM_I = createConnectQueryKey({
  schema: AccountService.method.whoAmI,
  cardinality: undefined,
});

/**
 * The toast seam. The data layer decides WHEN something is worth saying (a call failed, a
 * mutation declared a success message); the app decides HOW, which keeps any toast library, and
 * any platform, out of this package.
 */
export interface ToastHandlers {
  error?: (error: unknown) => void;
  success?: (message: string) => void;
}

// A mutation opts into a success toast by declaring it in meta, as a string or, when the message
// depends on what was sent, a function of the result and the input.
declare module "@tanstack/react-query" {
  interface Register {
    mutationMeta: {
      successToast?: string | ((data: unknown, variables: unknown) => string);
    };
  }
}

/**
 * makeQueryClient builds the one query client an app holds. Build it once, at the root: a client
 * rebuilt on render drops the whole cache.
 */
export function makeQueryClient(toast: ToastHandlers = {}): QueryClient {
  // Every failure surfaces once, app-wide, with no per-hook wiring. Unauthenticated is the
  // exception: the session is dead, so WhoAmI forgets the account and the root layout returns to
  // onboarding, with nothing to toast. Only an account WhoAmI still holds is reset; otherwise its
  // own unauthenticated answer would reset it, refetch it and fail again, forever.
  const onError = (error: unknown) => {
    if (!isUnauthenticated(error)) {
      toast.error?.(error);
      return;
    }
    const holdsAccount = client
      .getQueriesData({ queryKey: WHO_AM_I })
      .some(([, data]) => data !== undefined);
    if (holdsAccount) {
      void client.resetQueries({ queryKey: WHO_AM_I });
    }
  };
  const client = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60_000,
        refetchOnWindowFocus: true,
        // One retry, not three: a broken api should show itself quickly rather than look slow.
        retry: 1,
      },
    },
    mutationCache: new MutationCache({
      onError,
      onSuccess: (data, variables, _onMutateResult, mutation) => {
        const declared = mutation.meta?.successToast;
        const message = typeof declared === "function" ? declared(data, variables) : declared;
        if (message) {
          toast.success?.(message);
        }
      },
    }),
    queryCache: new QueryCache({ onError }),
  });
  return client;
}
