import { useLocalSearchParams } from "expo-router";
import { VerdictScreen } from "@/screens/verdict-screen";

export default function VerdictRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <VerdictScreen id={id} />;
}
