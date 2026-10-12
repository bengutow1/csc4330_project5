import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";

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
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60_000,
        refetchOnWindowFocus: true,
        // One retry, not three: a broken api should show itself quickly rather than look slow.
        retry: 1,
      },
    },
    // Every failure surfaces once, app-wide, with no per-hook wiring.
    mutationCache: new MutationCache({
      onError: (error) => toast.error?.(error),
      onSuccess: (data, variables, _onMutateResult, mutation) => {
        const declared = mutation.meta?.successToast;
        const message = typeof declared === "function" ? declared(data, variables) : declared;
        if (message) {
          toast.success?.(message);
        }
      },
    }),
    queryCache: new QueryCache({
      onError: (error) => toast.error?.(error),
    }),
  });
}
