import { useEffect } from "react";
import { RoomCard } from "@/components/rooms/RoomCard";
import { NavBar } from "@/components/NavBar";
import { useGame } from "@/context/GameContext";
import { useSocket } from "@/context/SocketContext";

export default function Rooms() {
  const { state, listRooms } = useGame();
  const { connected } = useSocket();

  useEffect(() => {
    if (connected) listRooms();
    const id = setInterval(() => {
      if (connected) listRooms();
    }, 5000);
    return () => clearInterval(id);
  }, [connected, listRooms]);

  const waitingRooms = state.rooms.filter((r) => r.state === "waiting");
  const activeRooms = state.rooms.filter(
    (r) => r.state === "active" || r.state === "survival",
  );

  return (
    <main className="min-h-screen bg-background">
      <NavBar title="Available Rooms" showStore={false} />

      <div className="max-w-7xl mx-auto px-3 sm:px-4 py-6 sm:py-12">
        <div className="mb-8 sm:mb-12">
          <h1 className="text-3xl sm:text-4xl md:text-5xl font-bold text-foreground mb-3 sm:mb-4">
            Available Rooms
          </h1>
          <p className="text-base sm:text-lg text-muted-foreground max-w-2xl">
            Join a room to enter the survival phase. Higher tier = higher stakes
            and volatility.
          </p>
        </div>

        {waitingRooms.length > 0 && (
          <section className="mb-10">
            <h2 className="text-xl sm:text-2xl font-bold text-foreground mb-4 sm:mb-6">
              Open Lobbies
            </h2>
            <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
              {waitingRooms.map((room) => (
                <RoomCard key={room.id} room={room} />
              ))}
            </div>
          </section>
        )}

        {activeRooms.length > 0 && (
          <section className="mb-10">
            <h2 className="text-xl sm:text-2xl font-bold text-foreground mb-4 sm:mb-6">
              In Progress
            </h2>
            <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
              {activeRooms.map((room) => (
                <RoomCard key={room.id} room={room} />
              ))}
            </div>
          </section>
        )}

        {state.rooms.length === 0 && (
          <div className="text-center py-20">
            <p className="text-muted-foreground text-lg mb-4">
              {connected ? "No rooms available" : "Connecting to server..."}
            </p>
          </div>
        )}
      </div>
    </main>
  );
}
