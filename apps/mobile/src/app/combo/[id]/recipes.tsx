import { useLocalSearchParams } from "expo-router";
import { ComboRecipesScreen } from "@/screens/combo-recipes-screen";

export default function ComboRecipesRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <ComboRecipesScreen id={id} />;
}
