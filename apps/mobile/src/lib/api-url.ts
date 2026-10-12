/**
 * apiUrl picks where the api lives for a build. EXPO_PUBLIC_API_URL wins when it is set, for a
 * phone on the LAN (localhost on a phone is the phone). Otherwise the Android emulator reaches the
 * Mac's localhost as 10.0.2.2, the iOS simulator shares the Mac's localhost, and web is
 * same-origin.
 */
export function apiUrl(os: string | undefined, override: string | undefined): string {
  if (override) {
    return override;
  }
  switch (os) {
    case "android":
      return "http://10.0.2.2:8080";
    case "web":
      return "/";
    case "ios":
      return "http://localhost:8080";
    default:
      // A platform Expo added after this was written: the simulator's answer is the best guess.
      return "http://localhost:8080";
  }
}

/** API_URL is this build's answer. Expo inlines both variables when it bundles. */
export const API_URL = apiUrl(process.env.EXPO_OS, process.env.EXPO_PUBLIC_API_URL);
