"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";

interface TelegramContextType {
  ready: boolean;
  userId: number | null;
  initDataRaw: string | null;
  username: string | null;
}

const TelegramContext = createContext<TelegramContextType>({
  ready: false,
  userId: null,
  initDataRaw: null,
  username: null,
});

export function useTelegram() {
  return useContext(TelegramContext);
}

export function TelegramProvider({ children }: { children: ReactNode }) {
  const [ctx, setCtx] = useState<TelegramContextType>({
    ready: false,
    userId: null,
    initDataRaw: null,
    username: null,
  });

  useEffect(() => {
    initTelegram().then(setCtx);
  }, []);

  if (!ctx.ready) return null;

  return (
    <TelegramContext.Provider value={ctx}>{children}</TelegramContext.Provider>
  );
}

async function initTelegram(): Promise<TelegramContextType> {
  try {
    const { ensureTelegramEnv } = await import("@/lib/mockEnv");
    await ensureTelegramEnv();

    const sdk = await import("@telegram-apps/sdk");
    sdk.init();

    const lp = sdk.retrieveLaunchParams() as Record<string, unknown>;
    const initDataRaw: string | null = (lp.initDataRaw as string) ?? null;
    const initData = lp.initData as
      | { user?: { id?: number; username?: string; firstName?: string } }
      | undefined;
    const user = initData?.user;
    const userId: number | null = user?.id ?? null;
    const username: string | null = user?.username ?? user?.firstName ?? null;

    try {
      sdk.miniApp.mount();
      sdk.miniApp.setHeaderColor("#0B0F14");
      sdk.miniApp.setBackgroundColor("#0B0F14");
    } catch (e) {
      console.warn("[TG] miniApp mount failed:", e);
    }

    try {
      await sdk.viewport.mount();
      sdk.viewport.expand();
    } catch (e) {
      console.warn("[TG] viewport mount failed:", e);
    }

    try {
      sdk.miniApp.ready();
    } catch (e) {
      console.warn("[TG] miniApp.ready failed:", e);
    }

    console.info("[TG] initialized", {
      userId,
      username,
      hasInitData: !!initDataRaw,
    });
    return { ready: true, userId, initDataRaw, username };
  } catch (e) {
    console.warn("[TG] SDK init failed, using fallback:", e);
    return { ready: true, userId: 1, initDataRaw: null, username: "Guest" };
  }
}
