"use client";

import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  useCallback,
  type ReactNode,
} from "react";
import { GameSocket } from "@/lib/ws";
import { useTelegram } from "@/context/TelegramProvider";

interface SocketContextType {
  connected: boolean;
  send: (type: string, payload?: unknown) => void;
  on: (type: string, listener: (payload: unknown) => void) => () => void;
}

const SocketContext = createContext<SocketContextType>({
  connected: false,
  send: () => {},
  on: () => () => {},
});

export function useSocket() {
  return useContext(SocketContext);
}

export function SocketProvider({ children }: { children: ReactNode }) {
  const { userId, initDataRaw } = useTelegram();
  const socketRef = useRef<GameSocket | null>(null);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    let params: string;
    if (initDataRaw) {
      params = `initData=${encodeURIComponent(initDataRaw)}`;
    } else {
      params = `user_id=${userId ?? 1}`;
    }

    const socket = new GameSocket(params);
    socketRef.current = socket;

    socket.on("_connected", () => setConnected(true));
    socket.on("_disconnected", () => setConnected(false));
    socket.connect();

    return () => {
      socket.disconnect();
      socketRef.current = null;
    };
  }, [userId, initDataRaw]);

  const send = useCallback((type: string, payload?: unknown) => {
    socketRef.current?.send(type, payload);
  }, []);

  const on = useCallback(
    (type: string, listener: (payload: unknown) => void) => {
      return socketRef.current?.on(type, listener) ?? (() => {});
    },
    [],
  );

  return (
    <SocketContext.Provider value={{ connected, send, on }}>
      {children}
    </SocketContext.Provider>
  );
}
