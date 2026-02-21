export enum RoundState {
  WAITING_FOR_PLAYERS = "WAITING_FOR_PLAYERS",
  COUNTDOWN = "COUNTDOWN",
  SURVIVAL_PHASE = "SURVIVAL_PHASE",
  LIQUIDATED = "LIQUIDATED",
  ROUND_COMPLETE = "ROUND_COMPLETE",
}

export enum PlayerState {
  JOINED = "JOINED",
  ACTIVE = "ACTIVE",
  ELIMINATED = "ELIMINATED",
  TOP3 = "TOP3",
  WINNER = "WINNER",
  LEFT = "LEFT",
  DISCONNECTED = "DISCONNECTED",
}

export interface EnginePlayer {
  id: number;
  username: string;
  state: PlayerState;
  lastPulseTimestamp: number;
  isAlive: boolean;
  starsSpent: number;
  timeSurvived: number;
  joinedAt: number;
  eliminatedAt: number | null;
  shardsEarned: number;
  payout: number;
  isBot: boolean;
  botSkill?: number;
  /** True if player left voluntarily during survival (forfeit). Still ELIMINATED for ranking; no re-entry same round. */
  voluntaryExit?: boolean;
}

export interface EngineRoom {
  id: string;
  type: "alpha" | "blitz";
  tier: number;
  roundState: RoundState;
  players: Map<number, EnginePlayer>;
  pool: number;
  timerMs: number;
  marginRatio: number;
  volatilityMul: number;
  winnerId: number | null;
  top3: number[];
  roundPaid: boolean;
  countdownMs: number;
  survivalStartedAt: number | null;
  createdAt: number;
}

export type GameEventType =
  | "JOIN_ROOM"
  | "PULSE"
  | "LEAVE_ROOM"
  | "DISCONNECT"
  | "RECONNECT";

export interface GameEvent {
  type: GameEventType;
  playerId: number;
  roomId: string;
  timestamp: number;
}

export interface DebugCommand {
  type:
    | "FORCE_LIQUIDATION"
    | "FORCE_DISCONNECT"
    | "FORCE_TOP3"
    | "INJECT_LAG"
    | "MASS_ELIMINATION"
    | "SET_PULSE_WINDOW"
    | "FORCE_COUNTDOWN_END";
  playerId?: number;
  value?: number;
}

export interface EngineConfig {
  tickIntervalMs: number;
  countdownDurationMs: number;
  /** Results screen → auto next round. 10–20s for fast loop / retention. */
  roundCompleteDelayMs: number;
  shardRate: number;
  botCount: number;
  botJoinDelayMs: number;
}

export const DEFAULT_ENGINE_CONFIG: EngineConfig = {
  tickIntervalMs: 100,
  countdownDurationMs: 5000,
  roundCompleteDelayMs: 15_000,
  shardRate: 0.5,
  botCount: 8,
  botJoinDelayMs: 500,
};

export interface EngineSnapshot {
  roundState: RoundState | null;
  playerState: PlayerState | null;
  engineRoom: EngineRoom | null;
  localStars: number;
  localShards: number;
  payoutInfo: { amount: number; rank: number } | null;
  shardCredit: number | null;
}
