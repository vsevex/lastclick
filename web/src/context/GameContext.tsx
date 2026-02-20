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
import { getPlayer, NotFoundError } from "@/lib/api";
import type {
  RoomInfo,
  RoomStatePayload,
  TickPayload,
  EliminationPayload,
  PulseAckPayload,
  PlayerProfile,
} from "@/types/game";
import type {
  DebugCommand,
  EngineRoom,
  RoundState,
  PlayerState,
} from "@/engine/types";

interface GameState {
  player: PlayerProfile | null;
  playerLoading: boolean;
  playerError: boolean;
  playerNotFound: boolean;
  rooms: RoomInfo[];
  currentRoom: RoomStatePayload | null;
  marginHistory: number[];
  eliminated: number[];
  lastPulseAck: PulseAckPayload | null;
  isInRoom: boolean;
  selfEliminated: boolean;
  forfeited: boolean;
}

const initialState: GameState = {
  player: null,
  playerLoading: false,
  playerError: false,
  playerNotFound: false,
  rooms: [],
  currentRoom: null,
  marginHistory: [],
  eliminated: [],
  lastPulseAck: null,
  isInRoom: false,
  selfEliminated: false,
  forfeited: false,
};

type Action =
  | { type: "SET_PLAYER"; player: PlayerProfile }
  | { type: "PLAYER_LOADING" }
  | { type: "PLAYER_ERROR" }
  | { type: "PLAYER_NOT_FOUND" }
  | { type: "SET_ROOMS"; rooms: RoomInfo[] }
  | { type: "ROOM_STATE"; payload: RoomStatePayload }
  | { type: "TICK"; payload: TickPayload }
  | { type: "ELIMINATION"; payload: EliminationPayload }
  | { type: "PULSE_ACK"; payload: PulseAckPayload }
  | { type: "FORFEIT" }
  | { type: "CLEAR_ROOM" };

function reducer(state: GameState, action: Action): GameState {
  switch (action.type) {
    case "SET_PLAYER":
      return {
        ...state,
        player: action.player,
        playerLoading: false,
        playerError: false,
        playerNotFound: false,
      };

    case "PLAYER_LOADING":
      return {
        ...state,
        playerLoading: true,
        playerError: false,
        playerNotFound: false,
      };

    case "PLAYER_ERROR":
      return {
        ...state,
        playerLoading: false,
        playerError: true,
        playerNotFound: false,
      };

    case "PLAYER_NOT_FOUND":
      return {
        ...state,
        playerLoading: false,
        playerError: false,
        playerNotFound: true,
      };

    case "SET_ROOMS":
      return { ...state, rooms: action.rooms };

    case "ROOM_STATE": {
      const sameRoom = state.currentRoom?.room_id === action.payload.room_id;
      const prevState = state.currentRoom?.state;
      const isWaiting = action.payload.state === "waiting";
      const isCountdown = action.payload.state === "active";
      const chartReset = sameRoom && isCountdown && prevState === "waiting";
      let marginHistory: number[];
      if (chartReset) {
        marginHistory = [action.payload.margin_ratio];
      } else if (sameRoom && isWaiting) {
        const next = [...state.marginHistory, action.payload.margin_ratio];
        if (next.length > 120) next.shift();
        marginHistory = next;
      } else if (sameRoom) {
        marginHistory = state.marginHistory;
      } else {
        marginHistory = [action.payload.margin_ratio];
      }
      return {
        ...state,
        currentRoom: action.payload,
        isInRoom: true,
        selfEliminated: false,
        forfeited: false,
        marginHistory,
        eliminated: sameRoom ? state.eliminated : [],
      };
    }

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
        selfEliminated:
          state.selfEliminated || action.payload.player_id === state.player?.ID,
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

    case "FORFEIT":
      return { ...state, forfeited: true };

    case "CLEAR_ROOM":
      return {
        ...state,
        currentRoom: null,
        isInRoom: false,
        marginHistory: [],
        eliminated: [],
        lastPulseAck: null,
        selfEliminated: false,
        forfeited: false,
      };

    default:
      return state;
  }
}

export interface EngineExtras {
  isPrototype: true;
  debugCommand: (cmd: DebugCommand) => void;
  resetRound: () => void;
  /** Results → next round delay (ms). 10–20s for fast loop. */
  roundCompleteDelayMs: number;
  engineRoom: EngineRoom | null;
  roundState: RoundState | null;
  playerState: PlayerState | null;
  payoutInfo: { amount: number; rank: number } | null;
  shardCredit: number | null;
  simulateDisconnect: () => void;
  simulateReconnect: () => void;
}

export interface GameContextType {
  state: GameState;
  listRooms: () => void;
  joinRoom: (roomId: string) => void;
  pulse: () => void;
  forfeit: () => void;
  clearRoom: () => void;
  refreshPlayer: () => void;
  engine?: EngineExtras;
}

export const GameContext = createContext<GameContextType | undefined>(
  undefined,
);

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
    dispatch({ type: "PLAYER_LOADING" });
    getPlayer(userId)
      .then((p) => dispatch({ type: "SET_PLAYER", player: p }))
      .catch((e) => {
        if (e instanceof NotFoundError) {
          dispatch({ type: "PLAYER_NOT_FOUND" });
        } else {
          dispatch({ type: "PLAYER_ERROR" });
        }
      });
  }, [userId]);

  useEffect(() => {
    if (userId) refreshPlayer();
  }, [userId, refreshPlayer]);

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
  }, [on, connected]);

  // On reconnect, request sync so server restores room state if still alive
  useEffect(() => {
    if (connected) {
      send("sync");
      send("list_rooms");
    }
  }, [connected, send]);

  const listRooms = useCallback(() => send("list_rooms"), [send]);
  const joinRoom = useCallback(
    (roomId: string) => send("join_room", { room_id: roomId }),
    [send],
  );
  const pulse = useCallback(() => send("pulse"), [send]);

  const forfeit = useCallback(() => {
    send("forfeit");
    dispatch({ type: "FORFEIT" });
  }, [send]);

  const clearRoom = useCallback(() => {
    dispatch({ type: "CLEAR_ROOM" });
  }, []);

  return (
    <GameContext.Provider
      value={{
        state,
        listRooms,
        joinRoom,
        pulse,
        forfeit,
        clearRoom,
        refreshPlayer,
      }}
    >
      {children}
    </GameContext.Provider>
  );
}
