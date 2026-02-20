import type { PlayerProfile, LeaderboardEntry } from "@/types/game";

export class NotFoundError extends Error {
  constructor(url: string) {
    super(`Not found: ${url}`);
    this.name = "NotFoundError";
  }
}

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (res.status === 404) throw new NotFoundError(url);
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.json() as Promise<T>;
}

export function getPlayer(id: number) {
  return fetchJSON<PlayerProfile>(`/api/player/${id}`);
}

export function getLeaderboardPlayers(count = 20) {
  return fetchJSON<LeaderboardEntry[]>(
    `/api/leaderboard/players?count=${count}`,
  );
}

export function getPlayerRank(playerID: number) {
  return fetchJSON<LeaderboardEntry>(`/api/leaderboard/rank/${playerID}`);
}
