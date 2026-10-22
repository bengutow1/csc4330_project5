import { PlaceholderScreen } from "@/screens/placeholder-screen";

export default function MeRoute() {
  return (
    <PlaceholderScreen
      title="Profile"
      issue={30}
      action={{ icon: "settings-outline", label: "Settings", href: "/settings" }}
    />
  );
}
