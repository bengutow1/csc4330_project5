import { useLocalSearchParams } from "expo-router";
import { RecipesScreen } from "@/screens/recipes-screen";

export default function RecipesRoute() {
  const { scanId } = useLocalSearchParams<{ scanId: string }>();
  return <RecipesScreen scanId={scanId} />;
}
