import { useLocalSearchParams } from "expo-router";
import { ComboBringScreen } from "@/screens/combo-bring-screen";

export default function ComboBringRoute() {
  const { id, recipeId } = useLocalSearchParams<{ id: string; recipeId: string }>();
  return <ComboBringScreen key={`${id}-${recipeId}`} id={id} recipeId={recipeId} />;
}
