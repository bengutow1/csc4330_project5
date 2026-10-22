import { useLocalSearchParams } from "expo-router";
import { RecipeScreen } from "@/screens/recipe-screen";

export default function RecipeRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <RecipeScreen id={id} />;
}
