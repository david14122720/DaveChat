import { api } from './api';

class WebSocketClient {
  constructor() {
    this.ws = null;
    this.listeners = {};
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.pendingMessages = [];
    this.pingInterval = null;
  }

  connect() {
    const token = api.getToken();
    if (!token) return;
    if (this.ws && this.ws.readyState === WebSocket.OPEN) return;
    this.maxReconnectAttempts = 5;

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const host = window.location.host;
    const url = protocol + '://' + host + '/api/ws';

    this.ws = new WebSocket(url, [token]);

    this._setupVisibilityHandler();

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this._startPing();
      this._flushPending();
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        this._emit(msg.type, msg);
      } catch (e) {
        console.error('WS parse error:', e);
      }
    };

    this.ws.onclose = () => {
      this._stopPing();
      this._emit('close', {});
      this._reconnect();
    };

    this.ws.onerror = (err) => {
      console.error('⚠️ WebSocket error:', err);
    };
  }

  _reconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return;
    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts - 1), 30000);
    const jitter = Math.random() * 1000;
    setTimeout(() => this.connect(), delay + jitter);
  }

  _flushPending() {
    while (this.pendingMessages.length > 0) {
      const msg = this.pendingMessages.shift();
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify(msg));
      } else {
        // Connection dropped again — re-queue and stop flushing
        this.pendingMessages.unshift(msg);
        break;
      }
    }
  }

  _startPing() {
    this._stopPing();
    this.pingInterval = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'ping' }));
      }
    }, 25000);
  }

  _stopPing() {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  _setupVisibilityHandler() {
    this._teardownVisibilityHandler();
    const handleVisibility = () => {
      if (document.visibilityState === 'visible') {
        console.log('[WS] Tab visible, reconnecting');
        this.maxReconnectAttempts = 5;
        if (this.ws) {
          this.ws.onclose = null;
          this.ws.close();
          this.ws = null;
        }
        this.connect();
      }
    };
    this._visibilityHandler = handleVisibility;
    document.addEventListener('visibilitychange', handleVisibility);

    this._pageshowHandler = (event) => {
      if (event.persisted) {
        handleVisibility();
      }
    };
    window.addEventListener('pageshow', this._pageshowHandler);
  }

  _teardownVisibilityHandler() {
    if (this._visibilityHandler) {
      document.removeEventListener('visibilitychange', this._visibilityHandler);
      this._visibilityHandler = null;
    }
    if (this._pageshowHandler) {
      window.removeEventListener('pageshow', this._pageshowHandler);
      this._pageshowHandler = null;
    }
  }

  send(type, payload = {}, target = '', extra = {}) {
    const msg = { type, payload, target, ...extra };
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
      return;
    }
    console.warn('⚠️ WS no conectado, encolando mensaje:', msg);
    this.pendingMessages.push(msg);
    this.connect(); // Ensure we're trying to connect
  }

  on(type, callback) {
    if (!this.listeners[type]) this.listeners[type] = [];
    this.listeners[type].push(callback);
    return () => {
      this.listeners[type] = this.listeners[type].filter(cb => cb !== callback);
    };
  }

  _emit(type, data) {
    (this.listeners[type] || []).forEach(cb => cb(data));
    (this.listeners['*'] || []).forEach(cb => cb(data));
  }

  disconnect() {
    this._teardownVisibilityHandler();
    this._stopPing();
    this.maxReconnectAttempts = 0;
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
  }

  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN;
  }
}

export const wsClient = new WebSocketClient();