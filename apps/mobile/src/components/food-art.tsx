import { Image, type ImageSource } from "expo-image";
import type { ReactNode } from "react";
import { type StyleProp, StyleSheet, View, type ViewStyle } from "react-native";
import { foodSlug } from "@/lib/food-slug";

// A photo for every ingredient and recipe the vision mock names. A food without one (a real
// model's list can name anything) keeps the icon its caller draws.
const PHOTOS: Record<string, ImageSource> = {
  apple: require("../../assets/food/apple.jpg"),
  "bell-peppers": require("../../assets/food/bell-peppers.jpg"),
  cheddar: require("../../assets/food/cheddar.jpg"),
  chicken: require("../../assets/food/chicken.jpg"),
  eggs: require("../../assets/food/eggs.jpg"),
  "energy-drinks": require("../../assets/food/energy-drinks.jpg"),
  "frozen-peas": require("../../assets/food/frozen-peas.jpg"),
  "greek-yogurt": require("../../assets/food/greek-yogurt.jpg"),
  "ground-beef": require("../../assets/food/ground-beef.jpg"),
  "hot-sauce": require("../../assets/food/hot-sauce.jpg"),
  "instant-ramen": require("../../assets/food/instant-ramen.jpg"),
  ketchup: require("../../assets/food/ketchup.jpg"),
  lemon: require("../../assets/food/lemon.jpg"),
  "mystery-container": require("../../assets/food/mystery-container.jpg"),
  onion: require("../../assets/food/onion.jpg"),
  "orange-juice": require("../../assets/food/orange-juice.jpg"),
  salsa: require("../../assets/food/salsa.jpg"),
  "shredded-cheese": require("../../assets/food/shredded-cheese.jpg"),
  soda: require("../../assets/food/soda.jpg"),
  spinach: require("../../assets/food/spinach.jpg"),
  tortillas: require("../../assets/food/tortillas.jpg"),
  "beef-tacos": require("../../assets/food/beef-tacos.jpg"),
  "cheese-quesadilla": require("../../assets/food/cheese-quesadilla.jpg"),
  "chicken-fajita-bowl": require("../../assets/food/chicken-fajita-bowl.jpg"),
  "lemon-water-fancy": require("../../assets/food/lemon-water-fancy.jpg"),
  "microwave-baked-apple": require("../../assets/food/microwave-baked-apple.jpg"),
  "sad-egg-on-toast": require("../../assets/food/sad-egg-on-toast.jpg"),
  "spicy-pea-ramen": require("../../assets/food/spicy-pea-ramen.jpg"),
  "spinach-cheddar-omelette": require("../../assets/food/spinach-cheddar-omelette.jpg"),
  "steak-frites": require("../../assets/food/steak-frites.jpg"),
  "yogurt-parfait": require("../../assets/food/yogurt-parfait.jpg"),
};

/**
 * FoodArt is the square that shows a food: its photo when there is one, otherwise the fallback,
 * on the box's own background.
 */
export function FoodArt(props: { name: string; style: StyleProp<ViewStyle>; children: ReactNode }) {
  const photo = PHOTOS[foodSlug(props.name)];
  return (
    <View style={[props.style, styles.clip]}>
      {photo === undefined ? (
        props.children
      ) : (
        <Image source={photo} style={StyleSheet.absoluteFill} contentFit="cover" transition={150} />
      )}
    </View>
  );
}

const styles = StyleSheet.create({ clip: { overflow: "hidden" } });
