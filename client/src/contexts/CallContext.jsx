import { createContext, useContext, useReducer, useEffect, useState, useCallback, useRef, useMemo } from 'react';
import { wsClient } from '../lib/websocket';
import { api } from '../lib/api';

const CallState = {
  IDLE: 'idle',
  CALLING: 'calling',
  RINGING: 'ringing',
  CONNECTED: 'connected',
  REJECTED: 'rejected',
  BUSY: 'busy',
  TIMEOUT: 'timeout',
  ENDED: 'ended',
};

const initialState = {
  callState: CallState.IDLE,
  peerId: null,
  callType: null,
  sdp: null,
  localStream: null,
  remoteStream: null,
  error: null,
};

function callReducer(state, action) {
  switch (action.type) {
    case 'SET_CALLING':
      return { ...state, callState: CallState.CALLING, peerId: action.peerId, callType: action.callType };
    case 'SET_RINGING':
      return { ...state, callState: CallState.RINGING, peerId: action.from, callType: action.callType, sdp: action.sdp };
    case 'SET_CONNECTED':
      return { ...state, callState: CallState.CONNECTED };
    case 'SET_REJECTED':
      return { ...state, callState: CallState.REJECTED, peerId: action.from || state.peerId };
    case 'SET_BUSY':
      return { ...state, callState: CallState.BUSY };
    case 'SET_TIMEOUT':
      return { ...state, callState: CallState.TIMEOUT };
    case 'SET_ENDED':
      return { ...state, callState: CallState.ENDED };
    case 'SET_ERROR':
      return { ...state, callState: CallState.IDLE, error: action.error };
    case 'SET_LOCAL_STREAM':
      return { ...state, localStream: action.stream };
    case 'SET_REMOTE_STREAM':
      return { ...state, remoteStream: action.stream };
    case 'RESET':
      return { ...initialState, callState: CallState.IDLE };
    default:
      return state;
  }
}

const CallContext = createContext(null);

export function CallProvider({ children }) {
  const [state, dispatch] = useReducer(callReducer, initialState);
  const pcRef = useRef(null);
  const localStreamRef = useRef(null);
  const remoteStreamRef = useRef(null);
  const ringtoneCtxRef = useRef(null);
  const ringtoneIntervalRef = useRef(null);
  const callStateRef = useRef(CallState.IDLE);
  const iceConfigRef = useRef(null);

  const [audioDevices, setAudioDevices] = useState([]);
  const [selectedAudioDevice, setSelectedAudioDevice] = useState('');

  useEffect(() => {
    navigator.mediaDevices.enumerateDevices().then(devices => {
      const inputs = devices.filter(d => d.kind === 'audioinput');
      setAudioDevices(inputs);
      if (!selectedAudioDevice && inputs.length > 0) {
        setSelectedAudioDevice(inputs[0].deviceId);
      }
    }).catch(() => {});
  }, []);

  useEffect(() => {
    const handler = () => {
      navigator.mediaDevices.enumerateDevices().then(devices => {
        setAudioDevices(devices.filter(d => d.kind === 'audioinput'));
      }).catch(() => {});
    };
    navigator.mediaDevices.addEventListener('devicechange', handler);
    return () => navigator.mediaDevices.removeEventListener('devicechange', handler);
  }, []);

  const getIceConfig = useCallback(async () => {
    if (iceConfigRef.current) return iceConfigRef.current;
    try {
      const cfg = await api.request('/api/config/webrtc');
      iceConfigRef.current = cfg;
      return cfg;
    } catch {
      const fallback = {
        iceServers: [
          { urls: 'stun:stun.l.google.com:19302' },
          { urls: 'stun:stun1.l.google.com:19302' },
        ],
      };
      iceConfigRef.current = fallback;
      return fallback;
    }
  }, []);

  const cleanupCall = useCallback(() => {
    if (pcRef.current) {
      pcRef.current.close();
      pcRef.current = null;
    }
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach(track => track.stop());
      localStreamRef.current = null;
    }
    remoteStreamRef.current = null;
    dispatch({ type: 'SET_LOCAL_STREAM', stream: null });
    dispatch({ type: 'SET_REMOTE_STREAM', stream: null });
  }, [dispatch]);

  const startRingtone = useCallback(() => {
    try {
      const ctx = new (window.AudioContext || window.webkitAudioContext)();
      ringtoneCtxRef.current = ctx;

      const playRing = () => {
        if (!ringtoneCtxRef.current) return;
        const c = ringtoneCtxRef.current;
        const osc = c.createOscillator();
        const gain = c.createGain();
        osc.type = 'sine';
        osc.frequency.setValueAtTime(440, c.currentTime);
        osc.frequency.setValueAtTime(480, c.currentTime + 0.3);
        osc.frequency.setValueAtTime(440, c.currentTime + 0.6);
        gain.gain.setValueAtTime(0, c.currentTime);
        gain.gain.linearRampToValueAtTime(0.3, c.currentTime + 0.05);
        gain.gain.linearRampToValueAtTime(0.3, c.currentTime + 1.2);
        gain.gain.linearRampToValueAtTime(0, c.currentTime + 1.8);
        osc.connect(gain);
        gain.connect(c.destination);
        osc.start(c.currentTime);
        osc.stop(c.currentTime + 2);
      };

      playRing();
      ringtoneIntervalRef.current = setInterval(playRing, 4000);
    } catch (e) {
      console.warn('🔇 No se pudo reproducir ringtone:', e);
    }
  }, []);

  const stopRingtone = useCallback(() => {
    if (ringtoneIntervalRef.current) {
      clearInterval(ringtoneIntervalRef.current);
      ringtoneIntervalRef.current = null;
    }
    if (ringtoneCtxRef.current) {
      ringtoneCtxRef.current.close().catch(() => {});
      ringtoneCtxRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (state.callState === CallState.RINGING) {
      startRingtone();
    } else {
      stopRingtone();
    }
    return stopRingtone;
  }, [state.callState, startRingtone, stopRingtone]);

  useEffect(() => {
    callStateRef.current = state.callState;
  }, [state.callState]);

  const createPeerConnection = useCallback(async (remoteUserId, callType = 'audio') => {
    const iceConfig = await getIceConfig();
    const pc = new RTCPeerConnection(iceConfig);
    pcRef.current = pc;

    let stream;
    const audioConstraints = {
      echoCancellation: true,
      noiseSuppression: true,
      autoGainControl: true,
    };
    if (selectedAudioDevice) {
      audioConstraints.deviceId = { exact: selectedAudioDevice };
    }
    try {
      stream = await navigator.mediaDevices.getUserMedia({
        audio: audioConstraints,
        video: callType === 'video',
      });
    } catch (err) {
      if (callType === 'video') {
        console.warn('⚠️ No se pudo acceder a la cámara, continuando solo con audio:', err.message);
        try {
          stream = await navigator.mediaDevices.getUserMedia({
            audio: audioConstraints,
            video: false,
          });
        } catch (audioErr) {
          console.error('Error al acceder al micrófono:', audioErr);
          if (audioErr.name === 'NotAllowedError') {
            throw new Error('Permiso de micrófono denegado');
          } else if (audioErr.name === 'NotFoundError') {
            throw new Error('No se encontró ningún micrófono');
          } else {
            throw new Error('Error de medios: ' + audioErr.message);
          }
        }
      } else {
        console.error('Error al acceder al micrófono:', err);
        if (err.name === 'NotAllowedError') {
          throw new Error('Permiso de micrófono denegado');
        } else if (err.name === 'NotFoundError') {
          throw new Error('No se encontró ningún micrófono');
        } else if (err.name === 'AbortError') {
          throw new Error('Error al acceder al dispositivo de audio');
        } else {
          throw new Error('Error de medios: ' + err.message);
        }
      }
    }

    localStreamRef.current = stream;
    dispatch({ type: 'SET_LOCAL_STREAM', stream });
    stream.getTracks().forEach(track => {
      pc.addTrack(track, stream);
    });

    pc.ontrack = (event) => {
      remoteStreamRef.current = event.streams[0];
      dispatch({ type: 'SET_REMOTE_STREAM', stream: event.streams[0] });
    };

    pc.onicecandidate = (event) => {
      if (event.candidate) {
        wsClient.send('ice-candidate', { candidate: event.candidate.toJSON() }, remoteUserId);
      }
    };

    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'disconnected' || pc.connectionState === 'failed') {
        cleanupCall();
        dispatch({ type: 'SET_ERROR', error: 'Conexión perdida' });
      }
    };

    pc.oniceconnectionstatechange = () => {
      console.log('[ICE state]', pc.iceConnectionState);
      if (pc.iceConnectionState === 'failed' || pc.iceConnectionState === 'disconnected') {
        dispatch({ type: 'SET_ERROR', error: `Conexión de red perdida (${pc.iceConnectionState})` });
      }
    };

    return pc;
  }, [getIceConfig, cleanupCall, selectedAudioDevice]);

  useEffect(() => {
    wsClient.connect();

    const unsubscribes = [];

    unsubscribes.push(wsClient.on('call-offer', (msg) => {
      dispatch({ type: 'SET_RINGING', from: msg.from, callType: msg.call_type, sdp: msg.payload?.sdp });
    }));

    unsubscribes.push(wsClient.on('call-answer', (msg) => {
      const pc = pcRef.current;
      if (pc && msg.payload?.sdp) {
        const remoteAnswer = new RTCSessionDescription(msg.payload.sdp);
        pc.setRemoteDescription(remoteAnswer)
          .catch(err => console.error('Error setting remote description:', err));
      }
      dispatch({ type: 'SET_CONNECTED' });
    }));

    unsubscribes.push(wsClient.on('ice-candidate', (msg) => {
      const pc = pcRef.current;
      if (pc && msg.payload?.candidate) {
        pc.addIceCandidate(new RTCIceCandidate(msg.payload.candidate))
          .catch(err => console.error('Error adding ICE candidate:', err));
      }
    }));

    unsubscribes.push(wsClient.on('call-busy', () => {
      cleanupCall();
      dispatch({ type: 'SET_BUSY' });
    }));

    unsubscribes.push(wsClient.on('call-reject', (msg) => {
      cleanupCall();
      dispatch({ type: 'SET_REJECTED', from: msg.from });
    }));

    unsubscribes.push(wsClient.on('call-timeout', () => {
      cleanupCall();
      dispatch({ type: 'SET_TIMEOUT' });
    }));

    unsubscribes.push(wsClient.on('call-started', () => {
    }));

    unsubscribes.push(wsClient.on('call-end', (msg) => {
      cleanupCall();
      dispatch({ type: 'SET_ENDED' });
    }));

    unsubscribes.push(wsClient.on('close', () => {
      if (callStateRef.current !== CallState.IDLE) {
        console.warn('⚠️ WS desconectado durante llamada');
        cleanupCall();
        dispatch({ type: 'SET_ERROR', error: 'Conexión perdida' });
      }
    }));

    return () => {
      unsubscribes.forEach(fn => fn());
      cleanupCall();
    };
  }, [cleanupCall]);

  const startCall = useCallback(async (targetUserId, callType = 'audio') => {
    try {
      dispatch({ type: 'SET_CALLING', peerId: targetUserId, callType });

      const pc = await createPeerConnection(targetUserId, callType);
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);

      wsClient.send('call-offer', { sdp: offer }, targetUserId, { call_type: callType });
    } catch (err) {
      console.error('Error al iniciar llamada:', err);
      cleanupCall();
      dispatch({ type: 'SET_ERROR', error: err.message });
    }
  }, [createPeerConnection, cleanupCall]);

  const acceptCall = useCallback(async () => {
    const callerId = state.peerId;
    if (!callerId || !state.sdp) return;

    try {
      const pc = await createPeerConnection(callerId, state.callType || 'audio');

      const remoteOffer = new RTCSessionDescription(state.sdp);
      await pc.setRemoteDescription(remoteOffer);

      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);

      wsClient.send('call-answer', { sdp: answer }, callerId);
      dispatch({ type: 'SET_CONNECTED' });
    } catch (err) {
      console.error('Error al aceptar llamada:', err);
      cleanupCall();
      dispatch({ type: 'SET_ERROR', error: err.message });
    }
  }, [state.peerId, state.sdp, createPeerConnection, cleanupCall]);

  const rejectCall = useCallback(() => {
    wsClient.send('call-reject', {}, state.peerId);
    cleanupCall();
    dispatch({ type: 'RESET' });
  }, [state.peerId, cleanupCall]);

  const endCall = useCallback(() => {
    wsClient.send('call-end', {}, state.peerId);
    cleanupCall();
    dispatch({ type: 'RESET' });
  }, [state.peerId, cleanupCall]);

  const resetCall = useCallback(() => {
    cleanupCall();
    dispatch({ type: 'RESET' });
  }, [cleanupCall]);

  const value = useMemo(() => ({
    ...state,
    startCall,
    acceptCall,
    rejectCall,
    endCall,
    resetCall,
    audioDevices,
    selectedAudioDevice,
    setSelectedAudioDevice,
    isIdle: state.callState === CallState.IDLE,
    isCalling: state.callState === CallState.CALLING,
    isRinging: state.callState === CallState.RINGING,
    isConnected: state.callState === CallState.CONNECTED,
  }), [state, startCall, acceptCall, rejectCall, endCall, resetCall,
      audioDevices, selectedAudioDevice, setSelectedAudioDevice]);

  return (
    <CallContext.Provider value={value}>
      {children}
    </CallContext.Provider>
  );
}

export function useCall() {
  const ctx = useContext(CallContext);
  if (!ctx) throw new Error('useCall must be used within CallProvider');
  return ctx;
}