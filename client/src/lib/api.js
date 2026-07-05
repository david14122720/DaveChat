const API_BASE = import.meta.env.VITE_API_URL || '';

class ApiClient {
  constructor() {
    this.baseUrl = API_BASE;
  }

  getToken() {
    return localStorage.getItem('davechat_token');
  }

  setToken(token) {
    localStorage.setItem('davechat_token', token);
  }

  clearToken() {
    localStorage.removeItem('davechat_token');
  }

  async request(path, options = {}) {
    const token = this.getToken();
    const headers = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const res = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers,
    });

    if (res.status === 401) {
      this.clearToken();
      throw new Error('Sesión expirada');
    }

    const data = await res.json();

    if (!res.ok) {
      throw new Error(data?.error?.message || `Error ${res.status}`);
    }

    return data;
  }

  // Auth
  async register(username, email, password) {
    const data = await this.request('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    });
    this.setToken(data.token);
    return data;
  }

  async login(email, password) {
    const data = await this.request('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    this.setToken(data.token);
    return data;
  }

  async me() {
    return this.request('/api/auth/me');
  }

  logout() {
    this.clearToken();
  }

  // Profiles
  async getProfiles() {
    return this.request('/api/profiles');
  }

  async getProfile(id) {
    return this.request(`/api/profiles/${id}`);
  }

  // Conversations
  async getConversations() {
    return this.request('/api/conversations');
  }

  async createConversation(userId) {
    return this.request('/api/conversations', {
      method: 'POST',
      body: JSON.stringify({ participant_id: userId }),
    });
  }

  // Messages
  async getMessages(conversationId, cursor) {
    const params = cursor ? `?cursor=${cursor}` : '';
    return this.request(`/api/conversations/${conversationId}/messages${params}`);
  }

  async sendMessage(conversationId, content) {
    return this.request(`/api/conversations/${conversationId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    });
  }

  async refreshToken(refreshToken) {
    const data = await this.request('/api/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    this.setToken(data.token);
    return data;
  }

  async uploadAvatar(formData) {
    const token = this.getToken();
    const res = await fetch(`${this.baseUrl}/api/profiles/me/avatar`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` },
      body: formData,
    });
    if (res.status === 401) { this.clearToken(); window.location.reload(); }
    if (!res.ok) { const err = await res.json().catch(() => ({})); throw new Error(err?.error?.message || `Upload failed (${res.status})`); }
    return res.json();
  }

  async deleteAvatar() {
    return this.request('/api/profiles/me/avatar', { method: 'DELETE' });
  }

  async searchMessages(conversationId, query) {
    return this.request(`/api/conversations/${conversationId}/messages/search?q=${encodeURIComponent(query)}`);
  }

  async markRead(conversationId) {
    return this.request(`/api/conversations/${conversationId}/read`, {
      method: 'POST',
    });
  }

  async getCallLogs() {
    return this.request('/api/call-logs');
  }

  async createCallLog(data) {
    return this.request('/api/call-logs', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }
}

export const api = new ApiClient();
