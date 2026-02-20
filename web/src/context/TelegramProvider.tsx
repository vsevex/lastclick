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
    const tg = window.Telegram?.WebApp;

    if (tg) {
      tg.ready();
      tg.expand();

      try {
        tg.setHeaderColor("#0B0F14");
        tg.setBackgroundColor("#0B0F14");
      } catch {
        /* older clients may not support these */
      }
    }

    const user = tg?.initDataUnsafe?.user;
    setCtx({
      ready: true,
      userId: user?.id ?? null,
      initDataRaw: tg?.initData || null,
      username: user?.username ?? user?.first_name ?? null,
    });
  }, []);

  if (!ctx.ready) return null;

  return (
    <TelegramContext.Provider value={ctx}>{children}</TelegramContext.Provider>
  );
}
