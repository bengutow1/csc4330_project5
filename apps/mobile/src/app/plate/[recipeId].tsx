import { useLocalSearchParams } from "expo-router";
import { PlateScreen } from "@/screens/plate-screen";

export default function PlateRoute() {
  const { recipeId } = useLocalSearchParams<{ recipeId: string }>();
  return <PlateScreen recipeId={recipeId} />;
}
