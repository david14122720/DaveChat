import React, { useEffect } from 'react';
import AuthLayout from './components/AuthLayout';
import MainLayout from './components/MainLayout';
import { useAuth } from './hooks/useAuth';
import { CallProvider } from './contexts/CallContext';
import CallUI from './components/CallUI';
import { wsClient } from './lib/websocket';
import { Loader2 } from 'lucide-react';

function App() {
  const { user, loading, signIn, signUp, signOut } = useAuth();

  useEffect(() => {
    if (user) {
      wsClient.connect();
    } else {
      wsClient.disconnect();
    }
  }, [user]);

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="w-10 h-10 text-brand-light animate-spin" />
          <p className="text-slate-400 font-medium animate-pulse">Cargando DaveChat...</p>
        </div>
      </div>
    );
  }

  return (
    <CallProvider>
      <CallUI />
      <div className="w-full h-full">
        {!user ? (
          <AuthLayout 
            onSignIn={signIn} 
            onSignUp={signUp} 
          />
        ) : (
          <MainLayout 
            user={user} 
            onLogout={signOut} 
          />
        )}
      </div>
    </CallProvider>
  );
}

export default App;
