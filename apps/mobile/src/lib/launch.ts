/**
 * OPEN_ROUTES are the routes in none of the root layout's guards, by their first segment. Whose
 * phone this is changes nothing there, so the launch cover leaves them alone: /status has to be
 * able to say the API is down, and the kit needs only Metro.
 */
export const OPEN_ROUTES: ReadonlySet<string> = new Set(["kit", "status"]);

/** LaunchState is what the root layout knows about whose phone this is, and where it is. */
export type LaunchState = {
  /** WhoAmI has answered with an account. */
  signedIn: boolean;
  /** The server said there is no session: this phone has no account yet. */
  noSession: boolean;
  /** The first segment of the route on screen, undefined until the first route resolves. */
  route: string | undefined;
};

/**
 * launchCoverShows says whether the launch cover hides the screen: only while the app can't yet
 * tell whose phone this is (still asking, or the server can't be reached), because reading that
 * silence as "no account" would offer a returning player onboarding. And only over a guarded
 * route, or before the first route resolves.
 */
export function launchCoverShows(state: LaunchState): boolean {
  if (state.signedIn || state.noSession) {
    return false;
  }
  return state.route === undefined || !OPEN_ROUTES.has(state.route);
}
