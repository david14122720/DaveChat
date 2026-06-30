import { api } from './api';

class WebSocketClient {
  constructor() {
    this.ws = null;
    this.listeners = {};
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
  }

  connect() {
    const token = api.getToken();
    if (!token) return;
    if (this.ws && this.ws.readyState === WebSocket.OPEN) return;
    this.maxReconnectAttempts = 5;

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const host = window.location.host;
    const url = protocol + '://' + host + '/api/ws?token=' + token;

    this.ws = new WebSocket(url);

    this.ws.onopen = () => {
      console.log('✅ WebSocket conectado');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        console.log('📩 WS recibe:', msg);
        this._emit(msg.type, msg);
      } catch (e) {
        console.error('WS parse error:', e);
      }
    };

    this.ws.onclose = () => {
      console.log('❌ WebSocket desconectado');
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
    console.log('🔄 Reintentando conexión WS (' + this.reconnectAttempts + '/' + this.maxReconnectAttempts + ')...');
    setTimeout(() => this.connect(), 1000 * this.reconnectAttempts);
  }

  send(type, payload = {}, target = '') {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('⚠️ WS no conectado');
      return;
    }
    const msg = { type, payload, target };
    console.log('📤 WS envía:', msg);
    this.ws.send(JSON.stringify(msg));
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
    this.maxReconnectAttempts = 0;
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN;
  }
}

export const wsClient = new WebSocketClient();
