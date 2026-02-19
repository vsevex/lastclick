# Last Click

Last Click transforms high-leverage crypto liquidation events into a real-time, communal survival game inside Telegram. Players do not trade. They compete to outlast others during simulated or live liquidation scenarios, managing timing, risk, and efficiency under pressure.

The system combines **Tontine** mechanics, flash-auction escalation, DeFi liquidation theatre, and squad-based competition into a high-frequency, monetized survival engine.

## Core Concept

Last Click is a real-time survival market.

Players purchase **Life Shares** and enter a room tracking a leveraged “Whale Position.” When the position approaches liquidation, a Survival Phase begins. Players must actively reinforce their position to remain alive.

The last surviving player before liquidation wins the pool (minus protocol fees).

## Game Architecture

### The Pacing Engine - Hybrid Market Pulse

To prevent market drought and liquidity fragility, the system uses a hybrid model.

**Alpha Rooms (Live Core)**:

- Track real high-leverage whale positions on TON.
- Lower manipulation tolerance.
- Higher Prestige Multiplier.
- Designed for high-conviction players.

**Blitz Rooms (Synthetic Pulse)**:

- Use statistically modeled volatility simulations based on historic datasets.
- Controlled pacing to maintain engagement.
- Near-liquidation hover windows tuned for optimal tension.
- Transparent labeling in simulated environments.

Rooms are clearly marked. Prestige multipliers reward participation in live rooms.

### Core Game Loop

1. Players enter the room by purchasing Life Shares (Stars).
2. Whale position approaches Maintenance Margin.
3. Survival Phase begins (e.g, 60 seconds).
4. Every few seconds, players must "Pulse" (1 Star cost) to extend the timer.
5. Each pulse extends the global timer slightly.
6. If a player misses the click window -> eliminated.
7. When liquidation occurs -> last surviving player wins the pool.

Session duration target: 2-4 minutes per round.

## Strategic Depth

### Survival Efficiency Score

To eliminate pure finger stamina gameplay, the primary competitive metric is:

Efficiency = (Time Survived × Volatility Multiplier) ÷ Stars Spent

Volatility Multiplier increases as liquidation proximity tightens.

**Strategic implications**:

- High-frequency clicking = high safety, low efficiency
- Precision timing = high efficiency, higher risk
- Skill expressed through risk discipline

Reputation is a rolling 30-day average of Survival Efficiency.

### Elo-Based Matchmaking

Efficiency bands group players.

**Benefits**:

- Prevents whale domination of beginners
- Encourages competitive optimization
- Enables tier-based progression

Higher tiers provide improved shard rewards and Prestige Multiplier.

## Social Layer

### Whale Watch Syndicates

Players form squads.

**Features**:

- Shared identity
- Squad leaderboards
- War Chest system

3% of the total rake is funneled into the Squad War Chest.

Squad Funds automatically ensure streak protection events based on defined contribution rules.

No manual favoritism allowed.

### Prestige Multiplier

Live Alpha rooms carry a higher Prestige Multiplier.

Higher prestige:

- Increases shard accrual rate
- Boosts leaderboard visibility
- Unlocks higher-limit rooms

### Seasonal Resets

Public Reputation and Squad ranks reset monthly.

Hidden lifetime rating persists for matchmaking stability.

Prevents stagnation and invites cyclical re-engagement.

## Ethical Extraction Model

### Burn Structure

Revenue sources:

- 10% rake from pool
- 100% capture of pulse clicks
- Cosmetic sales
- Tier access upgrades

### Consolation Claim - Blitz Shards

To reduce churn from heavy burn extraction:

40-60% of burned Stars convert into Blitz Shards.

Shards are used in cosmetic-only storefront:

- Custom haptic patterns
- Sentinel Avatar evolution
- Profile badges visible across Telegram groups
- Seasonal cosmetic drops

Shards are non-pay-to-win.

Shard decay or seasonal recalibration prevents inflation.

This converts "loss" into perceived asset acquisition.

## Monetization Mix

Rake (10%)
Sustainable peer-to-peer exchange fee.

Burn Capture (100%)
Micro-transaction for time-extension.

Cosmetic Upsell
Purely social signals, preserving skill-first integrity.

Tiered Rooms
Higher entry = higher volatility = higher burn velocity.

## Mobile Immersion

Adaptive Haptics
Haptic intensity increases as liquidation proximity tightens.
Activated only during high-danger windows to avoid desensitization.

**UI Design Principles**:

- Flashing proximity indicators
- Survivor counter
- Top-3 highlight emphasis
- Minimal analytical overlays (reduce rational cooling)

## Liquidity & Risk Management

**Synthetic volatility must**:

- Avoid predictable patterns
- Maintain statistical randomness
- Prevent exploitative timing farming

**Real whale rooms require**:

- Transparent oracle sourcing
- Honest latency modeling
- Clear labeling

Market pacing is controlled to maintain consistent Survival Phase frequency.

## Failure Modes

**Primary collapse risks**:

- Burn inflation leading to a shard economy saturation
- Latency-based sniping dominance
- Synthetic pattern detection
- Excessive extraction velocity causing churn spike

**Mitigations**:

- Volatility multiplier weighting
- Device latency normalization windows
- Statistical entropy injection in synthetic models
- Controlled shard emission

## Positioning

Last Click is not marketed as gambling.

It is positioned as:

A competitive volatility survival arena inside Telegram.

No guaranteed profits.
No hidden odds.
Clear fee structure.

## Strategic Intent

**Short-Term Mode (3-Month Cash Engine)**:

- High volatility
- Aggressive burn
- Heavy cosmetic monetization
- Fast prestige cycling

**Long-Term Mode (Ecosystem Evolution)**:

- Skill-tier development
- Advanced squad meta
- Live event sponsorship
- Market-integrated competitive seasons

## Vision

**Last Click converges**:

- DeFi liquidation theatre
- Battle royale survival tension
- Penny-auction escalation mechanics
- Skill-based efficiency optimization
- Squad-driven social proof

It creates a volatility-native competitive environment inside Telegram that monetizes time under pressure.
