import { Redirect } from "expo-router";
import { KitScreen } from "@/screens/kit-screen";

// The kit is for building and checking screens, so a release build sends the link home.
export default function KitRoute() {
  return __DEV__ ? <KitScreen /> : <Redirect href="/" />;
}
