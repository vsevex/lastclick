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
    const sdk = await import("@telegram-apps/sdk");
    sdk.init();

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const lp = sdk.retrieveLaunchParams() as any;
    const initDataRaw: string | null = lp.initDataRaw ?? null;
    const user = lp.initData?.user as
      | { id?: number; username?: string; firstName?: string }
      | undefined;
    const userId: number | null = user?.id ?? null;
    const username: string | null = user?.username ?? user?.firstName ?? null;

    try {
      sdk.miniApp.mount();
      sdk.miniApp.setHeaderColor("#0B0F14");
      sdk.miniApp.setBackgroundColor("#0B0F14");
    } catch {
      /* optional */
    }

    try {
      await sdk.viewport.mount();
      sdk.viewport.expand();
    } catch {
      /* optional */
    }

    try {
      sdk.miniApp.ready();
    } catch {
      /* optional */
    }

    return { ready: true, userId, initDataRaw, username };
  } catch {
    return { ready: true, userId: 1, initDataRaw: null, username: "Guest" };
  }
}
