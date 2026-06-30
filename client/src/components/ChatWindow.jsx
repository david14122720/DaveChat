import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Phone, Video, MoreVertical, Send, Smile, Paperclip, Mic, ArrowLeft, Loader2 } from 'lucide-react';
import { api } from '../lib/api';

const ChatWindow = ({ contact, currentUser, onBack }) => {
  const [msg, setMsg] = useState('');
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [conversationId, setConversationId] = useState(null);
  const [lastCursor, setLastCursor] = useState(null);
  const scrollRef = useRef(null);
  const pollRef = useRef(null);

  // Find or create conversation
  useEffect(() => {
    const initConv = async () => {
      try {
        const data = await api.getConversations();
        const convs = data.conversations || [];
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
      setMessages(data.messages || []);
      setLastCursor(data.next_cursor || null);
    } catch (err) {
      console.error('Error fetching messages:', err);
    } finally {
      setLoading(false);
    }
  };

  // Poll for new messages every 2 seconds
  useEffect(() => {
    if (!conversationId) return;

    pollRef.current = setInterval(async () => {
      try {
        const data = await api.getMessages(conversationId);
        if (data.messages?.length > 0) {
          setMessages(data.messages || []);
          setLastCursor(data.next_cursor || null);
        }
      } catch (err) {
        // silent fail on poll
      }
    }, 2000);

    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, [conversationId]);

  // Auto-scroll
  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSendMessage = async (e) => {
    if (e) e.preventDefault();
    if (!msg.trim() || sending || !conversationId) return;

    setSending(true);
    try {
      await api.sendMessage(conversationId, msg.trim());
      setMsg('');
      // Refresh messages
      const data = await api.getMessages(conversationId);
      setMessages(data.messages || []);
    } catch (err) {
      console.error('Error sending message:', err.message);
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="flex flex-col h-full w-full bg-slate-950">
      {/* Header Optimizado: 64px de alto fijo */}
      <header className="h-16 shrink-0 flex items-center justify-between px-3 md:px-4 bg-slate-900 border-b border-slate-800 z-20">
        <div className="flex items-center gap-2 md:gap-3 min-w-0">
          {/* Botón Atrás solo en móvil */}
          <button
            onClick={onBack}
            className="md:hidden p-2 -ml-1 hover:bg-slate-800 rounded-full transition-colors"
          >
            <ArrowLeft className="w-6 h-6 text-brand-light" />
          </button>

          <div className="relative shrink-0">
            <div className="w-10 h-10 md:w-11 md:h-11 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-slate-300">
              {contact.username[0].toUpperCase()}
            </div>
            <div className="absolute bottom-0 right-0 w-3 h-3 bg-brand-light border-2 border-slate-900 rounded-full"></div>
          </div>

          <div className="min-w-0">
            <h3 className="font-bold text-sm md:text-base text-slate-100 truncate">{contact.username}</h3>
            <p className="text-[10px] md:text-xs text-brand-light">En línea</p>
          </div>
        </div>

        <div className="flex items-center gap-1 md:gap-3">
          <button className="p-2 md:p-2.5 hover:bg-slate-800 rounded-xl transition-all">
            <Video className="w-5 h-5 md:w-6 md:h-6 text-slate-400" />
          </button>
          <button className="p-2 md:p-2.5 hover:bg-slate-800 rounded-xl transition-all">
            <Phone className="w-5 h-5 md:w-6 md:h-6 text-slate-400" />
          </button>
          <button className="p-2 md:p-2.5 hover:bg-slate-800 rounded-xl transition-all">
            <MoreVertical className="w-5 h-5 md:w-6 md:h-6 text-slate-400" />
          </button>
        </div>
      </header>

      {/* Área de Mensajes: Flex-1 con scroll independiente */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4 custom-scrollbar bg-[url('https://web.whatsapp.com/img/bg-chat-tile-dark_a4be512e71a95133d71dec5797616f1b.png')] bg-opacity-5">
        {loading ? (
          <div className="h-full flex items-center justify-center">
            <Loader2 className="w-8 h-8 text-brand-light animate-spin" />
          </div>
        ) : messages.length > 0 ? (
          messages.map((m, idx) => {
            const isMe = m.sender_id === currentUser.id;
            return (
              <div key={m.id || idx} className={`flex ${isMe ? 'justify-end' : 'justify-start'}`}>
                <div className={`
                  max-w-[85%] md:max-w-[70%] p-3 px-4 rounded-2xl shadow-lg relative
                  ${isMe ? 'bg-brand-dark text-white rounded-tr-none' : 'bg-slate-900 text-slate-200 rounded-tl-none border border-slate-800'}
                `}>
                  <p className="text-sm md:text-base leading-relaxed">{m.content}</p>
                  <span className="text-[9px] mt-1 block opacity-50 text-right">
                    12:00 PM
                  </span>
                </div>
              </div>
            );
          })
        ) : (
          <div className="h-full flex items-center justify-center text-slate-600 text-sm italic">
            Sin mensajes aún.
          </div>
        )}
        <div ref={scrollRef} />
      </div>

      {/* Input de Mensaje: Altura adaptable, botones grandes */}
      <footer className="p-3 md:p-4 bg-slate-900 border-t border-slate-800 shrink-0">
        <form onSubmit={handleSendMessage} className="flex items-center gap-2 md:gap-3 max-w-5xl mx-auto">
          <div className="flex-1 flex items-center gap-2 bg-slate-800 rounded-2xl px-3 md:px-4 py-1.5 md:py-2 border border-slate-700 focus-within:border-brand/50 transition-all">
            <Smile className="w-6 h-6 text-slate-500 cursor-pointer hidden sm:block" />
            <input
              type="text"
              value={msg}
              onChange={(e) => setMsg(e.target.value)}
              placeholder="Escribe un mensaje..."
              className="flex-1 bg-transparent border-none focus:ring-0 text-sm md:text-base py-2 outline-none text-white"
            />
            <Paperclip className="w-6 h-6 text-slate-500 cursor-pointer" />
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
