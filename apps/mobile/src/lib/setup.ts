import { createContext, useContext } from "react";

/**
 * Setup is the stretch of onboarding after the account exists and before the kitchen is saved.
 * While it runs, the root layout keeps the kitchen screen open and the tabs shut, so making the
 * account doesn't skip straight past O3.
 */
export interface Setup {
  /** The handle screen calls it as it makes the account, before the session exists. */
  start: () => void;
  /** The kitchen screen calls it once the kitchen is saved, which opens the tabs. */
  finish: () => void;
}

/** SetupContext carries the root layout's setup state down to the onboarding screens. */
export const SetupContext = createContext<Setup>({ start: () => {}, finish: () => {} });

/** useSetup is how an onboarding screen moves the root layout's guards. */
export function useSetup(): Setup {
  return useContext(SetupContext);
}
