import { BrowserRouter, Routes, Route } from "react-router-dom";
import { TelegramProvider } from "./context/TelegramProvider";
import { SocketProvider } from "./context/SocketContext";
import { GameProvider } from "./context/GameContext";
import { lazy, Suspense } from "react";

const Home = lazy(() => import("./pages/Home"));
const Rooms = lazy(() => import("./pages/Rooms"));
const Game = lazy(() => import("./pages/Game"));
const Profile = lazy(() => import("./pages/Profile"));
const Store = lazy(() => import("./pages/Store"));

export function App() {
  return (
    <TelegramProvider>
      <SocketProvider>
        <GameProvider>
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
        </GameProvider>
      </SocketProvider>
    </TelegramProvider>
  );
}
