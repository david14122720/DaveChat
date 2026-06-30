import React, { useState, useEffect, useCallback } from 'react';
import {
  Phone, PhoneIncoming, PhoneOutgoing,
  PhoneOff, Video,
  Loader2, RefreshCw
} from 'lucide-react';
import { api } from '../lib/api';

function formatRelativeTime(dateStr) {
  const now = new Date();
  const date = new Date(dateStr);
  const diffMs = now - date;
  const diffMin = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMin < 1) return 'Ahora';
  if (diffMin < 60) return 'hace ' + diffMin + ' min';
  if (diffHours < 24) return 'hace ' + diffHours + 'h';
  if (diffDays === 1) return 'Ayer';
  return 'hace ' + diffDays + ' días';
}

function formatDuration(secs) {
  if (!secs || secs <= 0) return null;
  const min = Math.floor(secs / 60);
  const s = secs % 60;
  return min + ':' + s.toString().padStart(2, '0');
}

const STATUS_CONFIG = {
  completed: { dot: 'bg-green-500', label: null, labelColor: null },
  missed: { dot: 'bg-red-500', label: 'Perdida', labelColor: 'text-red-400' },
  rejected: { dot: 'bg-orange-500', label: 'Rechazada', labelColor: 'text-orange-400' },
  cancelled: { dot: 'bg-gray-500', label: 'Cancelada', labelColor: 'text-slate-400' },
  busy: { dot: 'bg-yellow-500', label: 'Ocupado', labelColor: 'text-yellow-400' },
};

const CallHistory = ({ user }) => {
  const [calls, setCalls] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [refreshing, setRefreshing] = useState(false);

  const fetchCalls = useCallback(async (isRefresh = false) => {
    try {
      if (isRefresh) setRefreshing(true);
      const data = await api.getCallLogs();
      setCalls(Array.isArray(data) ? data : []);
      setError(null);
    } catch (err) {
      console.error('Error fetching call logs:', err.message);
      setError(err.message);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchCalls();
  }, [fetchCalls]);

  const getDirectionIcon = (call) => {
    const isOutgoing = String(call.caller_id) === String(user?.id);
    if (isOutgoing) {
      return <PhoneOutgoing className="w-5 h-5 text-green-400 shrink-0" />;
    }
    return <PhoneIncoming className="w-5 h-5 text-blue-400 shrink-0" />;
  };

  const getPeerLabel = (call) => {
    const isOutgoing = String(call.caller_id) === String(user?.id);
    if (isOutgoing) return 'Llamada a ' + call.callee_id;
    return 'Llamada de ' + call.caller_id;
  };

  const getCallTypeIcon = (call) => {
    if (call.type === 'video') return <Video className="w-3.5 h-3.5 text-slate-400" />;
    return <Phone className="w-3.5 h-3.5 text-slate-400" />;
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-40 gap-3">
        <Loader2 className="w-6 h-6 animate-spin text-brand-light" />
        <span className="text-xs text-slate-500 font-medium">Cargando llamadas...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-40 gap-3 p-4 text-center">
        <PhoneOff className="w-8 h-8 text-red-400" />
        <span className="text-sm text-slate-400">Error al cargar llamadas</span>
        <button
          onClick={() => fetchCalls(true)}
          className="text-xs text-brand-light hover:underline transition-colors"
        >
          Intentar de nuevo
        </button>
      </div>
    );
  }

  if (calls.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-40 gap-3 p-4 text-center">
        <Phone className="w-10 h-10 text-slate-600" />
        <span className="text-sm text-slate-500">Sin llamadas recientes</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 py-2 shrink-0">
        <span className="text-xs text-slate-500 font-medium">
          {calls.length} {calls.length === 1 ? 'llamada' : 'llamadas'}
        </span>
        <button
          onClick={() => fetchCalls(true)}
          disabled={refreshing}
          className="p-1.5 rounded-lg hover:bg-slate-800/50 text-slate-400 hover:text-brand-light transition-colors disabled:opacity-50"
          title="Actualizar"
        >
          <RefreshCw className={'w-4 h-4 ' + (refreshing ? 'animate-spin' : '')} />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto overflow-x-hidden custom-scrollbar">
        {calls.map((call) => {
          const statusCfg = STATUS_CONFIG[call.status] || STATUS_CONFIG.completed;
          const duration = formatDuration(call.duration_secs);
          const isCompleted = call.status === 'completed';

          return (
            <div
              key={call.id}
              className="flex items-start gap-3 px-4 py-3 hover:bg-slate-800/30 transition-colors cursor-pointer border-b border-slate-800/30 last:border-b-0"
            >
              <div className="mt-0.5 shrink-0">
                {getDirectionIcon(call)}
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-medium text-white truncate">
                    {getPeerLabel(call)}
                  </span>
                  <span className="shrink-0 text-xs text-slate-500 whitespace-nowrap">
                    {formatRelativeTime(call.started_at)}
                  </span>
                </div>

                <div className="flex items-center gap-1.5 mt-1">
                  {getCallTypeIcon(call)}

                  <span className="text-[10px] text-slate-500 uppercase tracking-wider">
                    {call.type === 'video' ? 'Video' : 'Audio'}
                  </span>

                  <span className="text-slate-600">·</span>

                  {isCompleted && duration ? (
                    <span className="text-xs text-slate-400">{duration}</span>
                  ) : statusCfg.label ? (
                    <span className={'text-xs ' + (statusCfg.labelColor || 'text-slate-400')}>
                      {statusCfg.label}
                    </span>
                  ) : null}

                  <span className={'w-1.5 h-1.5 rounded-full ' + statusCfg.dot + ' shrink-0'} />
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default CallHistory;
