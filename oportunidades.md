# Oportunidades de Optimización y Código Obsoleto - DaveChat

Este documento detalla los hallazgos encontrados durante el análisis del código, enfocándose en el rendimiento, la calidad del código y la eliminación de redundancias.

## 🚀 Servidor (Go)

### 🔴 Alto Impacto
- **Cuello de botella en actualizaciones de presencia**: En `server/internal/websocket/handler.go`, la función `hub.TouchPresence(client.UserID)` se llama en cada mensaje de WebSocket recibido y en cada `Pong`. Esto genera una cantidad masiva de consultas `UPDATE` en la base de datos.
  - **Propuesta**: Implementar un sistema de "dirty flag" o un temporizador para actualizar la presencia en la DB cada 30-60 segundos en lugar de cada mensaje.

### 🟡 Impacto Medio
- **Complejidad en limpieza de salas**: En `server/internal/websocket/hub.go`, el método `Unregister` itera sobre todas las salas (`h.rooms`) para eliminar al usuario. Si hay muchas salas, esto se vuelve ineficiente.
  - **Propuesta**: Almacenar en el struct `Client` una lista de las salas en las que el usuario está registrado para realizar la eliminación directa.

### 🟢 Bajo Impacto
- **Implementación manual de UUID**: En `server/internal/websocket/hub.go`, se utiliza una función `newUUID()` implementada manualmente.
  - **Propuesta**: Utilizar una librería estándar y probada como `github.com/google/uuid`.
- **Sugerencia de Índices DB**: Verificar que la tabla `call_logs` tenga índices optimizados para `caller_id`, `callee_id` y `started_at` debido a las consultas de actualización y búsqueda por fecha.

---

## 🎨 Cliente (React/TS)

### 🔴 Alto Impacto
- **Re-renders innecesarios en `CallContext`**: En `client/src/contexts/CallContext.jsx`, el objeto `value` del `CallProvider` se recrea en cada renderizado. Esto provoca que todos los componentes que usan `useCall()` se vuelvan a renderizar cada vez que cambie cualquier estado (como el estado de la llamada o la lista de dispositivos).
  - **Propuesta**: Envolver el objeto `value` en un `useMemo`.

### 🟡 Impacto Medio
- **Ineficiencia en el Ringtone**: En `client/src/contexts/CallContext.jsx`, `startRingtone` crea un nuevo `AudioContext` y un `Oscillator` cada 4 segundos.
  - **Propuesta**: Crear el `AudioContext` una sola vez y reutilizarlo, o utilizar un archivo de audio pregrabado en loop.

### 🟢 Bajo Impacto
- **Redundancia en constraints de audio**: En `client/src/contexts/CallContext.jsx`, dentro de `createPeerConnection`, las `audioConstraints` se definen dos veces (una en el flujo principal y otra en el catch de error de video).
  - **Propuesta**: Definir las constraints una sola vez al inicio de la función.
- **Backoff lineal en reconexiones WS**: En `client/src/lib/websocket.js`, el reintento de conexión utiliza un backoff lineal simple.
  - **Propuesta**: Implementar un backoff exponencial para reducir la carga en el servidor durante fallos masivos.
- **Logs de depuración**: Se detectaron aproximadamente 26 llamadas a `console.log` en el código fuente del cliente.
  - **Propuesta**: Eliminar los logs de depuración o implementar un sistema de logging configurable por entorno.
