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

### Después de eliminar una base

Si se borra y se crea de nuevo con el mismo nombre, el volumen Docker persiste con datos viejos. Si se quiere fresh start, borrar el volumen manualmente desde el servidor.
