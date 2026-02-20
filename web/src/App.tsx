import { BrowserRouter, Routes, Route } from "react-router-dom";
import { TelegramProvider } from "./context/TelegramProvider";
import { SocketProvider } from "./context/SocketContext";
import { GameProvider } from "./context/GameContext";
import { EngineProvider } from "./context/EngineContext";
import { lazy, Suspense } from "react";

const Home = lazy(() => import("./pages/Home"));
const Rooms = lazy(() => import("./pages/Rooms"));
const Game = lazy(() => import("./pages/Game"));
const Profile = lazy(() => import("./pages/Profile"));
const Store = lazy(() => import("./pages/Store"));

const isPrototype = import.meta.env.VITE_PROTOTYPE === "true";

function GameProviderSwitch({ children }: { children: React.ReactNode }) {
  if (isPrototype) {
    return <EngineProvider>{children}</EngineProvider>;
  }
  return (
    <SocketProvider>
      <GameProvider>{children}</GameProvider>
    </SocketProvider>
  );
}

export function App() {
  return (
    <TelegramProvider>
      <GameProviderSwitch>
        <BrowserRouter>
          <Suspense>
            <Routes>
              <Route path="/" element={<Home />} />
              <Route path="/rooms" element={<Rooms />} />
              <Route path="/game" element={<Game />} />
              <Route path="/profile" element={<Profile />} />
              <Route path="/store" element={<Store />} />
            </Routes>
          </Suspense>
        </BrowserRouter>
      </GameProviderSwitch>
    </TelegramProvider>
  );
}
