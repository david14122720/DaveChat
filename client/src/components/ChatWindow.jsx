import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Phone, Video, MoreVertical, Send, Smile, Paperclip, ArrowLeft, Loader2, Info } from 'lucide-react';
import { api } from '../lib/api';
import { useCall } from '../contexts/CallContext';
import { motion } from 'framer-motion';

const formatTime = (dateString) => {
  if (!dateString) return '';
  const date = new Date(dateString);
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
};

const isOnline = (contact) => contact?.status === 'online';

const formatPresence = (contact) => {
  if (isOnline(contact)) return 'Online';
  if (!contact?.last_seen) return 'Offline';
  const lastSeen = new Date(contact.last_seen);
  if (Number.isNaN(lastSeen.getTime())) return 'Offline';
  const diffMinutes = Math.max(1, Math.round((Date.now() - lastSeen.getTime()) / 60000));
  if (diffMinutes < 60) return 'Last seen ' + diffMinutes + 'm ago';
  const diffHours = Math.round(diffMinutes / 60);
  if (diffHours < 24) return 'Last seen ' + diffHours + 'h ago';
  return 'Last seen ' + lastSeen.toLocaleDateString([], { day: 'numeric', month: 'short' });
};

const ChatWindow = ({ contact, currentUser, onBack }) => {
  const [msg, setMsg] = useState('');
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [conversationId, setConversationId] = useState(null);
  const { startCall } = useCall();
  const scrollRef = useRef(null);
  const pollRef = useRef(null);
  const initializedRef = useRef(false);

  const normalizeMessages = useCallback((msgs) => {
    const arr = msgs || [];
    return [...arr].reverse();
  }, []);

  useEffect(() => {
    const initConv = async () => {
      try {
        const data = await api.getConversations();
        const convs = Array.isArray(data) ? data : (data.conversations || []);
        const existing = convs.find(c =>
          c.participants?.some(p => p.id === contact.id)
        );
        if (existing) {
          setConversationId(existing.id);
          return existing.id;
        }
        const created = await api.createConversation(contact.id);
        setConversationId(created.id);
        return created.id;
      } catch (err) {
        console.error('Error initializing conversation:', err);
        return null;
      }
    };
    initConv().then((convId) => {
      if (convId) {
        fetchMessages(convId);
      }
    });
  }, [contact.id]);

  const fetchMessages = async (convId) => {
    if (!convId) return;
    try {
      setLoading(true);
      const data = await api.getMessages(convId);
      setMessages(normalizeMessages(data.messages));
      initializedRef.current = true;
    } catch (err) {
      console.error('Error fetching messages:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!conversationId) return;

    pollRef.current = setInterval(async () => {
      try {
        const data = await api.getMessages(conversationId);
        if (data.messages?.length > 0) {
          setMessages(normalizeMessages(data.messages));
        }
      } catch (err) {
      }
    }, 2000);

    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [conversationId, normalizeMessages]);

  useEffect(() => {
    if (initializedRef.current && scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);

  const handleSendMessage = async (e) => {
    if (e) e.preventDefault();
    if (!msg.trim() || sending || !conversationId) return;

    setSending(true);
    try {
      await api.sendMessage(conversationId, msg.trim());
      setMsg('');
      const data = await api.getMessages(conversationId);
      setMessages(normalizeMessages(data.messages));
    } catch (err) {
      console.error('Error sending message:', err.message);
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="flex flex-col h-full w-full bg-slate-950">
      <header className="h-16 shrink-0 flex items-center justify-between px-3 md:px-4 bg-slate-900/80 backdrop-blur-md border-b border-slate-800 z-20">
        <div className="flex items-center gap-3 min-w-0">
          <button
            onClick={onBack}
            className="md:hidden p-2 -ml-1 hover:bg-slate-800 rounded-full transition-colors"
          >
            <ArrowLeft className="w-6 h-6 text-brand-light" />
          </button>

          <div className="relative shrink-0">
            {contact.avatar_url ? (
              <img
                src={contact.avatar_url.startsWith('http') || contact.avatar_url.startsWith('data:') ? contact.avatar_url : `${import.meta.env.VITE_API_URL || ''}${contact.avatar_url}`}
                alt={contact.username}
                className="w-10 h-10 md:w-11 md:h-11 rounded-full object-cover border border-slate-600 shadow-inner"
              />
            ) : (
              <div className="w-10 h-10 md:w-11 md:h-11 rounded-full bg-gradient-to-br from-slate-700 to-slate-800 border border-slate-600 flex items-center justify-center font-bold text-slate-200 shadow-inner">
                {contact.username[0].toUpperCase()}
              </div>
            )}
            <div className={'absolute bottom-0 right-0 w-3 h-3 border-2 border-slate-900 rounded-full ' + (isOnline(contact) ? 'bg-emerald-400' : 'bg-slate-600')}></div>
          </div>

          <div className="min-w-0">
            <h3 className="font-bold text-sm md:text-base text-slate-100 truncate">{contact.username}</h3>
            <p className={'text-[10px] md:text-xs font-medium uppercase tracking-wider ' + (isOnline(contact) ? 'text-emerald-300' : 'text-slate-500')}>{formatPresence(contact)}</p>
          </div>
        </div>

        <div className="flex items-center gap-1 md:gap-3">
          <button
            onClick={() => startCall(contact.id, 'video')}
            disabled={!isOnline(contact)}
            title={isOnline(contact) ? 'Start video call' : 'Contact is offline'}
            className="p-2 md:p-2.5 hover:bg-slate-800 rounded-xl transition-all group disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <Video className="w-5 h-5 md:w-6 md:h-6 text-slate-400 group-hover:text-brand-light" />
          </button>
          <button
            onClick={() => startCall(contact.id, 'audio')}
            disabled={!isOnline(contact)}
            title={isOnline(contact) ? 'Start audio call' : 'Contact is offline'}
            className="p-2 md:p-2.5 hover:bg-slate-800 rounded-xl transition-all group disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <Phone className="w-5 h-5 md:w-6 md:h-6 text-slate-400 group-hover:text-brand-light" />
          </button>
          <button className="p-2 md:p-2.5 hover:bg-slate-800 rounded-xl transition-all group">
            <MoreVertical className="w-5 h-5 md:w-6 md:h-6 text-slate-400 group-hover:text-brand-light" />
          </button>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto p-4 space-y-4 custom-scrollbar bg-[url('https://web.whatsapp.com/img/bg-chat-tile-dark_a4be512e71a95133d71dec5797616f1b.png')] bg-fixed bg-opacity-5">
        {loading ? (
          <div className="h-full flex items-center justify-center">
            <Loader2 className="w-8 h-8 text-brand-light animate-spin" />
          </div>
        ) : messages.length > 0 ? (
          messages.map((m, idx) => {
            const isMe = m.sender_id === currentUser.id;
            return (
              <motion.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                key={m.id || idx}
                className={'w-full flex ' + (isMe ? 'justify-end' : 'justify-start')}
              >
                <div className={'max-w-[85%] md:max-w-[70%] p-3 px-4 rounded-2xl shadow-sm relative ' +
                  (isMe
                    ? 'bg-brand-dark text-white rounded-tr-none border border-brand/30'
                    : 'bg-slate-900 text-slate-200 rounded-tl-none border border-slate-800')
                }>
                  <p className="text-sm md:text-base leading-relaxed">{m.content}</p>
                  <span className="text-[9px] mt-1 block opacity-50 text-right font-medium">
                    {formatTime(m.created_at)}
                  </span>
                </div>
              </motion.div>
            );
          })
        ) : (
          <div className="h-full flex flex-col items-center justify-center text-center p-8">
            <div className="w-20 h-20 bg-slate-900 rounded-full flex items-center justify-center mb-4 border border-slate-800 shadow-xl">
              <Info className="w-10 h-10 text-slate-600" />
            </div>
            <h3 className="text-slate-400 font-medium">Sin mensajes aún</h3>
            <p className="text-slate-600 text-sm mt-1 max-w-xs">
              Inicia la conversación con {contact.username}. Tus mensajes están cifrados de extremo a extremo.
            </p>
          </div>
        )}
        <div ref={scrollRef} />
      </div>

      <footer className="p-3 md:p-4 bg-slate-900/80 backdrop-blur-md border-t border-slate-800 shrink-0">
        <form onSubmit={handleSendMessage} className="flex items-center gap-2 md:gap-3 max-w-5xl mx-auto">
          <div className="flex-1 flex items-center gap-2 bg-slate-800/50 rounded-2xl px-3 md:px-4 py-1.5 md:py-2 border border-slate-700 focus-within:border-brand/50 transition-all">
            <Smile className="w-6 h-6 text-slate-500 cursor-pointer hidden sm:block hover:text-brand-light transition-colors" />
            <input
              type="text"
              value={msg}
              onChange={(e) => setMsg(e.target.value)}
              placeholder="Escribe un mensaje..."
              className="flex-1 bg-transparent border-none focus:ring-0 text-sm md:text-base py-2 outline-none text-white placeholder:text-slate-500"
            />
            <Paperclip className="w-6 h-6 text-slate-500 cursor-pointer hover:text-brand-light transition-colors" />
          </div>

          <button
            type="submit"
            disabled={!msg.trim() || sending}
            className="h-12 w-12 md:h-14 md:w-14 shrink-0 bg-brand-light hover:bg-brand text-slate-950 rounded-2xl flex items-center justify-center shadow-lg active:scale-95 transition-all disabled:opacity-50"
          >
            {sending ? <Loader2 className="w-6 h-6 animate-spin" /> : <Send className="w-6 h-6" />}
          </button>
        </form>
      </footer>
    </div>
  );
};

export default ChatWindow;
