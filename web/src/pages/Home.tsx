import { HeroSection } from "@/components/landing/HeroSection";
import { NavBar } from "@/components/NavBar";

export default function Home() {
  return (
    <main className="min-h-screen bg-background">
      <NavBar showProfile showStore />
      <HeroSection />
    </main>
  );
}
