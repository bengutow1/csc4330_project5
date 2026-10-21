import { focusManager } from "@tanstack/react-query";

/**
 * The slice of React Native's AppState the data layer listens to. The app's root passes AppState
 * itself; this package never imports react-native, so it names only what it calls.
 */
export interface AppStateEvents {
  addEventListener(type: "change", listener: (status: string) => void): { remove(): void };
}

/**
 * wireAppState tells TanStack Query when the app is in the foreground, so a stale query refetches
 * when the player comes back to it. React Native has no window focus, so refetchOnWindowFocus does
 * nothing until this runs.
 */
export function wireAppState(appState: AppStateEvents): void {
  focusManager.setEventListener((setFocused) => {
    const subscription = appState.addEventListener("change", (status) => {
      setFocused(status === "active");
    });
    return () => subscription.remove();
  });
}
