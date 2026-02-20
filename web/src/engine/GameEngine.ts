import { TIERS } from "@/types/game";
import type {
  RoomInfo,
  RoomStatePayload,
  TickPayload,
  EliminationPayload,
  PulseAckPayload,
} from "@/types/game";
import {
  RoundState,
  PlayerState,
  DEFAULT_ENGINE_CONFIG,
  type EngineRoom,
  type EnginePlayer,
  type GameEvent,
  type DebugCommand,
  type EngineConfig,
} from "./types";
import { VolatilityModel } from "./VolatilityModel";

type Listener = (data: unknown) => void;

export class GameEngine {
  private rooms = new Map<string, EngineRoom>();
  private volatilityModels = new Map<string, VolatilityModel>();
  private playerRoomMap = new Map<number, string>();
  private localPlayerId: number;
  private stars: number;
  private shards: number;
  private tickHandle: number | null = null;
  private listeners = new Map<string, Set<Listener>>();
  private config: EngineConfig;
  private debugOverrides = {
    lagMs: 0,
    pulseWindowMs: null as number | null,
  };
  private botTimers: number[] = [];
  private lastTickTime = 0;
  private roundCompleteResetTimeouts = new Map<string, number>();

  constructor(
    localPlayerId: number,
    initialStars = 500,
    config?: Partial<EngineConfig>,
  ) {
    this.localPlayerId = localPlayerId;
    this.stars = initialStars;
    this.shards = 0;
    this.config = { ...DEFAULT_ENGINE_CONFIG, ...config };
    this.seedRooms();
    this.startTick();
  }

  /* ── public api ────────────────────────────────────────── */

  getLocalPlayer() {
    return { id: this.localPlayerId, stars: this.stars, shards: this.shards };
  }

  getConfig(): EngineConfig {
    return this.config;
  }

  getRooms(): RoomInfo[] {
    const out: RoomInfo[] = [];
    for (const room of this.rooms.values()) out.push(this.toRoomInfo(room));
    return out;
  }

  getRoom(roomId: string): EngineRoom | null {
    return this.rooms.get(roomId) ?? null;
  }

  getPlayerInRoom(roomId: string, playerId: number): EnginePlayer | null {
    return this.rooms.get(roomId)?.players.get(playerId) ?? null;
  }

  getLocalRoomId(): string | null {
    return this.playerRoomMap.get(this.localPlayerId) ?? null;
  }

  /** Reset round and transition to WAITING (next round). Room persists; keeps players together. */
  resetRound(roomId: string) {
    const room = this.rooms.get(roomId);
    if (!room) return;

    const t = this.roundCompleteResetTimeouts.get(roomId);
    if (t != null) {
      clearTimeout(t);
      this.roundCompleteResetTimeouts.delete(roomId);
    }

    // Only reset from ROUND_COMPLETE (or LIQUIDATED already transitioned there)
    if (room.roundState !== RoundState.ROUND_COMPLETE) return;

    // Remove players who left the room; they don't stay for next round
    for (const [id, p] of [...room.players.entries()]) {
      if (p.state === PlayerState.LEFT) {
        room.players.delete(id);
        this.playerRoomMap.delete(id);
      }
    }

    // Reset remaining players for next round
    const tier = TIERS[room.tier];
    for (const player of room.players.values()) {
      player.state = PlayerState.JOINED;
      player.lastPulseTimestamp = 0;
      player.isAlive = true;
      player.starsSpent = 0;
      player.timeSurvived = 0;
      player.eliminatedAt = null;
      player.payout = 0;
      player.shardsEarned = 0;
    }

    // Reset room round state; room identity (id, type, tier) unchanged
    room.pool = 0;
    room.roundPaid = false;
    room.winnerId = null;
    room.top3 = [];
    room.timerMs = 0;
    room.marginRatio = 0;
    room.volatilityMul = 1;
    room.countdownMs = 0;
    room.survivalStartedAt = null;

    this.volatilityModels.delete(room.id);
    room.roundState = RoundState.WAITING_FOR_PLAYERS;

    this.emitRoomState(room);
    this.emit("room_list", this.getRooms());
    this.emit("round_transition", {
      roomId: room.id,
      from: RoundState.ROUND_COMPLETE,
      to: RoundState.WAITING_FOR_PLAYERS,
    });

    if (tier && room.players.size >= tier.minPlayers) {
      this.transitionRoom(room, RoundState.COUNTDOWN);
    }
  }

  dispatch(event: GameEvent) {
    if (this.debugOverrides.lagMs > 0) {
      setTimeout(() => this.processEvent(event), this.debugOverrides.lagMs);
    } else {
      this.processEvent(event);
    }
  }

  debugCommand(cmd: DebugCommand) {
    switch (cmd.type) {
      case "FORCE_LIQUIDATION":
        for (const room of this.rooms.values()) {
          if (room.roundState === RoundState.SURVIVAL_PHASE) {
            const vm = this.volatilityModels.get(room.id);
            if (vm) vm.forceMargin(1.0);
            room.marginRatio = 1.0;
            this.transitionRoom(room, RoundState.LIQUIDATED);
          }
        }
        break;

      case "FORCE_DISCONNECT":
        if (cmd.playerId != null) {
          const rid = this.playerRoomMap.get(cmd.playerId);
          if (rid) {
            this.processEvent({
              type: "DISCONNECT",
              playerId: cmd.playerId,
              roomId: rid,
              timestamp: Date.now(),
            });
          }
        }
        break;

      case "FORCE_TOP3":
        for (const room of this.rooms.values()) {
          if (room.roundState === RoundState.SURVIVAL_PHASE) {
            this.forceToTop3(room);
          }
        }
        break;

      case "INJECT_LAG":
        this.debugOverrides.lagMs = cmd.value ?? 0;
        break;

      case "MASS_ELIMINATION": {
        const count = cmd.value ?? 5;
        for (const room of this.rooms.values()) {
          if (room.roundState === RoundState.SURVIVAL_PHASE) {
            this.massEliminate(room, count);
          }
        }
        break;
      }

      case "SET_PULSE_WINDOW":
        this.debugOverrides.pulseWindowMs =
          cmd.value != null ? cmd.value * 1000 : null;
        break;

      case "FORCE_COUNTDOWN_END":
        for (const room of this.rooms.values()) {
          if (room.roundState === RoundState.COUNTDOWN) {
            room.countdownMs = 0;
          }
        }
        break;
    }
  }

  destroy() {
    if (this.tickHandle != null) {
      clearInterval(this.tickHandle);
      this.tickHandle = null;
    }
    for (const t of this.botTimers) clearTimeout(t);
    this.botTimers = [];
    for (const t of this.roundCompleteResetTimeouts.values()) clearTimeout(t);
    this.roundCompleteResetTimeouts.clear();
    this.listeners.clear();
  }

  on(event: string, listener: Listener): () => void {
    if (!this.listeners.has(event)) this.listeners.set(event, new Set());
    this.listeners.get(event)!.add(listener);
    return () => this.listeners.get(event)?.delete(listener);
  }

  /* ── event processing ──────────────────────────────────── */

  private processEvent(event: GameEvent) {
    switch (event.type) {
      case "JOIN_ROOM":
        this.handleJoinRoom(event);
        break;
      case "PULSE":
        this.handlePulse(event);
        break;
      case "LEAVE_ROOM":
        this.handleLeaveRoom(event);
        break;
      case "DISCONNECT":
        this.handleDisconnect(event);
        break;
      case "RECONNECT":
        this.handleReconnect(event);
        break;
    }
  }

  private handleJoinRoom(event: GameEvent) {
    const room = this.rooms.get(event.roomId);
    if (!room) return;
    // No late join: only allow join while waiting. No re-entry after countdown/survival.
    if (room.roundState !== RoundState.WAITING_FOR_PLAYERS) return;

    const tier = TIERS[room.tier];
    if (!tier) return;

    const isLocal = event.playerId === this.localPlayerId;

    if (isLocal) {
      if (this.stars < tier.entryCost) {
        this.emit("error", { message: "Insufficient stars" });
        return;
      }
      // Entry deducted when countdown starts, not on join (stops volatility-scout)
    }

    const player: EnginePlayer = {
      id: event.playerId,
      username: isLocal ? "You" : `Bot_${event.playerId}`,
      state: PlayerState.JOINED,
      lastPulseTimestamp: event.timestamp,
      isAlive: true,
      starsSpent: 0, // set when countdown starts
      timeSurvived: 0,
      joinedAt: event.timestamp,
      eliminatedAt: null,
      shardsEarned: 0,
      payout: 0,
      isBot: !isLocal,
      botSkill: isLocal ? undefined : 0.3 + Math.random() * 0.6,
    };

    room.players.set(event.playerId, player);
    // pool updated when countdown starts (entry deduction before survival)
    this.playerRoomMap.set(event.playerId, room.id);

    if (isLocal) {
      this.emitRoomState(room);
      this.spawnBots(room);
    }

    this.emit("room_list", this.getRooms());
    // balance emitted when countdown starts (on deduction)

    if (room.players.size >= tier.minPlayers) {
      this.transitionRoom(room, RoundState.COUNTDOWN);
    }
  }

  private handlePulse(event: GameEvent) {
    const roomId = this.playerRoomMap.get(event.playerId);
    if (!roomId) return;
    const room = this.rooms.get(roomId);
    if (!room || room.roundState !== RoundState.SURVIVAL_PHASE) return;

    const player = room.players.get(event.playerId);
    if (!player || player.state !== PlayerState.ACTIVE) return;

    const tier = TIERS[room.tier];
    const pulseWindowMs =
      this.debugOverrides.pulseWindowMs ?? (tier?.pulseWindowSec ?? 5) * 1000;

    player.lastPulseTimestamp = event.timestamp;

    const ack: PulseAckPayload = {
      player_id: event.playerId,
      extension_ms: pulseWindowMs,
      timer_ms: room.timerMs,
      server_time_ms: event.timestamp,
    };

    this.emit("pulse_ack", ack);
  }

  private handleLeaveRoom(event: GameEvent) {
    const roomId = this.playerRoomMap.get(event.playerId);
    if (!roomId) return;
    const room = this.rooms.get(roomId);
    if (!room) return;

    const player = room.players.get(event.playerId);
    if (!player) return;

    if (player.state === PlayerState.ACTIVE) {
      this.eliminatePlayer(room, event.playerId);
      player.state = PlayerState.LEFT;
    } else if (player.state === PlayerState.JOINED) {
      player.state = PlayerState.LEFT;
    }

    this.emitRoomState(room);
    this.checkWinCondition(room);
  }

  private handleDisconnect(event: GameEvent) {
    const roomId = this.playerRoomMap.get(event.playerId);
    if (!roomId) return;
    const room = this.rooms.get(roomId);
    if (!room) return;

    const player = room.players.get(event.playerId);
    if (!player) return;

    if (
      player.state === PlayerState.ACTIVE ||
      player.state === PlayerState.JOINED
    ) {
      player.state = PlayerState.DISCONNECTED;
    }

    this.emit("player_state_change", {
      playerId: event.playerId,
      state: player.state,
    });
    this.emitRoomState(room);
  }

  private handleReconnect(event: GameEvent) {
    const roomId = this.playerRoomMap.get(event.playerId);
    if (!roomId) return;
    const room = this.rooms.get(roomId);
    if (!room) return;

    const player = room.players.get(event.playerId);
    if (!player) return;

    if (player.state === PlayerState.DISCONNECTED) {
      const tier = TIERS[room.tier];
      const pulseWindowMs =
        this.debugOverrides.pulseWindowMs ?? (tier?.pulseWindowSec ?? 5) * 1000;

      if (
        room.roundState === RoundState.SURVIVAL_PHASE &&
        Date.now() - player.lastPulseTimestamp > pulseWindowMs
      ) {
        this.eliminatePlayer(room, event.playerId);
      } else if (room.roundState === RoundState.SURVIVAL_PHASE) {
        player.state = PlayerState.ACTIVE;
      }
    }

    this.checkWinCondition(room);
    this.emitRoomState(room);
    this.emit("reconnect_state", {
      playerId: event.playerId,
      playerState: player.state,
      roundState: room.roundState,
    });
  }

  /* ── tick engine ───────────────────────────────────────── */

  private startTick() {
    this.lastTickTime = Date.now();
    this.tickHandle = window.setInterval(
      () => this.tick(),
      this.config.tickIntervalMs,
    );
  }

  private tick() {
    const now = Date.now();
    const dt = now - this.lastTickTime;
    this.lastTickTime = now;

    for (const room of this.rooms.values()) {
      this.tickRoom(room, dt, now);
    }
  }

  private tickRoom(room: EngineRoom, dtMs: number, now: number) {
    switch (room.roundState) {
      case RoundState.COUNTDOWN:
        room.countdownMs -= dtMs;
        if (room.countdownMs <= 0) {
          this.transitionRoom(room, RoundState.SURVIVAL_PHASE);
        } else {
          this.emitRoomState(room);
        }
        break;

      case RoundState.SURVIVAL_PHASE: {
        const volModel = this.volatilityModels.get(room.id);
        if (volModel) {
          const { marginRatio, volatilityMul } = volModel.tick(dtMs);
          room.marginRatio = marginRatio;
          room.volatilityMul = volatilityMul;
        }

        room.timerMs = Math.max(0, room.timerMs - dtMs);

        if (room.marginRatio >= 1.0) {
          this.transitionRoom(room, RoundState.LIQUIDATED);
          return;
        }

        const tier = TIERS[room.tier];
        const pulseWindowMs =
          this.debugOverrides.pulseWindowMs ??
          (tier?.pulseWindowSec ?? 5) * 1000;

        for (const player of room.players.values()) {
          if (
            player.state === PlayerState.ACTIVE ||
            player.state === PlayerState.DISCONNECTED
          ) {
            if (now - player.lastPulseTimestamp > pulseWindowMs) {
              this.eliminatePlayer(room, player.id);
            }
          }
        }

        this.tickBots(room, now);
        this.checkWinCondition(room);

        if (room.roundState === RoundState.SURVIVAL_PHASE) {
          const tick: TickPayload = {
            timer_ms: room.timerMs,
            margin_ratio: room.marginRatio,
            volatility_mul: room.volatilityMul,
            alive: this.countAlive(room),
          };
          this.emit("tick", tick);
        }
        break;
      }

      default:
        break;
    }
  }

  /* ── state transitions ─────────────────────────────────── */

  private transitionRoom(room: EngineRoom, newState: RoundState) {
    const oldState = room.roundState;
    room.roundState = newState;

    switch (newState) {
      case RoundState.COUNTDOWN: {
        room.countdownMs = this.config.countdownDurationMs;
        // Entry deduction BEFORE survival: charge everyone now. No late join after this.
        const tier = TIERS[room.tier];
        if (tier) {
          for (const player of room.players.values()) {
            player.starsSpent = tier.entryCost;
            room.pool += tier.entryCost;
            if (player.id === this.localPlayerId) {
              this.stars -= tier.entryCost;
            }
          }
          this.emit("balance", { stars: this.stars, shards: this.shards });
        }
        break;
      }

      case RoundState.SURVIVAL_PHASE: {
        room.survivalStartedAt = Date.now();
        for (const player of room.players.values()) {
          if (player.state === PlayerState.JOINED) {
            player.state = PlayerState.ACTIVE;
            player.lastPulseTimestamp = Date.now();
          }
        }
        this.volatilityModels.set(room.id, new VolatilityModel(room.tier));
        const tier = TIERS[room.tier];
        room.timerMs = (tier?.survivalTimeSec ?? 120) * 1000;
        break;
      }

      case RoundState.LIQUIDATED: {
        const activePlayers = [...room.players.values()].filter(
          (p) => p.state === PlayerState.ACTIVE,
        );
        for (const player of activePlayers) {
          player.state = PlayerState.ELIMINATED;
          player.isAlive = false;
          player.eliminatedAt = Date.now();
          player.timeSurvived = room.survivalStartedAt
            ? Date.now() - room.survivalStartedAt
            : 0;
        }
        this.assignRankings(room);
        this.transitionRoom(room, RoundState.ROUND_COMPLETE);
        return;
      }

      case RoundState.ROUND_COMPLETE: {
        this.distributePayout(room);
        this.creditShards(room);
        // Room persists. Schedule transition to WAITING (next round). Short delay = fast loop.
        const delayMs = this.config.roundCompleteDelayMs;
        const t = window.setTimeout(() => {
          this.roundCompleteResetTimeouts.delete(room.id);
          this.resetRound(room.id);
        }, delayMs) as unknown as number;
        this.roundCompleteResetTimeouts.set(room.id, t);
        break;
      }
    }

    this.emitRoomState(room);
    this.emit("room_list", this.getRooms());
    this.emit("round_transition", {
      roomId: room.id,
      from: oldState,
      to: newState,
    });
  }

  private eliminatePlayer(room: EngineRoom, playerId: number) {
    const player = room.players.get(playerId);
    if (!player || !player.isAlive) return;

    player.state = PlayerState.ELIMINATED;
    player.isAlive = false;
    player.eliminatedAt = Date.now();
    player.timeSurvived = room.survivalStartedAt
      ? Date.now() - room.survivalStartedAt
      : 0;

    const alive = this.countAlive(room);
    const elimination: EliminationPayload = {
      player_id: playerId,
      alive,
    };
    this.emit("elimination", elimination);
  }

  private checkWinCondition(room: EngineRoom) {
    if (room.roundState !== RoundState.SURVIVAL_PHASE) return;

    const alive = this.countAlive(room);
    const total = room.players.size;
    const roundOver = (alive <= 3 && total > 3) || alive === 1 || alive === 0;

    if (roundOver) {
      if (alive >= 1) this.finalizeActiveSurvivors(room);
      this.assignRankings(room);
      this.transitionRoom(room, RoundState.ROUND_COMPLETE);
    }
  }

  private finalizeActiveSurvivors(room: EngineRoom) {
    for (const p of room.players.values()) {
      if (p.state === PlayerState.ACTIVE) {
        p.timeSurvived = room.survivalStartedAt
          ? Date.now() - room.survivalStartedAt
          : 0;
      }
    }
  }

  private assignRankings(room: EngineRoom) {
    const active = [...room.players.values()]
      .filter((p) => p.state === PlayerState.ACTIVE)
      .sort((a, b) => b.timeSurvived - a.timeSurvived);

    const eliminated = [...room.players.values()]
      .filter(
        (p) =>
          p.state === PlayerState.ELIMINATED ||
          p.state === PlayerState.DISCONNECTED,
      )
      .sort((a, b) => b.timeSurvived - a.timeSurvived);

    const ranked = [...active, ...eliminated];
    room.top3 = [];

    if (ranked.length >= 1) {
      ranked[0].state = PlayerState.WINNER;
      room.winnerId = ranked[0].id;
      room.top3.push(ranked[0].id);
    }
    if (ranked.length >= 2) {
      ranked[1].state = PlayerState.TOP3;
      room.top3.push(ranked[1].id);
    }
    if (ranked.length >= 3) {
      ranked[2].state = PlayerState.TOP3;
      room.top3.push(ranked[2].id);
    }
  }

  private distributePayout(room: EngineRoom) {
    if (room.roundPaid) return;
    room.roundPaid = true;

    const pool = room.pool;
    const splits = [0.6, 0.25, 0.15];

    for (let i = 0; i < room.top3.length && i < 3; i++) {
      const player = room.players.get(room.top3[i]);
      if (!player) continue;

      player.payout = Math.floor(pool * splits[i]);
      if (player.id === this.localPlayerId) {
        this.stars += player.payout;
      }

      this.emit("payout", {
        playerId: player.id,
        amount: player.payout,
        rank: i + 1,
      });
    }

    this.emit("balance", { stars: this.stars, shards: this.shards });
  }

  private creditShards(room: EngineRoom) {
    const tier = TIERS[room.tier];
    if (!tier) return;

    for (const player of room.players.values()) {
      if (
        player.state === PlayerState.ELIMINATED ||
        player.state === PlayerState.LEFT ||
        player.state === PlayerState.DISCONNECTED
      ) {
        player.shardsEarned = Math.floor(
          tier.entryCost * this.config.shardRate,
        );
        if (player.id === this.localPlayerId) {
          this.shards += player.shardsEarned;
        }
        this.emit("shard_credit", {
          playerId: player.id,
          shards: player.shardsEarned,
        });
      }
    }

    this.emit("balance", { stars: this.stars, shards: this.shards });
  }

  /* ── bot ai ────────────────────────────────────────────── */

  private spawnBots(room: EngineRoom) {
    const tier = TIERS[room.tier];
    if (!tier) return;

    const botCount = Math.min(
      this.config.botCount,
      tier.maxPlayers - room.players.size,
    );

    let botIdCounter = 0;
    for (let i = 0; i < botCount; i++) {
      const delay = (i + 1) * this.config.botJoinDelayMs + Math.random() * 500;
      const timer = window.setTimeout(() => {
        if (room.roundState !== RoundState.WAITING_FOR_PLAYERS) return;
        let botId: number;
        do {
          botId = 10000 + botIdCounter++;
        } while (room.players.has(botId));
        this.processEvent({
          type: "JOIN_ROOM",
          playerId: botId,
          roomId: room.id,
          timestamp: Date.now(),
        });
      }, delay);
      this.botTimers.push(timer);
    }
  }

  private tickBots(room: EngineRoom, now: number) {
    const tier = TIERS[room.tier];
    const pulseWindowMs =
      this.debugOverrides.pulseWindowMs ?? (tier?.pulseWindowSec ?? 5) * 1000;

    for (const player of room.players.values()) {
      if (!player.isBot || player.state !== PlayerState.ACTIVE) continue;

      const elapsed = now - player.lastPulseTimestamp;
      const threshold = pulseWindowMs * (player.botSkill ?? 0.5);

      if (elapsed > threshold) {
        const missChance = 1 - (player.botSkill ?? 0.5);
        if (Math.random() > missChance * 0.15) {
          player.lastPulseTimestamp = now;
        }
      }
    }
  }

  /* ── debug helpers ─────────────────────────────────────── */

  private forceToTop3(room: EngineRoom) {
    const bots = [...room.players.values()].filter(
      (p) => p.state === PlayerState.ACTIVE && p.isBot,
    );
    const target = Math.max(0, this.countAlive(room) - 3);
    let eliminated = 0;
    for (const bot of bots) {
      if (eliminated >= target) break;
      this.eliminatePlayer(room, bot.id);
      eliminated++;
    }
  }

  private massEliminate(room: EngineRoom, count: number) {
    const bots = [...room.players.values()].filter(
      (p) => p.state === PlayerState.ACTIVE && p.isBot,
    );
    for (let i = 0; i < Math.min(count, bots.length); i++) {
      this.eliminatePlayer(room, bots[i].id);
    }
  }

  /* ── seeding ───────────────────────────────────────────── */

  private seedRooms() {
    const types: Array<"alpha" | "blitz"> = ["alpha", "blitz"];
    for (const type of types) {
      for (let tier = 1; tier <= 3; tier++) {
        const id = `sim_${type}_t${tier}_${Date.now().toString(36)}`;
        const room: EngineRoom = {
          id,
          type,
          tier,
          roundState: RoundState.WAITING_FOR_PLAYERS,
          players: new Map(),
          pool: 0,
          timerMs: 0,
          marginRatio: 0,
          volatilityMul: 1,
          winnerId: null,
          top3: [],
          roundPaid: false,
          countdownMs: 0,
          survivalStartedAt: null,
          createdAt: Date.now(),
        };
        this.rooms.set(id, room);
      }
    }
  }

  /* ── emit helpers ──────────────────────────────────────── */

  private toRoomInfo(room: EngineRoom): RoomInfo {
    return {
      id: room.id,
      type: room.type,
      tier: room.tier,
      state: this.mapState(room.roundState),
      players: room.players.size,
      pool: room.pool,
    };
  }

  private mapState(
    rs: RoundState,
  ): "waiting" | "active" | "survival" | "finished" {
    switch (rs) {
      case RoundState.WAITING_FOR_PLAYERS:
        return "waiting";
      case RoundState.COUNTDOWN:
        return "active";
      case RoundState.SURVIVAL_PHASE:
      case RoundState.LIQUIDATED:
        return "survival";
      case RoundState.ROUND_COMPLETE:
        return "finished";
    }
  }

  private emitRoomState(room: EngineRoom) {
    const payload: RoomStatePayload = {
      room_id: room.id,
      state: this.mapState(room.roundState),
      type: room.type,
      tier: room.tier,
      pool: room.pool,
      alive: this.countAlive(room),
      total: room.players.size,
      timer_ms:
        room.roundState === RoundState.COUNTDOWN
          ? room.countdownMs
          : room.timerMs,
      margin_ratio: room.marginRatio,
      volatility_mul: room.volatilityMul,
      winner_id: room.winnerId ?? 0,
    };
    this.emit("room_state", payload);
  }

  private countAlive(room: EngineRoom): number {
    let n = 0;
    for (const p of room.players.values()) {
      if (p.state === PlayerState.ACTIVE || p.state === PlayerState.JOINED) n++;
    }
    return n;
  }

  private emit(event: string, data: unknown) {
    this.listeners.get(event)?.forEach((fn) => fn(data));
  }
}
