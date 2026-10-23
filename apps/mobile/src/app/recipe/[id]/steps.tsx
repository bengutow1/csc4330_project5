import { useLocalSearchParams } from "expo-router";
import { CookingStepsScreen } from "@/screens/cooking-steps-screen";

export default function CookingStepsRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <CookingStepsScreen id={id} />;
}
