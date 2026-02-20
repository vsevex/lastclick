import {
  useEffect,
  useReducer,
  useCallback,
  useRef,
  type ReactNode,
} from "react";
import { useTelegram } from "@/context/TelegramProvider";
import { GameContext, type GameContextType } from "@/context/GameContext";
import { GameEngine } from "@/engine/GameEngine";
import {
  RoundState,
  PlayerState,
  type EngineRoom,
  type DebugCommand,
} from "@/engine/types";
import type {
  RoomInfo,
  RoomStatePayload,
  TickPayload,
  EliminationPayload,
  PulseAckPayload,
  PlayerProfile,
} from "@/types/game";

interface EngineState {
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
  engineRoom: EngineRoom | null;
  roundState: RoundState | null;
  playerState: PlayerState | null;
  payoutInfo: { amount: number; rank: number } | null;
  shardCredit: number | null;
}

const initialState: EngineState = {
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
  engineRoom: null,
  roundState: null,
  playerState: null,
  payoutInfo: null,
  shardCredit: null,
};

type Action =
  | { type: "SET_PLAYER"; player: PlayerProfile }
  | { type: "SET_ROOMS"; rooms: RoomInfo[] }
  | { type: "ROOM_STATE"; payload: RoomStatePayload }
  | { type: "TICK"; payload: TickPayload }
  | { type: "ELIMINATION"; payload: EliminationPayload }
  | { type: "PULSE_ACK"; payload: PulseAckPayload }
  | { type: "FORFEIT" }
  | { type: "CLEAR_ROOM" }
  | { type: "BALANCE"; stars: number; shards: number }
  | { type: "ENGINE_ROOM"; room: EngineRoom | null }
  | { type: "ROUND_TRANSITION"; roundState: RoundState }
  | { type: "PLAYER_STATE"; playerState: PlayerState }
  | { type: "PAYOUT"; amount: number; rank: number }
  | { type: "SHARD_CREDIT"; shards: number };

function reducer(state: EngineState, action: Action): EngineState {
  switch (action.type) {
    case "SET_PLAYER":
      return {
        ...state,
        player: action.player,
        playerLoading: false,
        playerError: false,
        playerNotFound: false,
      };

    case "SET_ROOMS":
      return { ...state, rooms: action.rooms };

    case "ROOM_STATE":
      return {
        ...state,
        currentRoom: action.payload,
        isInRoom: true,
        selfEliminated: false,
        forfeited: false,
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
        currentRoom: { ...state.currentRoom, alive: action.payload.alive },
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
        engineRoom: null,
        roundState: null,
        playerState: null,
        payoutInfo: null,
        shardCredit: null,
      };

    case "BALANCE":
      if (!state.player) return state;
      return {
        ...state,
        player: {
          ...state.player,
          StarsBalance: action.stars,
          ShardsBalance: action.shards,
        },
      };

    case "ENGINE_ROOM":
      return {
        ...state,
        engineRoom: action.room,
        roundState: action.room?.roundState ?? null,
      };

    case "ROUND_TRANSITION":
      return { ...state, roundState: action.roundState };

    case "PLAYER_STATE":
      return { ...state, playerState: action.playerState };

    case "PAYOUT":
      return {
        ...state,
        payoutInfo: { amount: action.amount, rank: action.rank },
      };

    case "SHARD_CREDIT":
      return { ...state, shardCredit: action.shards };

    default:
      return state;
  }
}

export function EngineProvider({ children }: { children: ReactNode }) {
  const { userId, username } = useTelegram();
  const engineRef = useRef<GameEngine | null>(null);
  const [state, dispatch] = useReducer(reducer, initialState);

  useEffect(() => {
    const playerId = userId ?? 1;
    const engine = new GameEngine(playerId, 500);
    engineRef.current = engine;

    const mockPlayer: PlayerProfile = {
      ID: playerId,
      Username: username ?? `Player_${playerId}`,
      Elo: 1000,
      LifetimeElo: 1000,
      EfficiencyAvg: 50.0,
      StarsBalance: 500,
      ShardsBalance: 0,
      SquadID: null,
      PrestigeMult: 1.0,
      CreatedAt: new Date().toISOString(),
    };
    dispatch({ type: "SET_PLAYER", player: mockPlayer });

    dispatch({ type: "SET_ROOMS", rooms: engine.getRooms() });

    const unsubs = [
      engine.on("room_list", (data) => {
        dispatch({ type: "SET_ROOMS", rooms: data as RoomInfo[] });
      }),
      engine.on("room_state", (data) => {
        dispatch({ type: "ROOM_STATE", payload: data as RoomStatePayload });
      }),
      engine.on("tick", (data) => {
        dispatch({ type: "TICK", payload: data as TickPayload });
      }),
      engine.on("elimination", (data) => {
        dispatch({
          type: "ELIMINATION",
          payload: data as EliminationPayload,
        });
      }),
      engine.on("pulse_ack", (data) => {
        dispatch({ type: "PULSE_ACK", payload: data as PulseAckPayload });
      }),
      engine.on("balance", (data) => {
        const b = data as { stars: number; shards: number };
        dispatch({ type: "BALANCE", stars: b.stars, shards: b.shards });
      }),
      engine.on("round_transition", (data) => {
        const t = data as { roomId: string; to: RoundState };
        const isCurrentRoom = t.roomId === engineRef.current?.getLocalRoomId();
        if (isCurrentRoom) {
          dispatch({ type: "ROUND_TRANSITION", roundState: t.to });
        }

        if (isCurrentRoom) {
          const room = engine.getRoom(t.roomId);
          if (room) dispatch({ type: "ENGINE_ROOM", room });

          const lp = engine.getPlayerInRoom(t.roomId, userId ?? 1);
          if (lp) dispatch({ type: "PLAYER_STATE", playerState: lp.state });
        }
      }),
      engine.on("payout", (data) => {
        const p = data as { playerId: number; amount: number; rank: number };
        if (p.playerId === (userId ?? 1)) {
          dispatch({ type: "PAYOUT", amount: p.amount, rank: p.rank });
        }
      }),
      engine.on("shard_credit", (data) => {
        const s = data as { playerId: number; shards: number };
        if (s.playerId === (userId ?? 1)) {
          dispatch({ type: "SHARD_CREDIT", shards: s.shards });
        }
      }),
    ];

    return () => {
      unsubs.forEach((fn) => fn());
      engine.destroy();
      engineRef.current = null;
    };
  }, [userId, username]);

  const listRooms = useCallback(() => {
    if (!engineRef.current) return;
    dispatch({ type: "SET_ROOMS", rooms: engineRef.current.getRooms() });
  }, []);

  const joinRoom = useCallback(
    (roomId: string) => {
      engineRef.current?.dispatch({
        type: "JOIN_ROOM",
        playerId: userId ?? 1,
        roomId,
        timestamp: Date.now(),
      });
    },
    [userId],
  );

  const pulse = useCallback(() => {
    const rid = engineRef.current?.getLocalRoomId();
    if (!rid) return;
    engineRef.current?.dispatch({
      type: "PULSE",
      playerId: userId ?? 1,
      roomId: rid,
      timestamp: Date.now(),
    });
  }, [userId]);

  const forfeit = useCallback(() => {
    const rid = engineRef.current?.getLocalRoomId();
    if (!rid) return;
    engineRef.current?.dispatch({
      type: "LEAVE_ROOM",
      playerId: userId ?? 1,
      roomId: rid,
      timestamp: Date.now(),
    });
    dispatch({ type: "FORFEIT" });
  }, [userId]);

  const clearRoom = useCallback(() => {
    dispatch({ type: "CLEAR_ROOM" });
  }, []);

  const refreshPlayer = useCallback(() => {
    if (!engineRef.current) return;
    const lp = engineRef.current.getLocalPlayer();
    dispatch({ type: "BALANCE", stars: lp.stars, shards: lp.shards });
  }, []);

  const debugCommand = useCallback((cmd: DebugCommand) => {
    engineRef.current?.debugCommand(cmd);
  }, []);

  const resetRound = useCallback(() => {
    const rid = engineRef.current?.getLocalRoomId();
    if (rid) engineRef.current?.resetRound(rid);
  }, []);

  const simulateDisconnect = useCallback(() => {
    const rid = engineRef.current?.getLocalRoomId();
    if (!rid) return;
    engineRef.current?.dispatch({
      type: "DISCONNECT",
      playerId: userId ?? 1,
      roomId: rid,
      timestamp: Date.now(),
    });
  }, [userId]);

  const simulateReconnect = useCallback(() => {
    const rid = engineRef.current?.getLocalRoomId();
    if (!rid) return;
    engineRef.current?.dispatch({
      type: "RECONNECT",
      playerId: userId ?? 1,
      roomId: rid,
      timestamp: Date.now(),
    });
  }, [userId]);

  const value: GameContextType = {
    state,
    listRooms,
    joinRoom,
    pulse,
    forfeit,
    clearRoom,
    refreshPlayer,
    engine: {
      isPrototype: true,
      debugCommand,
      resetRound,
      roundCompleteDelayMs:
        engineRef.current?.getConfig?.()?.roundCompleteDelayMs ?? 15_000,
      engineRoom: state.engineRoom,
      roundState: state.roundState,
      playerState: state.playerState,
      payoutInfo: state.payoutInfo,
      shardCredit: state.shardCredit,
      simulateDisconnect,
      simulateReconnect,
    },
  };

  return <GameContext.Provider value={value}>{children}</GameContext.Provider>;
}
