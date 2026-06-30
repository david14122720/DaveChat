CREATE TABLE IF NOT EXISTS profiles (
    id CHAR(36) PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    status ENUM('online','offline','away') DEFAULT 'offline',
    last_seen DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
);

CREATE TABLE IF NOT EXISTS conversations (
    id CHAR(36) PRIMARY KEY,
    type ENUM('direct','group') DEFAULT 'direct',
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    last_message_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE TABLE IF NOT EXISTS participants (
    conversation_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    joined_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (conversation_id, user_id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS messages (
    id CHAR(36) PRIMARY KEY,
    conversation_id CHAR(36) NOT NULL,
    sender_id CHAR(36) NOT NULL,
    content TEXT NOT NULL,
    type ENUM('text','call') DEFAULT 'text',
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES profiles(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_conv_created ON messages(conversation_id, created_at);
CREATE INDEX idx_participants_user ON participants(user_id);
CREATE INDEX idx_conversations_last_msg ON conversations(last_message_at DESC);

-- Read receipts
CREATE TABLE IF NOT EXISTS read_receipts (
    message_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    read_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (message_id, user_id),
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Refresh tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_refresh_user (user_id),
    INDEX idx_refresh_hash (token_hash),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Call logs
CREATE TABLE IF NOT EXISTS call_logs (
    id CHAR(36) PRIMARY KEY,
    caller_id CHAR(36) NOT NULL,
    callee_id CHAR(36) NOT NULL,
    type ENUM('audio','video') NOT NULL,
    status ENUM('missed','completed','rejected','cancelled','busy') NOT NULL,
    started_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    ended_at DATETIME(3) NULL,
    duration_secs INT UNSIGNED DEFAULT 0,
    INDEX idx_caller (caller_id),
    INDEX idx_callee (callee_id),
    FOREIGN KEY (caller_id) REFERENCES profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (callee_id) REFERENCES profiles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Full-text search on messages
ALTER TABLE messages ADD FULLTEXT INDEX IF NOT EXISTS ft_messages_content (content);
