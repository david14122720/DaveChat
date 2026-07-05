import React, { useState, useEffect, useRef } from 'react';
import { PhoneOff, Mic, MicOff, Video, User, AlertCircle } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { useCall } from '../contexts/CallContext';

const STATUS_MESSAGES = {
  calling: 'Llamando...',
  ringing: 'Llamada entrante',
  rejected: 'Llamada rechazada',
  busy: 'Ocupado',
  timeout: 'No contestó',
  ended: 'Llamada finalizada',
};

const AUTO_HIDE_STATES = ['rejected', 'busy', 'timeout', 'ended'];

export default function CallOverlay() {
  const call = useCall();
  const { callState, peerId, localStream, remoteStream, callType, endCall, resetCall, audioDevices, selectedAudioDevice, setSelectedAudioDevice } = call;

  const remoteVideoRef = useRef(null);
  const localVideoRef = useRef(null);
  const [elapsed, setElapsed] = useState(0);
  const timerRef = useRef(null);
  const [micEnabled, setMicEnabled] = useState(true);
  const [videoEnabled, setVideoEnabled] = useState(true);

  const isVisible = callState !== 'idle' && callState !== 'ringing';

  useEffect(() => {
    if (AUTO_HIDE_STATES.includes(callState)) {
      const timer = setTimeout(() => resetCall(), 2000);
      return () => clearTimeout(timer);
    }
  }, [callState, resetCall]);

  useEffect(() => {
    if (callState === 'connected') {
      const startTime = Date.now();
      setElapsed(0);
      timerRef.current = setInterval(() => {
        setElapsed(Math.floor((Date.now() - startTime) / 1000));
      }, 1000);
    }
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [callState]);

  useEffect(() => {
    if (remoteVideoRef.current && remoteStream) {
      remoteVideoRef.current.srcObject = remoteStream;
    }
  }, [remoteStream]);

  useEffect(() => {
    if (localVideoRef.current && localStream) {
      localVideoRef.current.srcObject = localStream;
    }
  }, [localStream]);

  const formatTimer = (seconds) => {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return String(m).padStart(2, '0') + ':' + String(s).padStart(2, '0');
  };

  const toggleMic = () => {
    if (localStream) {
      const audioTrack = localStream.getAudioTracks()[0];
      if (audioTrack) {
        audioTrack.enabled = !audioTrack.enabled;
        setMicEnabled(audioTrack.enabled);
      }
    }
  };

  const toggleVideo = () => {
    if (localStream) {
      const videoTrack = localStream.getVideoTracks()[0];
      if (videoTrack) {
        videoTrack.enabled = !videoTrack.enabled;
        setVideoEnabled(videoTrack.enabled);
      }
    }
  };

  const [showMicPicker, setShowMicPicker] = useState(false);

  if (callState === 'idle' && call.error) {
    return (
      <div className="fixed inset-0 z-50 flex items-end justify-center pb-24 pointer-events-none">
        <div className="bg-red-500/10 border border-red-500/20 rounded-2xl p-4 flex items-center gap-3 text-red-400 text-sm pointer-events-auto backdrop-blur-xl shadow-2xl">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <p className="flex-1">{call.error}</p>
          <button
            onClick={resetCall}
            className="text-red-400/60 hover:text-red-400 transition-colors text-lg leading-none"
            title="Descartar"
          >
            ×
          </button>
        </div>
      </div>
    );
  }

  if (!isVisible) return null;

  const showVideo = callState === 'connected' && remoteStream;

  return (
    <AnimatePresence>
      <motion.div
        key="call-overlay"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.2 }}
        className="fixed inset-0 z-50 flex flex-col bg-slate-950/95 backdrop-blur-lg"
      >
        <div className="absolute top-0 left-0 right-0 z-10 flex flex-col items-center pt-8 pb-4 bg-gradient-to-b from-black/50 to-transparent">
          <div className="flex flex-col items-center gap-1">
            <span className="text-slate-400 text-sm font-medium">
              {peerId ? 'ID: ' + peerId : '—'}
            </span>
            {callState === 'connected' ? (
              <span className="text-white text-3xl font-mono font-bold tracking-wider">
                {formatTimer(elapsed)}
              </span>
            ) : (
              <span className="text-white/80 text-lg font-medium">
                {STATUS_MESSAGES[callState] || callState}
              </span>
            )}
          </div>
        </div>

        <div className="flex-1 flex items-center justify-center relative">
          {showVideo ? (
            <video
              ref={remoteVideoRef}
              autoPlay
              playsInline
              className="w-full h-full object-cover"
            />
          ) : (
            <motion.div
              animate={{
                scale: [1, 1.05, 1],
                opacity: [0.7, 1, 0.7],
              }}
              transition={{
                duration: 2,
                repeat: Infinity,
                ease: 'easeInOut',
              }}
              className="w-32 h-32 rounded-full bg-slate-800 border-2 border-slate-600 flex items-center justify-center shadow-2xl"
            >
              <User className="w-16 h-16 text-slate-400" />
            </motion.div>
          )}
        </div>

        <div className="absolute bottom-28 right-4 z-20">
          <div className="w-32 h-48 rounded-xl overflow-hidden border-2 border-slate-600 shadow-2xl bg-slate-900">
            {localStream ? (
              <video
                ref={localVideoRef}
                autoPlay
                muted
                playsInline
                className="w-full h-full object-cover scale-x-[-1]"
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center">
                <User className="w-8 h-8 text-slate-600" />
              </div>
            )}
          </div>
        </div>

        <motion.div
          initial={{ y: 50, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ delay: 0.15, duration: 0.3 }}
          className="absolute bottom-0 left-0 right-0 z-10 flex items-center justify-center gap-6 pb-10 pt-6 bg-gradient-to-t from-black/50 to-transparent"
        >
          <div className="relative">
            <button
              onClick={toggleMic}
              className={'w-14 h-14 md:w-16 md:h-16 rounded-full flex items-center justify-center shadow-xl transition-all active:scale-95 ' +
                (micEnabled
                  ? 'bg-slate-700 hover:bg-slate-600 text-white'
                  : 'bg-red-600/80 hover:bg-red-600 text-white')
              }
              title={micEnabled ? 'Silenciar micrófono' : 'Activar micrófono'}
            >
              {micEnabled ? <Mic className="w-6 h-6 md:w-7 md:h-7" /> : <MicOff className="w-6 h-6 md:w-7 md:h-7" />}
            </button>
            {audioDevices.length > 1 && (
              <button
                onClick={() => setShowMicPicker(!showMicPicker)}
                className="absolute -top-1 -right-1 w-5 h-5 bg-slate-600 rounded-full flex items-center justify-center text-[10px] text-white hover:bg-slate-500 transition-colors"
                title="Cambiar micrófono"
              >
                ⚙
              </button>
            )}
            {showMicPicker && audioDevices.length > 1 && (
              <div className="absolute bottom-20 left-1/2 -translate-x-1/2 bg-slate-900 border border-slate-700 rounded-2xl p-3 shadow-2xl w-64 z-50">
                <p className="text-xs text-slate-400 font-medium mb-2">Micrófono</p>
                {audioDevices.map(d => (
                  <button
                    key={d.deviceId}
                    onClick={() => { setSelectedAudioDevice(d.deviceId); setShowMicPicker(false); }}
                    className={'w-full text-left px-3 py-2 rounded-xl text-sm transition-colors ' +
                      (selectedAudioDevice === d.deviceId
                        ? 'bg-brand/20 text-brand-light'
                        : 'text-slate-300 hover:bg-slate-800')
                    }
                  >
                    {d.label || `Micrófono ${d.deviceId.slice(0, 8)}...`}
                  </button>
                ))}
              </div>
            )}
          </div>

          {callType === 'video' && (
            <button
              onClick={toggleVideo}
              className={'w-14 h-14 md:w-16 md:h-16 rounded-full flex items-center justify-center shadow-xl transition-all active:scale-95 ' +
                (videoEnabled
                  ? 'bg-slate-700 hover:bg-slate-600 text-white'
                  : 'bg-red-600/80 hover:bg-red-600 text-white')
              }
              title={videoEnabled ? 'Desactivar video' : 'Activar video'}
            >
              <Video className="w-6 h-6 md:w-7 md:h-7" />
            </button>
          )}

          <button
            onClick={endCall}
            className="w-14 h-14 md:w-16 md:h-16 rounded-full flex items-center justify-center bg-red-600 hover:bg-red-500 text-white shadow-xl transition-all active:scale-95 hover:scale-110"
            title="Colgar llamada"
          >
            <PhoneOff className="w-6 h-6 md:w-7 md:h-7" />
          </button>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
}
