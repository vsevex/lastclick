// ===== Room =====
export type RoomType = "alpha" | "blitz";
export type RoomState = "waiting" | "active" | "survival" | "finished";

export interface RoomInfo {
  id: string;
  type: RoomType;
  tier: number;
  state: RoomState;
  players: number;
  pool: number;
}

export interface RoomStatePayload {
  room_id: string;
  state: RoomState;
  type: RoomType;
  tier: number;
  pool: number;
  alive: number;
  total: number;
  timer_ms: number;
  margin_ratio: number;
  volatility_mul: number;
  winner_id: number;
}

export interface TickPayload {
  timer_ms: number;
  margin_ratio: number;
  volatility_mul: number;
  alive: number;
}

export interface EliminationPayload {
  player_id: number;
  alive: number;
}

export interface PulseAckPayload {
  player_id: number;
  extension_ms: number;
  timer_ms: number;
}

// ===== Player (from REST /api/player/{id}) =====
export interface PlayerProfile {
  ID: number;
  Username: string;
  Elo: number;
  LifetimeElo: number;
  EfficiencyAvg: number;
  StarsBalance: number;
  ShardsBalance: number;
  SquadID: string | null;
  PrestigeMult: number;
  CreatedAt: string;
}

// ===== Leaderboard =====
export interface LeaderboardEntry {
  PlayerID: number;
  Score: number;
  Rank: number;
}

// ===== Tier Config (mirrors backend room.Tiers) =====
export interface TierConfig {
  tier: number;
  entryCost: number;
  minPlayers: number;
  maxPlayers: number;
  pulseWindowSec: number;
  survivalTimeSec: number;
  prestigeMult: number;
}

export const TIERS: Record<number, TierConfig> = {
  1: {
    tier: 1,
    entryCost: 5,
    minPlayers: 3,
    maxPlayers: 20,
    pulseWindowSec: 5,
    survivalTimeSec: 120,
    prestigeMult: 1.0,
  },
  2: {
    tier: 2,
    entryCost: 20,
    minPlayers: 5,
    maxPlayers: 30,
    pulseWindowSec: 4,
    survivalTimeSec: 150,
    prestigeMult: 1.5,
  },
  3: {
    tier: 3,
    entryCost: 100,
    minPlayers: 5,
    maxPlayers: 50,
    pulseWindowSec: 3,
    survivalTimeSec: 180,
    prestigeMult: 2.0,
  },
};
