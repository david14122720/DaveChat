import React, { Suspense, lazy, useEffect } from 'react';
import { useAuth } from './hooks/useAuth';
import { CallProvider } from './contexts/CallContext';
import { Loader2 } from 'lucide-react';

const AuthLayout = lazy(() => import('./components/AuthLayout'));
const MainLayout = lazy(() => import('./components/MainLayout'));
const CallUI = lazy(() => import('./components/CallUI'));

function App() {
  const { user, loading, signIn, signUp, signOut } = useAuth();

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

  const fallback = (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <Loader2 className="w-10 h-10 text-brand-light animate-spin" />
        <p className="text-slate-400 font-medium animate-pulse">Cargando DaveChat...</p>
      </div>
    </div>
  );

  return (
    <Suspense fallback={fallback}>
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
    </Suspense>
  );
}

export default App;