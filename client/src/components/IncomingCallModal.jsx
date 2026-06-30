import React, { useRef, useState, useEffect } from 'react';
import { Phone, PhoneOff, User } from 'lucide-react';
import { motion } from 'framer-motion';
import { useCall } from '../contexts/CallContext';

const TIMEOUT_SECONDS = 30;

export default function IncomingCallModal() {
  const call = useCall();
  const { peerId, callType, acceptCall, rejectCall } = call;
  const [countdown, setCountdown] = useState(TIMEOUT_SECONDS);
  const ringIntervalRef = useRef(null);

  useEffect(() => {
    if (countdown <= 0) {
      rejectCall();
      return;
    }
    const timer = setInterval(() => {
      setCountdown((prev) => prev - 1);
    }, 1000);
    return () => clearInterval(timer);
  }, [countdown, rejectCall]);

  useEffect(() => {
    const playBeep = () => {
      try {
        const ctx = new (window.AudioContext || window.webkitAudioContext)();
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.frequency.value = 440;
        gain.gain.value = 0.3;
        osc.start();
        setTimeout(() => {
          osc.stop();
          ctx.close();
        }, 500);
      } catch (e) {
      }
    };

    playBeep();
    ringIntervalRef.current = setInterval(playBeep, 3000);

    return () => {
      if (ringIntervalRef.current) {
        clearInterval(ringIntervalRef.current);
        ringIntervalRef.current = null;
      }
    };
  }, []);

  const progress = ((TIMEOUT_SECONDS - countdown) / TIMEOUT_SECONDS) * 100;

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm"
    >
      <motion.div
        initial={{ scale: 0.9, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ type: 'spring', damping: 20, stiffness: 300 }}
        className="w-full max-w-sm mx-4 bg-slate-900 border border-slate-700 rounded-3xl shadow-2xl overflow-hidden"
      >
        <div className="p-8 flex flex-col items-center gap-6">
          <motion.div
            animate={{ scale: [1, 1.05, 1] }}
            transition={{ duration: 1.5, repeat: Infinity, ease: 'easeInOut' }}
            className="w-24 h-24 rounded-full bg-slate-800 border-2 border-slate-600 flex items-center justify-center shadow-xl"
          >
            <User className="w-12 h-12 text-slate-400" />
          </motion.div>

          <div className="text-center">
            <h2 className="text-xl font-bold text-white">Llamada entrante</h2>
            <p className="text-slate-400 mt-1">ID: {peerId || '—'}</p>
            <p className="text-brand-light text-sm mt-1 font-medium">
              {callType === 'video' ? 'Llamada de video' : 'Llamada de audio'}
            </p>
          </div>

          <div className="w-full space-y-2">
            <div className="w-full h-2 bg-slate-800 rounded-full overflow-hidden">
              <motion.div
                className="h-full rounded-full"
                animate={{ backgroundColor: progress > 70 ? '#ef4444' : progress > 40 ? '#f59e0b' : '#25D366' }}
                style={{ width: (100 - progress) + '%' }}
                transition={{ duration: 0.5 }}
              />
            </div>
            <p className="text-xs text-slate-500 text-center">
              La llamada expirará en {countdown}s
            </p>
          </div>

          <div className="flex items-center gap-8">
            <button
              onClick={rejectCall}
              className="w-16 h-16 rounded-full bg-red-600 hover:bg-red-500 text-white flex items-center justify-center shadow-xl transition-all active:scale-95 hover:scale-110"
              title="Rechazar"
            >
              <PhoneOff className="w-7 h-7" />
            </button>

            <button
              onClick={acceptCall}
              className="w-16 h-16 rounded-full bg-brand-light hover:bg-brand text-slate-950 flex items-center justify-center shadow-xl transition-all active:scale-95 hover:scale-110"
              title="Aceptar"
            >
              <Phone className="w-7 h-7" />
            </button>
          </div>
        </div>
      </motion.div>
    </motion.div>
  );
}
