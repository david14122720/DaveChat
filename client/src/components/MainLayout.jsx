import React, { useState, useEffect } from 'react';
import { Search, MoreVertical, MessageSquare, Settings, LogOut, Loader2, Menu } from 'lucide-react';
import { api } from '../lib/api';
import ChatWindow from './ChatWindow';

const MainLayout = ({ user, onLogout }) => {
  const [contacts, setContacts] = useState([]);
  const [selectedContact, setSelectedContact] = useState(null);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    fetchProfiles();
  }, []);

  const fetchProfiles = async () => {
    try {
      setLoading(true);
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
      
      {/* SIDEBAR: 100% en móvil, ancho fijo en desktop */}
      <aside className={`
        ${selectedContact ? 'hidden md:flex' : 'flex'} 
        w-full md:w-80 lg:w-96 flex-col border-r border-slate-800 bg-slate-900/50 backdrop-blur-xl shrink-0
      `}>
        {/* Header Perfil */}
        <div className="h-16 flex items-center justify-between px-4 bg-slate-900/80 border-b border-slate-800 shrink-0">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-brand-dark flex items-center justify-center font-bold text-brand-light border border-brand/20">
              {user?.username?.[0] || user?.email?.[0]?.toUpperCase()}
            </div>
            <div className="flex flex-col">
              <span className="font-semibold text-sm">{user?.username || user?.email?.split('@')[0]}</span>
              <span className="text-[10px] text-brand-light">En línea</span>
            </div>
          </div>
          <div className="flex gap-4 text-slate-400">
            <Settings className="w-6 h-6 cursor-pointer hover:text-white transition-colors" />
            <LogOut className="w-6 h-6 cursor-pointer hover:text-red-400 transition-colors" onClick={onLogout} />
          </div>
        </div>

        {/* Buscador */}
        <div className="p-4 shrink-0">
          <div className="relative group">
            <input 
              type="text" 
              placeholder="Buscar contactos..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-slate-800/50 border border-slate-700 rounded-2xl py-3 pl-11 pr-4 text-sm focus:ring-2 focus:ring-brand-light outline-none transition-all"
            />
            <Search className="absolute left-4 top-3.5 w-4 h-4 text-slate-500 group-focus-within:text-brand-light transition-colors" />
          </div>
        </div>

        {/* Lista de Chats */}
        <div className="flex-1 overflow-y-auto overflow-x-hidden custom-scrollbar">
          {loading ? (
            <div className="flex flex-col items-center justify-center h-40 gap-3">
              <Loader2 className="w-6 h-6 animate-spin text-brand-light" />
              <span className="text-xs text-slate-500">Cargando...</span>
            </div>
          ) : filteredContacts.length > 0 ? (
            filteredContacts.map(contact => (
              <div 
                key={contact.id}
                onClick={() => setSelectedContact(contact)}
                className={`
                  p-4 flex items-center gap-4 cursor-pointer transition-all border-b border-slate-800/30
                  hover:bg-slate-800/50 
                  ${selectedContact?.id === contact.id ? 'bg-brand/10 md:bg-slate-800/80 border-r-4 md:border-r-0 md:border-l-4 border-brand-light' : ''}
                `}
              >
                <div className="relative shrink-0">
                  <div className="w-12 h-12 rounded-2xl bg-slate-800 border border-slate-700 flex items-center justify-center font-bold text-slate-300">
                    {contact.username[0].toUpperCase()}
                  </div>
                  <div className="absolute -bottom-1 -right-1 w-3.5 h-3.5 bg-brand-light border-2 border-slate-900 rounded-full shadow-lg"></div>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex justify-between items-baseline">
                    <h3 className="font-medium text-slate-200 truncate">{contact.username}</h3>
                    <span className="text-[10px] text-slate-500">hace 5m</span>
                  </div>
                  <p className="text-xs text-slate-500 truncate mt-0.5">Toca para iniciar chat</p>
                </div>
              </div>
            ))
          ) : (
            <div className="p-8 text-center text-slate-600">No hay contactos</div>
          )}
        </div>
      </aside>

      {/* CHAT AREA: 100% en móvil cuando está abierto */}
      <main className={`
        ${!selectedContact ? 'hidden md:flex' : 'flex'} 
        flex-1 flex-col bg-slate-950 relative w-full h-full
      `}>
        {selectedContact ? (
          <ChatWindow 
            contact={selectedContact} 
            currentUser={user} 
            onBack={() => setSelectedContact(null)} 
          />
        ) : (
          <div className="hidden md:flex flex-1 flex-col items-center justify-center text-slate-600">
            <div className="w-24 h-24 bg-slate-900 rounded-full flex items-center justify-center mb-6 border border-slate-800">
              <MessageSquare className="w-10 h-10 text-brand/10" />
            </div>
            <h2 className="text-xl font-bold text-slate-400">DaveChat Premium</h2>
            <p className="mt-2 text-sm text-slate-500">Selecciona un chat para empezar a escribir.</p>
          </div>
        )}
      </main>
    </div>
  );
};

export default MainLayout;
