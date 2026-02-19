/**
 * Last Click - Game Type Definitions
 * Core interfaces for the game simulation, state management, and UI
 */

// ===== Room & Game Configuration =====
export interface Room {
  id: string;
  name: string;
  volatility: number; // 0.5 - 5.0 (percent per update cycle)
  playerCount: number;
  maxPlayers: number;
  minBuyIn: number;
  startingWhaleLong: number;
  description: string;
}

// ===== Player & Account =====
export interface PlayerProfile {
  id: string;
  username: string;
  avatar: string;
  tier: "Bronze" | "Silver" | "Gold" | "Platinum" | "Diamond";
  seasonalRank: number;
  totalSurvivalTime: number;
  bestEfficiency: number;
  totalGamesPlayed: number;
  shards: number;
  squadId?: string;
  squadName?: string;
  cosmetics: {
    avatar: string;
    nameplate: string;
    particleEffect: string;
  };
}

export interface PlayerState {
  id: string;
  username: string;
  avatar: string;
  stars: number;
  shards: number;
  efficiency: number;
  isAlive: boolean;
  survivalTime: number;
  leaderboardPosition: number;
}

// ===== Whale Position & Price =====
export interface WhalePosition {
  currentPrice: number;
  liquidationPrice: number;
  volatility: number; // Volatility multiplier for efficiency scoring
  priceHistory: number[]; // Last 100 prices for chart
  liquidationProximity: number; // Percentage distance from liquidation
}

// ===== Survival Phase =====
export interface SurvivalPhase {
  isActive: boolean;
  timeRemaining: number; // Seconds
  pulsesRequired: number; // Number of Pulse clicks needed
  pulseWindow: number; // Time window to click Pulse (seconds)
  survivorCount: number;
  players: PlayerState[];
  lastUpdateTime: number;
}

// ===== Game State =====
export interface GameState {
  roomId: string;
  currentRoom: Room;
  playerState: PlayerState;
  whalePosition: WhalePosition;
  survivalPhase: SurvivalPhase;
  leaderboard: PlayerState[];
  gameStatus: "waiting" | "active" | "finished";
  elapsedTime: number;
  roundStartTime: number;
}

// ===== Cosmetics & Store =====
export interface CosmeticItem {
  id: string;
  name: string;
  category: "avatar" | "nameplate" | "particle";
  price: number; // In shards
  rarity: "common" | "rare" | "epic" | "legendary";
  description: string;
}

export interface CosmeticStore {
  avatars: CosmeticItem[];
  nameplates: CosmeticItem[];
  particles: CosmeticItem[];
}

// ===== Game Engine Events =====
export interface GameEngineUpdate {
  timestamp: number;
  whalePosition: WhalePosition;
  liquidationProximity: number;
  survivalPhaseTriggered: boolean;
  playersEliminated: string[];
}

// ===== Leaderboard =====
export interface LeaderboardEntry extends PlayerState {
  squadName?: string;
  timedOut: boolean;
}
