import React, { useState, useEffect } from 'react';
import { Search, MessageSquare, Settings, LogOut, Loader2, Wifi, WifiOff, Phone } from 'lucide-react';
import { motion } from 'framer-motion';
import { api } from '../lib/api';
import ChatWindow from './ChatWindow';
import CallHistory from './CallHistory';

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

const MainLayout = ({ user, onLogout }) => {
  const [contacts, setContacts] = useState([]);
  const [selectedContact, setSelectedContact] = useState(null);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [activeTab, setActiveTab] = useState('chats');

  useEffect(() => {
    fetchProfiles();
    const interval = setInterval(fetchProfiles, 15000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (!selectedContact) return;
    const freshContact = contacts.find((contact) => contact.id === selectedContact.id);
    if (freshContact) setSelectedContact(freshContact);
  }, [contacts, selectedContact?.id]);

  const fetchProfiles = async () => {
    try {
      const data = await api.getProfiles();
      setContacts(data || []);
    } catch (error) {
      console.error('Error fetching profiles:', error.message);
    } finally {
      setLoading(false);
    }
  };

  const filteredContacts = contacts.filter(c =>
    c.username.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="flex w-screen h-[100dvh] bg-slate-950 text-white overflow-hidden fixed inset-0">

      <aside className={
        (selectedContact ? 'hidden md:flex' : 'flex') +
        ' w-full md:w-80 lg:w-96 flex-col border-r border-slate-800 bg-slate-900/50 backdrop-blur-xl shrink-0'
      }>
        <div className="h-16 flex items-center justify-between px-4 bg-slate-900/80 border-b border-slate-800 shrink-0 z-10">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-gradient-to-br from-brand-dark to-slate-800 flex items-center justify-center font-bold text-brand-light border border-brand/30 shadow-sm">
              {user?.username?.[0] || user?.email?.[0]?.toUpperCase()}
            </div>
            <div className="flex flex-col">
              <span className="font-semibold text-sm text-slate-200">{user?.username || user?.email?.split('@')[0]}</span>
              <span className="text-[10px] text-brand-light font-medium uppercase tracking-wider">Connected</span>
            </div>
          </div>
          <div className="flex gap-3 text-slate-400">
            <Settings className="w-5 h-5 cursor-pointer hover:text-white transition-colors" />
            <LogOut className="w-5 h-5 cursor-pointer hover:text-red-400 transition-colors" onClick={onLogout} />
          </div>
        </div>

        <div className="flex gap-2 px-4 py-3 border-b border-slate-800 shrink-0">
          <button
            onClick={() => setActiveTab('chats')}
            className={'flex-1 py-2 px-3 rounded-xl text-sm font-medium transition-all ' +
              (activeTab === 'chats'
                ? 'bg-brand-light text-white shadow-lg shadow-brand-light/20'
                : 'text-slate-400 hover:text-white hover:bg-slate-800/50')
            }
          >
            <MessageSquare className="w-4 h-4 inline mr-1.5 -mt-0.5" />
            Chats
          </button>
          <button
            onClick={() => setActiveTab('calls')}
            className={'flex-1 py-2 px-3 rounded-xl text-sm font-medium transition-all ' +
              (activeTab === 'calls'
                ? 'bg-brand-light text-white shadow-lg shadow-brand-light/20'
                : 'text-slate-400 hover:text-white hover:bg-slate-800/50')
            }
          >
            <Phone className="w-4 h-4 inline mr-1.5 -mt-0.5" />
            Llamadas
          </button>
        </div>

        {activeTab === 'chats' ? (
          <>
            <div className="p-4 shrink-0 space-y-3">
              <div className="flex items-center justify-between rounded-2xl border border-slate-800 bg-slate-900 px-4 py-3 text-xs text-slate-400">
                <span>{contacts.filter(isOnline).length} online</span>
                <span>{contacts.length} contacts</span>
              </div>
              <div className="relative group">
                <input
                  type="text"
                  placeholder="Search contacts..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full bg-slate-800/50 border border-slate-700 rounded-2xl py-3 pl-11 pr-4 text-sm focus:ring-2 focus:ring-brand-light focus:border-transparent outline-none transition-all placeholder:text-slate-500 group-hover:border-slate-600"
                />
                <Search className="absolute left-4 top-3.5 w-4 h-4 text-slate-500 group-focus-within:text-brand-light transition-colors" />
              </div>
            </div>

            <div className="flex-1 overflow-y-auto overflow-x-hidden custom-scrollbar">
              {loading ? (
                <div className="flex flex-col items-center justify-center h-40 gap-3">
                  <Loader2 className="w-6 h-6 animate-spin text-brand-light" />
                  <span className="text-xs text-slate-500 font-medium">Loading contacts...</span>
                </div>
              ) : filteredContacts.length > 0 ? (
                <div className="p-2 space-y-1">
                  {filteredContacts.map(contact => (
                    <motion.div
                      whileHover={{ scale: 1.01 }}
                      key={contact.id}
                      onClick={() => setSelectedContact(contact)}
                      className={'p-3 flex items-center gap-4 cursor-pointer transition-all rounded-2xl border ' +
                        (selectedContact?.id === contact.id ? 'bg-brand/10 text-white border-brand/30' : 'border-transparent text-slate-300 hover:bg-slate-800/50')
                      }
                    >
                      <div className="relative shrink-0">
                        <div className="w-12 h-12 rounded-2xl bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-slate-300 shadow-sm">
                          {contact.username[0].toUpperCase()}
                        </div>
                        <div className={'absolute -bottom-1 -right-1 w-3.5 h-3.5 border-2 border-slate-900 rounded-full shadow-lg ' + (isOnline(contact) ? 'bg-emerald-400' : 'bg-slate-600')}></div>
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex justify-between items-center gap-3">
                          <h3 className="font-medium truncate">{contact.username}</h3>
                          {isOnline(contact) ? <Wifi className="w-3.5 h-3.5 text-emerald-400" /> : <WifiOff className="w-3.5 h-3.5 text-slate-600" />}
                        </div>
                        <p className={'text-xs truncate mt-0.5 ' + (isOnline(contact) ? 'text-emerald-300/80' : 'text-slate-500')}>{formatPresence(contact)}</p>
                      </div>
                    </motion.div>
                  ))}
                </div>
              ) : (
                <div className="p-8 text-center text-slate-600">
                  <p className="text-sm italic">No contacts found</p>
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 overflow-hidden">
            <CallHistory user={user} />
          </div>
        )}
      </aside>

      <main className={(!selectedContact ? 'hidden md:flex' : 'flex') + ' flex-1 flex-col bg-slate-950 relative w-full h-full'}>
        {selectedContact ? (
          <ChatWindow
            contact={selectedContact}
            currentUser={user}
            onBack={() => setSelectedContact(null)}
          />
        ) : (
          <div className="hidden md:flex flex-1 flex-col items-center justify-center text-slate-600 p-8 text-center">
            <motion.div
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              className="w-24 h-24 bg-slate-900 rounded-full flex items-center justify-center mb-6 border border-slate-800 shadow-2xl"
            >
              <MessageSquare className="w-10 h-10 text-brand/20" />
            </motion.div>
            <h2 className="text-2xl font-bold text-slate-400">DaveChat Premium</h2>
            <p className="mt-2 text-sm text-slate-500 max-w-xs leading-relaxed">
              Welcome back. Choose a contact to start chatting.
            </p>
          </div>
        )}
      </main>
    </div>
  );
};

export default MainLayout;
