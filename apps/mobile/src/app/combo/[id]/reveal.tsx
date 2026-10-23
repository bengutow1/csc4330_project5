import { useLocalSearchParams } from "expo-router";
import { ComboRevealScreen } from "@/screens/combo-reveal-screen";

export default function ComboRevealRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <ComboRevealScreen id={id} />;
}
