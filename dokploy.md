# Dokploy — Base de Datos

Documentación de cómo desplegar y gestionar bases de datos en Dokploy usando el MCP.

---

## Proyecto y entorno

Todas las bases de datos están en el proyecto **"bases de datos"**, entorno **"production"**.

| Recurso | ID |
|---------|----|
| Organización | `GiRtc8Ao6pVfB8k0UTyie` |
| Proyecto "bases de datos" | `WUrK6-W3-nUb2CCbMag_0` |
| Entorno "production" | `NOB2_Ht7g53ccOsXYaarh` |

---

## MariaDB (DaveChat)

| Campo | Valor |
|-------|-------|
| **ID** | `5jeoMT1jaISFJpzIWSUFF` |
| **Nombre** | `davechat-db` |
| **App name** | `davechat-mariadb-66rn9m` |
| **Imagen** | `mariadb:11` |
| **Host** | `192.168.101.133` |
| **Puerto** | `3307` |
| **Base de datos** | `davechat` |
| **Usuario** | `davechat` |
| **Password** | `davechat_pass` |
| **Root password** | `root_davechat_2024` |
| **Volumen** | `davechat-mariadb-66rn9m-data` → `/var/lib/mysql` |
| **Estado** | running |

### Tablas creadas

- `profiles`, `conversations`, `participants`, `messages`
- `read_receipts`, `refresh_tokens`, `call_logs`
- Índices: FULLTEXT en `messages.content`, compuestos en `messages(conversation_id, created_at)` descendente, etc.

### Permisos

- `davechat@%` → ALL PRIVILEGES ON `davechat.*`
- `root@%` → full server access

---

## Volume Setup (DaveChat — Uploads)

Para persistir los avatares subidos por los usuarios entre redeploys, la aplicación DaveChat necesita un volumen Docker nombrado montado en `/app/uploads`.

| Campo | Valor |
|-------|-------|
| **Nombre del volumen** | `davechat-m3bjt5-uploads` |
| **Mount path** | `/app/uploads` |
| **Propósito** | Persistir avatares subidos (`/uploads/*`) |
| **App relacionada** | DaveChat (`_-BbzRdJ18Cuh6EXslZvQ`) |

### Pasos en la UI de Dokploy

1. Ir a **Aplicaciones Web → production → DaveChat**
2. Abrir la pestaña **Volumes** → **Add Volume**
3. **Volume name**: `davechat-m3bjt5-uploads`
4. **Mount path**: `/app/uploads`
5. **Save** — esto registra el named volume en Dokploy (no reinicia el contenedor todavía)
6. **NO redeploy aún** — el volumen debe existir antes del primer redeploy

### Secuencia crítica: configure → verify → redeploy

```
 1. Add volume (UI) ──→ 2. Verify aparece en la lista ──→ 3. Redeploy
```

**No invertir el orden.** Si se redeploya sin el volumen montado, los avatares existentes en la capa writable del contenedor se pierden permanentemente. Dokploy guarda la configuración del volumen en su propia base de datos al hacer Save — el container restart solo ocurre en el deploy.

### Verificar persistencia

1. Subir un avatar de prueba (Settings → Upload Photo)
2. Ir a la URL del avatar (`https://davechat.tudominio.com/uploads/<filename>`) y confirmar que carga
3. Ir a Dokploy → botón **Redeploy** en la app DaveChat
4. Esperar que termine el redeploy
5. Recargar la página de DaveChat — el avatar debe seguir visible
6. (Opcional) Verificar que el volumen aparece en Dokploy: Volumes → `davechat-m3bjt5-uploads` con estado `running`

---

## Cómo crear una base de datos desde el MCP

### 1. Crear el servicio

Usar `dokploy-mcp_mysql-create` **tanto para MySQL como para MariaDB**.

| Parámetro | Ejemplo |
|-----------|---------|
| `name` | Nombre descriptivo (ej: `davechat-db`) |
| `appName` | Slug para Docker (ej: `davechat-mariadb`) |
| `databaseName` | Nombre de la base de datos interna |
| `databaseUser` | Usuario de la DB |
| `databasePassword` | Password del usuario |
| `databaseRootPassword` | Password root |
| `dockerImage` | `mariadb:11` o `mysql:8` o `postgres:18` |
| `environmentId` | ID del entorno en Dokploy |

```json
// Ejemplo MariaDB
dokploy-mcp_mysql-create({
  name: "davechat-db",
  appName: "davechat-mariadb",
  databaseName: "davechat",
  databaseUser: "davechat",
  databasePassword: "davechat_pass",
  databaseRootPassword: "root_davechat_2024",
  dockerImage: "mariadb:11",
  environmentId: "NOB2_Ht7g53ccOsXYaarh"
})
```

### 2. Configurar puerto externo

Usar `dokploy-mcp_mysql-saveExternalPort`:

| Parámetro | Valor |
|-----------|-------|
| `mysqlId` | ID del servicio |
| `externalPort` | Puerto (ej: 3307, 5432, etc.) |

**Nota:** Esta herramienta devuelve un error de schema inválido en la respuesta, pero el cambio **se aplica igual**. Verificar con `mysql-one` que el campo `externalPort` se haya actualizado.

### 3. Iniciar / desplegar

```json
dokploy-mcp_mysql-start({ mysqlId: "..." })
```

Esperar a que el estado pase de `idle` → `starting` → `running` / `done`.

### 4. Ejecutar el schema (init.sql)

No hay forma de montar `init.sql` desde el MCP, así que se ejecuta manualmente:

```bash
docker run --rm \
  -v /ruta/al/init.sql:/init.sql \
  --entrypoint bash mysql:8 \
  -c "mysql -h <HOST> -P <PORT> -u <USER> -p<PASS> <DB> < /init.sql"
```

El contenedor `mysql:8` funciona como cliente contra MariaDB también.

### 5. Probar conexión

```bash
docker run --rm --entrypoint bash mysql:8 \
  -c "mysql -h <HOST> -P <PORT> -u <USER> -p<PASS> <DB> -e 'SELECT VERSION(); SHOW TABLES;'"
```

---

## Credenciales MySQL vs MariaDB

La diferencia es solo el `dockerImage`:

| Motor | dockerImage |
|-------|-------------|
| MySQL 8 | `mysql:8` |
| MariaDB 11 | `mariadb:11` |
| PostgreSQL 18 | `postgres:18` |

El MCP usa el mismo endpoint `mysql-create` para MySQL y MariaDB (internamente Dokploy los trata igual).

---

## Troubleshooting

### Errores de schema en respuestas MCP

Algunas tools de Dokploy devuelven error:
> `MCP error -32602: Structured content does not match the tool's output schema`

**Causa:** El servidor MCP de Dokploy tiene un bug en el formato de respuesta para ciertos endpoints (`saveExternalPort`, `application-one`, `project-one`, `deploy`).

**Solución:** Ignorar el error. Verificar que el cambio se aplicó consultando el recurso con `*-one`.

### La base no acepta conexiones remotas

Verificar:
1. `SELECT user, host FROM mysql.user` — el usuario debe tener `%` como host
2. `SELECT @@bind_address` — debe ser `*` (all interfaces)
3. `SHOW VARIABLES LIKE 'skip_networking'` — debe ser OFF
4. Firewall del host: `iptables -L -n`
5. Que el externalPort esté seteado (consultar con `*-one`)

### Variables de entorno en la app

Si la app está en Dokploy con `DB_DSN` como env var y el MCP `saveEnvironment` devuelve error de schema, no se puede actualizar desde el MCP. Solución:

1. **Cambiar el nombre de la variable en el código** (ej: `DB_DSN` → `DATABASE_DSN`), así el valor viejo en Dokploy queda inerte
2. El default hardcodeado en `config.go` se usa cuando la env var no está seteada
3. Para desarrollo local, actualizar `server/.env` y `infra/docker-compose.yml` con el nuevo nombre

---
