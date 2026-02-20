export interface WSEnvelope {
  type: string;
  payload?: unknown;
}

type Listener = (payload: unknown) => void;

export class GameSocket {
  private ws: WebSocket | null = null;
  private listeners = new Map<string, Set<Listener>>();
  private reconnectDelay = 1000;
  private maxReconnectDelay = 16000;
  private shouldReconnect = true;
  private connectParams: string;

  constructor(connectParams: string) {
    this.connectParams = connectParams;
  }

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl =
      import.meta.env.VITE_WS_URL || `${proto}//${window.location.host}`;
    const url = `${wsUrl}/ws?${this.connectParams}`;

    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      this.reconnectDelay = 1000;
      this.emit("_connected", null);
    };

    this.ws.onmessage = (event) => {
      try {
        const msg: WSEnvelope = JSON.parse(event.data);
        this.emit(msg.type, msg.payload);
      } catch {
        /* ignore malformed messages */
      }
    };

    this.ws.onclose = () => {
      this.emit("_disconnected", null);
      if (this.shouldReconnect) {
        setTimeout(() => this.connect(), this.reconnectDelay);
        this.reconnectDelay = Math.min(
          this.reconnectDelay * 2,
          this.maxReconnectDelay,
        );
      }
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  disconnect() {
    this.shouldReconnect = false;
    this.ws?.close();
    this.ws = null;
  }

  send(type: string, payload?: unknown) {
    if (this.ws?.readyState !== WebSocket.OPEN) return;
    const msg: WSEnvelope = { type };
    if (payload !== undefined) {
      msg.payload = payload;
    }
    this.ws.send(JSON.stringify(msg));
  }

  on(type: string, listener: Listener): () => void {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, new Set());
    }
    this.listeners.get(type)!.add(listener);
    return () => this.listeners.get(type)?.delete(listener);
  }

  get connected() {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  private emit(type: string, payload: unknown) {
    this.listeners.get(type)?.forEach((fn) => fn(payload));
  }
}
