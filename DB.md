# DaveChat — Base de Datos MariaDB

Este documento define el esquema de base de datos para DaveChat, que corre en MariaDB 11.4 vía Docker.

## Esquema

### Tabla: `profiles`
| Columna | Tipo | Descripción |
|---------|------|-------------|
| `id` | CHAR(36) PK | UUID v4 del usuario |
| `username` | VARCHAR(50) UNIQUE | Nombre de usuario único |
| `email` | VARCHAR(255) UNIQUE | Email único (usado para login) |
| `password_hash` | VARCHAR(255) | Hash bcrypt de la contraseña |
| `avatar_url` | TEXT | URL del avatar |
| `status` | ENUM('online','offline','away') | Estado de presencia |
| `last_seen` | DATETIME(3) | Última actividad |
| `created_at` | DATETIME(3) | Fecha de registro |
| `updated_at` | DATETIME(3) | Última actualización |

### Tabla: `conversations`
| Columna | Tipo | Descripción |
|---------|------|-------------|
| `id` | CHAR(36) PK | UUID v4 |
| `type` | ENUM('direct','group') | Tipo de conversación |
| `created_at` | DATETIME(3) | Fecha de creación |
| `last_message_at` | DATETIME(3) | Fecha del último mensaje |

### Tabla: `participants`
| Columna | Tipo | Descripción |
|---------|------|-------------|
| `conversation_id` | CHAR(36) FK | Referencia a conversations |
| `user_id` | CHAR(36) FK | Referencia a profiles |
| `joined_at` | DATETIME(3) | Fecha de ingreso |
| PK | (conversation_id, user_id) | |

### Tabla: `messages`
| Columna | Tipo | Descripción |
|---------|------|-------------|
| `id` | CHAR(36) PK | UUID v4 |
| `conversation_id` | CHAR(36) FK | Referencia a conversations |
| `sender_id` | CHAR(36) FK | Referencia a profiles |
| `content` | TEXT | Contenido del mensaje |
| `type` | ENUM('text','call') | Tipo de mensaje |
| `created_at` | DATETIME(3) | Fecha de envío |

### Tabla: `read_receipts`
| Columna | Tipo | Descripción |
|---------|------|-------------|
| `message_id` | CHAR(36) FK | Referencia a messages |
| `user_id` | CHAR(36) FK | Referencia a profiles |
| `read_at` | DATETIME(3) | Fecha de lectura |
| PK | (message_id, user_id) | |

### Tabla: `refresh_tokens`
| Columna | Tipo | Descripción |
|---------|------|-------------|
| `id` | CHAR(36) PK | UUID v4 del token |
| `user_id` | CHAR(36) FK | Referencia a profiles |
| `token_hash` | VARCHAR(255) | SHA-256 del refresh token |
| `expires_at` | DATETIME(3) | Fecha de expiración (30 días) |
| `created_at` | DATETIME(3) | Fecha de creación |

### Tabla: `call_logs`
| Columna | Tipo | Descripción |
|---------|------|-------------|
| `id` | CHAR(36) PK | UUID v4 |
| `caller_id` | CHAR(36) FK | Referencia a profiles (quien llama) |
| `callee_id` | CHAR(36) FK | Referencia a profiles (quien recibe) |
| `type` | ENUM('audio','video') | Tipo de llamada |
| `status` | ENUM('missed','completed','rejected','cancelled','busy') | Estado |
| `started_at` | DATETIME(3) | Inicio de la llamada |
| `ended_at` | DATETIME(3) NULL | Fin de la llamada |
| `duration_secs` | INT UNSIGNED | Duración en segundos |

## Índices

```sql
CREATE INDEX idx_messages_conv_created ON messages(conversation_id, created_at);
CREATE INDEX idx_participants_user ON participants(user_id);
CREATE INDEX idx_conversations_last_msg ON conversations(last_message_at DESC);
CREATE INDEX idx_refresh_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_caller ON call_logs(caller_id);
CREATE INDEX idx_callee ON call_logs(callee_id);
CREATE FULLTEXT INDEX ft_messages_content ON messages(content);
```

## Esquema de Inicialización

El esquema se inicializa automáticamente via Docker. Ver `infra/mariadb/init.sql` para el DDL completo y `infra/docker-compose.yml` para la configuración del contenedor.

## Diferencias con Supabase/PostgreSQL

| PostgreSQL (anterior) | MariaDB (nuevo) | Razón |
|-----------------------|-----------------|-------|
| UUID nativo | CHAR(36) | MariaDB no tiene UUID nativo en modo MySQL |
| TIMESTAMPTZ | DATETIME(3) | Sin timezone, todo se almacena en UTC |
| auth.users + trigger | profiles con password_hash | Auth propio con bcrypt + JWT |
| Row Level Security | Sin RLS | Auth via middleware Go |
| Realtime subscriptions | Polling cada 2s | Más simple, sin dependencias externas |
