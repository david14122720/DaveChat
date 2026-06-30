import React from 'react';
import { PhoneOff, Mic, MicOff, Video, VideoOff, Maximize2, Minimize2 } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';

const VideoOverlay = ({ type, contact, onHangup }) => {
  return (
    <AnimatePresence>
      <motion.div 
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        exit={{ opacity: 0, scale: 0.9 }}
        className="absolute inset-0 z-50 bg-slate-950 flex flex-col items-center justify-center"
      >
        {/* Remote Stream Container */}
        <div className="relative w-full h-full flex items-center justify-center overflow-hidden">
          {/* Placeholder for Remote Video */}
          <div className="absolute inset-0 bg-gradient-to-br from-slate-800 to-slate-900 flex flex-col items-center justify-center text-slate-500">
            <div className="w-32 h-32 rounded-full bg-slate-700 flex items-center justify-center text-4xl font-bold mb-4 shadow-2xl">
              {contact.name[0]}
            </div>
            <h2 className="text-2xl font-bold text-white">{contact.name}</h2>
            <p className="text-brand-light animate-pulse mt-2">Llamada de {type === 'video' ? 'video' : 'voz'}...</p>
          </div>

          {/* Local Stream (PIP) */}
          <motion.div 
            drag
            dragConstraints={{ left: -300, right: 300, top: -200, bottom: 200 }}
            className="absolute top-6 right-6 w-32 md:w-48 aspect-video bg-slate-800 rounded-2xl border-2 border-brand-light shadow-2xl overflow-hidden cursor-move z-10"
          >
            <div className="w-full h-full flex items-center justify-center bg-slate-700 text-slate-500 text-xs">
              Tu cámara
            </div>
          </motion.div>

          {/* Controls Overlay */}
          <div className="absolute bottom-12 flex items-center gap-6 p-6 bg-slate-900/60 backdrop-blur-xl rounded-full border border-slate-700/50 shadow-2xl transition-all hover:bg-slate-900/80">
            <button className="p-4 bg-slate-800 hover:bg-slate-700 rounded-full transition-colors group">
              <MicOff className="w-6 h-6 text-white group-hover:text-brand-light" />
            </button>
            
            {type === 'video' && (
              <button className="p-4 bg-slate-800 hover:bg-slate-700 rounded-full transition-colors group">
                <VideoOff className="w-6 h-6 text-white group-hover:text-brand-light" />
              </button>
            )}

            <button 
              onClick={onHangup}
              className="p-5 bg-red-500 hover:bg-red-600 rounded-full transition-all hover:rotate-12 transform shadow-lg shadow-red-500/30"
            >
              <PhoneOff className="w-7 h-7 text-white" />
            </button>

            <button className="p-4 bg-slate-800 hover:bg-slate-700 rounded-full transition-colors">
              <Maximize2 className="w-6 h-6 text-white" />
            </button>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  );
};

export default VideoOverlay;
