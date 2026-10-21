import type { Transport } from "@connectrpc/connect";
import { TransportProvider } from "@connectrpc/connect-query";
import { type QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, useEffect } from "react";
import { type AppStateEvents, wireAppState } from "./focus.ts";

/**
 * ApiProvider hands every hook below it the transport and the query client. An app wires the
 * data layer here, once, and never imports the query libraries itself. appState is React Native's
 * AppState, which this package cannot import: it is how queries know the app came back.
 */
export function ApiProvider(props: {
  transport: Transport;
  queryClient: QueryClient;
  appState: AppStateEvents;
  children: ReactNode;
}) {
  const { appState } = props;
  useEffect(() => {
    wireAppState(appState);
  }, [appState]);

  return (
    <TransportProvider transport={props.transport}>
      <QueryClientProvider client={props.queryClient}>{props.children}</QueryClientProvider>
    </TransportProvider>
  );
}
