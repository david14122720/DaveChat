import { useState, useEffect } from 'react';
import { api } from '../lib/api';

export const useAuth = () => {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = api.getToken();
    if (!token) {
      setLoading(false);
      return;
    }

    api.me()
      .then((userData) => setUser(userData))
      .catch(() => {
        api.clearToken();
      })
      .finally(() => setLoading(false));
  }, []);

  const signIn = async (email, password) => {
    const data = await api.login(email, password);
    setUser(data.user);
    return data;
  };

  const signUp = async (email, password, username) => {
    const data = await api.register(username, email, password);
    setUser(data.user);
    return data;
  };

  const signOut = () => {
    api.logout();
    setUser(null);
  };

  return { user, loading, signIn, signUp, signOut };
};
