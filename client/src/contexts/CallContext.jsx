import { createContext, useContext, useReducer, useEffect, useState, useCallback, useRef } from 'react';
import { wsClient } from '../lib/websocket';

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

  const getIceConfig = useCallback(() => ({
    iceServers: [
      { urls: 'stun:stun.l.google.com:19302' },
      { urls: 'stun:stun1.l.google.com:19302' },
    ],
  }), []);

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

  const createPeerConnection = useCallback(async (remoteUserId, callType = 'audio') => {
    const pc = new RTCPeerConnection(getIceConfig());
    pcRef.current = pc;

    try {
      const audioConstraints = {
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true,
      };
      if (selectedAudioDevice) {
        audioConstraints.deviceId = { exact: selectedAudioDevice };
      }
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: audioConstraints,
        video: callType === 'video',
      });
      localStreamRef.current = stream;
      dispatch({ type: 'SET_LOCAL_STREAM', stream });

      stream.getTracks().forEach(track => {
        pc.addTrack(track, stream);
      });
    } catch (err) {
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

    pc.ontrack = (event) => {
      console.log('📡 Track remoto recibido');
      remoteStreamRef.current = event.streams[0];
      dispatch({ type: 'SET_REMOTE_STREAM', stream: event.streams[0] });
    };

    pc.onicecandidate = (event) => {
      if (event.candidate) {
        console.log('🧊 Enviando ICE candidate');
        wsClient.send('ice-candidate', { candidate: event.candidate.toJSON() }, remoteUserId);
      }
    };

    pc.onconnectionstatechange = () => {
      console.log('🔗 Conexión WebRTC state:', pc.connectionState);
      if (pc.connectionState === 'disconnected' || pc.connectionState === 'failed') {
        console.log('❌ Conexión WebRTC perdida');
        cleanupCall();
        dispatch({ type: 'SET_ERROR', error: 'Conexión perdida' });
      }
    };

    pc.oniceconnectionstatechange = () => {
      console.log('🧊 ICE state:', pc.iceConnectionState);
    };

    return pc;
  }, [getIceConfig, cleanupCall, selectedAudioDevice]);

  useEffect(() => {
    wsClient.connect();

    const unsubscribes = [];

    unsubscribes.push(wsClient.on('call-offer', (msg) => {
      console.log('📞 LLAMADA ENTRANTE de', msg.from, msg.call_type);
      dispatch({ type: 'SET_RINGING', from: msg.from, callType: msg.call_type, sdp: msg.payload?.sdp });
    }));

    unsubscribes.push(wsClient.on('call-answer', (msg) => {
      console.log('📞 LLAMADA ACEPTADA por', msg.from);
      const pc = pcRef.current;
      if (pc && msg.payload?.sdp) {
        const remoteAnswer = new RTCSessionDescription(msg.payload.sdp);
        pc.setRemoteDescription(remoteAnswer)
          .then(() => console.log('✅ Remote description set'))
          .catch(err => console.error('Error setting remote description:', err));
      }
      dispatch({ type: 'SET_CONNECTED' });
    }));

    unsubscribes.push(wsClient.on('ice-candidate', (msg) => {
      console.log('🧊 ICE candidate recibido de', msg.from);
      const pc = pcRef.current;
      if (pc && msg.payload?.candidate) {
        pc.addIceCandidate(new RTCIceCandidate(msg.payload.candidate))
          .catch(err => console.error('Error adding ICE candidate:', err));
      }
    }));

    unsubscribes.push(wsClient.on('call-busy', () => {
      console.log('📞 DESTINO OCUPADO');
      cleanupCall();
      dispatch({ type: 'SET_BUSY' });
    }));

    unsubscribes.push(wsClient.on('call-reject', (msg) => {
      console.log('📞 LLAMADA RECHAZADA por', msg.from);
      cleanupCall();
      dispatch({ type: 'SET_REJECTED', from: msg.from });
    }));

    unsubscribes.push(wsClient.on('call-timeout', () => {
      console.log('⏰ TIMEOUT — no contestó');
      cleanupCall();
      dispatch({ type: 'SET_TIMEOUT' });
    }));

    unsubscribes.push(wsClient.on('call-started', () => {
      console.log('✅ LLAMADA INICIADA');
    }));

    unsubscribes.push(wsClient.on('call-end', (msg) => {
      console.log('📞 LLAMADA FINALIZADA por', msg.from);
      cleanupCall();
      dispatch({ type: 'SET_ENDED' });
    }));

    unsubscribes.push(wsClient.on('close', () => {
      if (state.callState !== CallState.IDLE) {
        console.log('⚠️ WS desconectado durante llamada');
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
      console.log('📞 Iniciando llamada ' + callType + ' con ' + targetUserId + '...');
      dispatch({ type: 'SET_CALLING', peerId: targetUserId, callType });

      const pc = await createPeerConnection(targetUserId, callType);
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);

      console.log('📤 Enviando offer SDP');
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
      console.log('📞 Aceptando llamada...');

      const pc = await createPeerConnection(callerId, state.callType || 'audio');

      const remoteOffer = new RTCSessionDescription(state.sdp);
      await pc.setRemoteDescription(remoteOffer);

      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);

      console.log('📤 Enviando answer SDP');
      wsClient.send('call-answer', { sdp: answer }, callerId);
      dispatch({ type: 'SET_CONNECTED' });
    } catch (err) {
      console.error('Error al aceptar llamada:', err);
      cleanupCall();
      dispatch({ type: 'SET_ERROR', error: err.message });
    }
  }, [state.peerId, state.sdp, createPeerConnection, cleanupCall]);

  const rejectCall = useCallback(() => {
    console.log('📞 Rechazando llamada...');
    wsClient.send('call-reject', {}, state.peerId);
    cleanupCall();
    dispatch({ type: 'RESET' });
  }, [state.peerId, cleanupCall]);

  const endCall = useCallback(() => {
    console.log('📞 Colgando llamada...');
    wsClient.send('call-end', {}, state.peerId);
    cleanupCall();
    dispatch({ type: 'RESET' });
  }, [state.peerId, cleanupCall]);

  const resetCall = useCallback(() => {
    cleanupCall();
    dispatch({ type: 'RESET' });
  }, [cleanupCall]);

  const value = {
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
  };

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
