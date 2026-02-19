"use client";

import {
  createContext,
  useContext,
  useEffect,
  useReducer,
  useCallback,
  type ReactNode,
} from "react";
import { useSocket } from "@/context/SocketContext";
import { useTelegram } from "@/context/TelegramProvider";
import { getPlayer } from "@/lib/api";
import type {
  RoomInfo,
  RoomStatePayload,
  TickPayload,
  EliminationPayload,
  PulseAckPayload,
  PlayerProfile,
} from "@/types/game";

interface GameState {
  player: PlayerProfile | null;
  rooms: RoomInfo[];
  currentRoom: RoomStatePayload | null;
  marginHistory: number[];
  eliminated: number[];
  lastPulseAck: PulseAckPayload | null;
  isInRoom: boolean;
}

const initialState: GameState = {
  player: null,
  rooms: [],
  currentRoom: null,
  marginHistory: [],
  eliminated: [],
  lastPulseAck: null,
  isInRoom: false,
};

type Action =
  | { type: "SET_PLAYER"; player: PlayerProfile }
  | { type: "SET_ROOMS"; rooms: RoomInfo[] }
  | { type: "ROOM_STATE"; payload: RoomStatePayload }
  | { type: "TICK"; payload: TickPayload }
  | { type: "ELIMINATION"; payload: EliminationPayload }
  | { type: "PULSE_ACK"; payload: PulseAckPayload }
  | { type: "LEAVE_ROOM" };

function reducer(state: GameState, action: Action): GameState {
  switch (action.type) {
    case "SET_PLAYER":
      return { ...state, player: action.player };

    case "SET_ROOMS":
      return { ...state, rooms: action.rooms };

    case "ROOM_STATE":
      return {
        ...state,
        currentRoom: action.payload,
        isInRoom: true,
        marginHistory:
          state.currentRoom?.room_id === action.payload.room_id
            ? state.marginHistory
            : [action.payload.margin_ratio],
        eliminated:
          state.currentRoom?.room_id === action.payload.room_id
            ? state.eliminated
            : [],
      };

    case "TICK": {
      if (!state.currentRoom) return state;
      const history = [...state.marginHistory, action.payload.margin_ratio];
      if (history.length > 120) history.shift();
      return {
        ...state,
        currentRoom: {
          ...state.currentRoom,
          timer_ms: action.payload.timer_ms,
          margin_ratio: action.payload.margin_ratio,
          volatility_mul: action.payload.volatility_mul,
          alive: action.payload.alive,
        },
        marginHistory: history,
      };
    }

    case "ELIMINATION": {
      if (!state.currentRoom) return state;
      return {
        ...state,
        currentRoom: {
          ...state.currentRoom,
          alive: action.payload.alive,
        },
        eliminated: [...state.eliminated, action.payload.player_id],
      };
    }

    case "PULSE_ACK": {
      if (!state.currentRoom) return state;
      return {
        ...state,
        currentRoom: {
          ...state.currentRoom,
          timer_ms: action.payload.timer_ms,
        },
        lastPulseAck: action.payload,
      };
    }

    case "LEAVE_ROOM":
      return {
        ...state,
        currentRoom: null,
        isInRoom: false,
        marginHistory: [],
        eliminated: [],
        lastPulseAck: null,
      };

    default:
      return state;
  }
}

interface GameContextType {
  state: GameState;
  listRooms: () => void;
  joinRoom: (roomId: string) => void;
  pulse: () => void;
  leaveRoom: () => void;
  refreshPlayer: () => void;
}

const GameContext = createContext<GameContextType | undefined>(undefined);

export function useGame() {
  const ctx = useContext(GameContext);
  if (!ctx) throw new Error("useGame must be used within GameProvider");
  return ctx;
}

export function GameProvider({ children }: { children: ReactNode }) {
  const { send, on, connected } = useSocket();
  const { userId } = useTelegram();
  const [state, dispatch] = useReducer(reducer, initialState);

  const refreshPlayer = useCallback(() => {
    if (!userId) return;
    getPlayer(userId)
      .then((p) => dispatch({ type: "SET_PLAYER", player: p }))
      .catch(() => {});
  }, [userId]);

  useEffect(() => {
    if (userId && connected) refreshPlayer();
  }, [userId, connected, refreshPlayer]);

  useEffect(() => {
    const unsubs = [
      on("room_list", (payload) => {
        dispatch({ type: "SET_ROOMS", rooms: (payload as RoomInfo[]) ?? [] });
      }),
      on("room_state", (payload) => {
        dispatch({ type: "ROOM_STATE", payload: payload as RoomStatePayload });
      }),
      on("tick", (payload) => {
        dispatch({ type: "TICK", payload: payload as TickPayload });
      }),
      on("elimination", (payload) => {
        dispatch({
          type: "ELIMINATION",
          payload: payload as EliminationPayload,
        });
      }),
      on("pulse_ack", (payload) => {
        dispatch({ type: "PULSE_ACK", payload: payload as PulseAckPayload });
      }),
    ];
    return () => unsubs.forEach((fn) => fn());
  }, [on]);

  useEffect(() => {
    if (connected) send("list_rooms");
  }, [connected, send]);

  const listRooms = useCallback(() => send("list_rooms"), [send]);
  const joinRoom = useCallback(
    (roomId: string) => send("join_room", { room_id: roomId }),
    [send],
  );
  const pulse = useCallback(() => send("pulse"), [send]);

  const leaveRoom = useCallback(() => {
    dispatch({ type: "LEAVE_ROOM" });
  }, []);

  return (
    <GameContext.Provider
      value={{ state, listRooms, joinRoom, pulse, leaveRoom, refreshPlayer }}
    >
      {children}
    </GameContext.Provider>
  );
}
