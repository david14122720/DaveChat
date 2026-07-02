import React, { useState, useEffect, useRef } from 'react';
import { Search, MessageSquare, Settings, LogOut, Loader2, Wifi, WifiOff, Phone, Trash2 } from 'lucide-react';
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
  const [avatarUrl, setAvatarUrl] = useState(null);
  const [avatarUploading, setAvatarUploading] = useState(false);
  const [showProfileSettings, setShowProfileSettings] = useState(false);
  const fileInputRef = useRef(null);
  const settingsFileInputRef = useRef(null);

  // Initialize avatar from user prop on mount only
  // Subsequent updates come from upload/delete handlers (not user prop changes)
  useEffect(() => {
    setAvatarUrl(user?.avatar || user?.avatar_url || null);
  }, []);

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

  const handleAvatarChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    e.target.value = '';

    // Client-side validation
    const validTypes = ['image/jpeg', 'image/png', 'image/webp'];
    if (!validTypes.includes(file.type)) {
      alert('Only JPEG, PNG, and WebP images are accepted.');
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      alert('File size must be less than 5MB.');
      return;
    }

    setAvatarUploading(true);

    try {
      // Load image into an offscreen canvas and resize to max 400×400
      const image = await new Promise((resolve, reject) => {
        const img = new Image();
        img.onload = () => resolve(img);
        img.onerror = () => reject(new Error('Failed to load image'));
        img.src = URL.createObjectURL(file);
      });

      const maxSize = 400;
      let { width, height } = image;
      if (width > maxSize || height > maxSize) {
        const ratio = Math.min(maxSize / width, maxSize / height);
        width = Math.round(width * ratio);
        height = Math.round(height * ratio);
      }

      const canvas = document.createElement('canvas');
      canvas.width = width;
      canvas.height = height;
      const ctx = canvas.getContext('2d');
      ctx.drawImage(image, 0, 0, width, height);

      const blob = await new Promise((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.9));
      if (!blob) throw new Error('Failed to create image blob');

      const formData = new FormData();
      formData.append('avatar', blob, 'avatar.jpg');

      const result = await api.uploadAvatar(formData);
      setAvatarUrl(result.avatar_url);
    } catch (err) {
      console.error('Avatar upload error:', err);
      alert(err.message || 'Failed to upload avatar');
    } finally {
      setAvatarUploading(false);
    }
  };

  const handleDeleteAvatar = async () => {
    if (!confirm('Remove your profile picture?')) return;
    try {
      const result = await api.deleteAvatar();
      setAvatarUrl(null);
    } catch (err) {
      console.error('Avatar delete error:', err);
      alert(err.message || 'Failed to delete avatar');
    }
  };

  const filteredContacts = contacts.filter(c =>
    c.username.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <>
      <div className="flex w-screen h-[100dvh] bg-slate-950 text-white overflow-hidden fixed inset-0">
      <aside className={
        (selectedContact ? 'hidden md:flex' : 'flex') +
        ' w-full md:w-80 lg:w-96 flex-col border-r border-slate-800 bg-slate-900/50 backdrop-blur-xl shrink-0'
      }>
        <div className="h-16 flex items-center justify-between px-4 bg-slate-900/80 border-b border-slate-800 shrink-0 z-10">
          <div className="flex items-center gap-3">
            <div className="relative group shrink-0">
              <input
                type="file"
                accept="image/*"
                ref={fileInputRef}
                onChange={handleAvatarChange}
                className="hidden"
              />
              {avatarUploading ? (
                <div className="w-10 h-10 rounded-full bg-gradient-to-br from-brand-dark to-slate-800 flex items-center justify-center border border-brand/30 shadow-sm">
                  <Loader2 className="w-5 h-5 animate-spin text-brand-light" />
                </div>
              ) : avatarUrl ? (
                <div className="relative" onClick={() => fileInputRef.current?.click()}>
                  <img
                    src={avatarUrl.startsWith('http') ? avatarUrl : `${import.meta.env.VITE_API_URL || ''}${avatarUrl}`}
                    alt="Avatar"
                    className="w-10 h-10 rounded-full object-cover border border-brand/30 shadow-sm cursor-pointer"
                  />
                  <div
                    className="absolute inset-0 rounded-full bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center cursor-pointer"
                    onClick={(e) => { e.stopPropagation(); handleDeleteAvatar(); }}
                  >
                    <Trash2 className="w-4 h-4 text-red-400" />
                  </div>
                </div>
              ) : (
                <div
                  className="w-10 h-10 rounded-full bg-gradient-to-br from-brand-dark to-slate-800 flex items-center justify-center font-bold text-brand-light border border-brand/30 shadow-sm cursor-pointer hover:opacity-80 transition-opacity"
                  onClick={() => fileInputRef.current?.click()}
                >
                  {user?.username?.[0] || user?.email?.[0]?.toUpperCase()}
                </div>
              )}
            </div>
            <div className="flex flex-col">
              <span className="font-semibold text-sm text-slate-200">{user?.username || user?.email?.split('@')[0]}</span>
              <span className="text-[10px] text-brand-light font-medium uppercase tracking-wider">Connected</span>
            </div>
          </div>
          <div className="flex gap-3 text-slate-400">
            <Settings className="w-5 h-5 cursor-pointer hover:text-white transition-colors" onClick={() => setShowProfileSettings(true)} />
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

      {/* Profile Settings Modal */}
      {showProfileSettings && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowProfileSettings(false)}>
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            className="bg-slate-900 border border-slate-800 rounded-3xl p-8 w-full max-w-sm mx-4 shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-lg font-bold text-slate-200 mb-6">Profile Settings</h2>

            <div className="flex flex-col items-center gap-4 mb-6">
              <div className="relative group">
                <div className="w-24 h-24 rounded-full overflow-hidden border-2 border-brand/30 shadow-lg">
                  {avatarUrl ? (
                    <img
                      src={avatarUrl.startsWith('http') ? avatarUrl : `${import.meta.env.VITE_API_URL || ''}${avatarUrl}`}
                      alt="Avatar"
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <div className="w-full h-full bg-gradient-to-br from-brand-dark to-slate-800 flex items-center justify-center font-bold text-3xl text-brand-light">
                      {user?.username?.[0] || user?.email?.[0]?.toUpperCase()}
                    </div>
                  )}
                </div>
              </div>

              <div className="flex gap-3">
                <button
                  onClick={() => fileInputRef.current?.click()}
                  disabled={avatarUploading}
                  className="px-4 py-2 text-sm font-medium bg-brand-light text-white rounded-xl hover:opacity-90 transition-opacity disabled:opacity-50 flex items-center gap-2"
                >
                  {avatarUploading ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
                  )}
                  {avatarUploading ? 'Uploading...' : avatarUrl ? 'Change Photo' : 'Upload Photo'}
                </button>
                {avatarUrl && (
                  <button
                    onClick={handleDeleteAvatar}
                    className="px-4 py-2 text-sm font-medium bg-red-500/10 text-red-400 border border-red-500/20 rounded-xl hover:bg-red-500/20 transition-colors flex items-center gap-2"
                  >
                    <Trash2 className="w-4 h-4" />
                    Remove
                  </button>
                )}
              </div>
            </div>

            <div className="space-y-3 border-t border-slate-800 pt-4">
              <div>
                <label className="text-xs text-slate-500 font-medium uppercase tracking-wider">Username</label>
                <p className="text-sm text-slate-200 mt-1">{user?.username}</p>
              </div>
              <div>
                <label className="text-xs text-slate-500 font-medium uppercase tracking-wider">Email</label>
                <p className="text-sm text-slate-200 mt-1">{user?.email}</p>
              </div>
            </div>

            <button
              onClick={() => setShowProfileSettings(false)}
              className="mt-6 w-full py-3 text-sm font-medium bg-slate-800 text-slate-300 rounded-xl hover:bg-slate-700 transition-colors"
            >
              Close
            </button>
          </motion.div>
        </div>
      )}
    </>
  );
};

export default MainLayout;
