import * as Location from "expo-location";
import { useEffect, useState } from "react";
import { snapPoint } from "@/lib/spots";

/** Point is where the phone is, in degrees. */
export type Point = { lat: number; lng: number };

/**
 * usePlayerPoint is where the phone is, once the player has let the app look. It never asks by
 * itself: ask() shows the prompt, which the fries moment calls when the player taps Find food. A
 * phone that cannot fix its place (location services off) counts as off, so the screen offers the
 * search box. The point stays on the phone except for the one ListSpots call that uses it.
 */
export function usePlayerPoint() {
  const [permission, requestPermission] = Location.useForegroundPermissions();
  const [point, setPoint] = useState<Point>();
  const [unfixable, setUnfixable] = useState(false);
  const granted = permission?.granted === true;

  useEffect(() => {
    if (!granted) {
      return;
    }
    let live = true;
    Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced })
      .then((fix) => {
        if (live) {
          setPoint(snapPoint(fix.coords.latitude, fix.coords.longitude));
        }
      })
      // No fix to be had: the screen shows the search box instead, which says why.
      .catch(() => {
        if (live) {
          setUnfixable(true);
        }
      });
    return () => {
      live = false;
    };
  }, [granted]);

  return {
    /** Still finding out whether the app may look, or where the phone is. */
    locating: permission === null || (granted && point === undefined && !unfixable),
    /** The player has let the app look. */
    granted,
    /** Asking again would show the prompt. False once the player has said no for good. */
    canAsk: permission?.canAskAgain ?? true,
    point,
    /** There is no point to search around: the player said no, or the phone could not tell. */
    off: permission !== null && (!granted || unfixable),
    ask: requestPermission,
  };
}
