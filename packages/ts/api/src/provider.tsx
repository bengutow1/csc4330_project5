import type { Transport } from "@connectrpc/connect";
import { TransportProvider } from "@connectrpc/connect-query";
import { type QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

/**
 * ApiProvider hands every hook below it the transport and the query client. An app wires the
 * data layer here, once, and never imports the query libraries itself.
 */
export function ApiProvider(props: {
  transport: Transport;
  queryClient: QueryClient;
  children: ReactNode;
}) {
  return (
    <TransportProvider transport={props.transport}>
      <QueryClientProvider client={props.queryClient}>{props.children}</QueryClientProvider>
    </TransportProvider>
  );
}
