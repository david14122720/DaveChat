import React, { useState } from 'react';
import { Mail, Lock, LogIn, UserPlus, AlertCircle, Loader2 } from 'lucide-react';
import { motion } from 'framer-motion';

const AuthLayout = ({ onSignIn, onSignUp }) => {
  const [isLogin, setIsLogin] = useState(true);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [username, setUsername] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      if (isLogin) {
        await onSignIn(email, password);
      } else {
        await onSignUp(email, password, username);
      }
    } catch (err) {
      setError(err.message || 'Ocurrió un error inesperado');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,_var(--tw-gradient-stops))] from-brand/10 via-transparent to-transparent pointer-events-none"></div>
      
      <motion.div 
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        className="max-w-md w-full bg-slate-900/50 backdrop-blur-xl rounded-3xl shadow-2xl border border-slate-800 p-8 relative z-10"
      >
        <div className="text-center mb-8">
          <div className="inline-flex p-3 bg-brand/10 rounded-2xl mb-4">
            <LogIn className="w-8 h-8 text-brand-light" />
          </div>
          <h1 className="text-3xl font-bold text-white mb-2">DaveChat</h1>
          <p className="text-slate-400 text-sm">{isLogin ? 'Accede a tu zona privada' : 'Crea tu perfil de mensajería'}</p>
        </div>

        {error && (
          <motion.div 
            initial={{ opacity: 0, x: -10 }} 
            animate={{ opacity: 1, x: 0 }}
            className="mb-6 p-4 bg-red-500/10 border border-red-500/20 rounded-2xl flex items-center gap-3 text-red-400 text-sm"
          >
            <AlertCircle className="w-5 h-5 flex-shrink-0" />
            <p>{error}</p>
          </motion.div>
        )}

        <form className="space-y-5" onSubmit={handleSubmit}>
          {!isLogin && (
            <div className="space-y-2">
              <label className="text-xs font-semibold text-slate-500 uppercase tracking-wider ml-1">Username</label>
              <div className="relative group">
                <input 
                  type="text" 
                  required
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full bg-slate-800/50 border border-slate-700 rounded-2xl py-3 px-11 text-white focus:ring-2 focus:ring-brand-light focus:border-transparent outline-none transition-all group-hover:border-slate-600" 
                  placeholder="ej. ElDave" 
                />
                <UserPlus className="absolute left-4 top-3.5 text-slate-500 w-5 h-5 group-hover:text-brand-light transition-colors" />
              </div>
            </div>
          )}
          
          <div className="space-y-2">
            <label className="text-xs font-semibold text-slate-500 uppercase tracking-wider ml-1">Email</label>
            <div className="relative group">
              <input 
                type="email" 
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full bg-slate-800/50 border border-slate-700 rounded-2xl py-3 px-11 text-white focus:ring-2 focus:ring-brand-light focus:border-transparent outline-none transition-all group-hover:border-slate-600" 
                placeholder="tu@email.com" 
              />
              <Mail className="absolute left-4 top-3.5 text-slate-500 w-5 h-5 group-hover:text-brand-light transition-colors" />
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-xs font-semibold text-slate-500 uppercase tracking-wider ml-1">Contraseña</label>
            <div className="relative group">
              <input 
                type="password" 
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-slate-800/50 border border-slate-700 rounded-2xl py-3 px-11 text-white focus:ring-2 focus:ring-brand-light focus:border-transparent outline-none transition-all group-hover:border-slate-600" 
                placeholder="••••••••" 
              />
              <Lock className="absolute left-4 top-3.5 text-slate-500 w-5 h-5 group-hover:text-brand-light transition-colors" />
            </div>
          </div>

          <button 
            type="submit" 
            disabled={loading}
            className="w-full bg-brand-light hover:bg-brand text-slate-950 font-bold py-4 rounded-2xl shadow-lg shadow-brand/20 transition-all active:scale-[0.98] disabled:opacity-50 flex items-center justify-center gap-3"
          >
            {loading ? <Loader2 className="w-5 h-5 animate-spin" /> : (isLogin ? <LogIn className="w-5 h-5" /> : <UserPlus className="w-5 h-5" />)}
            {isLogin ? 'Entrar ahora' : 'Registrar Cuenta'}
          </button>
        </form>

        <div className="mt-8 pt-6 border-t border-slate-800 text-center">
          <button 
            onClick={() => { setIsLogin(!isLogin); setError(''); }}
            className="text-slate-400 hover:text-brand-light text-sm font-medium transition-colors"
          >
            {isLogin ? '¿Nuevo en DaveChat? Crea una cuenta' : '¿Ya tienes cuenta? Inicia sesión'}
          </button>
        </div>
      </motion.div>
    </div>
  );
};

export default AuthLayout;
