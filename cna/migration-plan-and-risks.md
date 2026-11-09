# A5 – Plan de Migración y Matriz de Riesgos

## 1. Orden de Extracción Justificado y Criterio de "Hecho" (Definition of Done)

El proceso de migración de la arquitectura monolítica hacia microservicios se realiza aplicando el patrón *Strangler Fig*, ordenando la extracción de **menor a mayor riesgo**:

### Paso 1: Servicio de Exportación Asíncrona (`Export Service` / `/v2/export`)
* **Justificación de menor riesgo:** Es un proceso asíncrono y de baja criticidad inmediata. Si falla la generación de un reporte o PDF, no afecta las transacciones principales ni la persistencia de datos en la plataforma.
* **Criterio de "Hecho" (Definition of Done):**
  * Redirección exitosa del 100% del tráfico `/v2/export` en el API Gateway hacia el worker en Node.js.
  * Respuestas síncronas del Gateway con código `202 Accepted`.
  * Publicación y consumo de eventos `export.requested` y `export.completed` a través de RabbitMQ.
  * Archivos PDF guardados correctamente en Object Storage (MinIO / S3).
  * Tasa de error < 0.1% en un periodo de prueba continuo de 48 horas.

### Paso 2: Servicio de Búsqueda Síncrona (`Search Service` / `/v2/search`)
* **Justificación de riesgo medio:** Es un flujo síncrono de lectura intensa. Un fallo degrada la experiencia de búsqueda pero no corrompe la base de datos ni afecta las escrituras.
* **Criterio de "Hecho" (Definition of Done):**
  * API Gateway redirige peticiones de lectura `/v2/search` al microservicio Express.js.
  * Latencia de respuesta por debajo de los 150 ms en el percentil 95 (p95).
  * Consultas a la base de datos `db_core` ejecutadas de forma eficiente mediante índices optimizados.
  * Manejo adecuado de errores con códigos HTTP `400 Bad Request` y `503 Service Unavailable`.

### Paso 3: Desacoplamiento Event-Driven (`BookStack Core Event Bus`)
* **Justificación de mayor riesgo:** Requiere modificar la capa de persistencia y controladores del monolito PHP/Laravel para emitir eventos de dominio tras escrituras exitosas.
* **Criterio de "Hecho" (Definition of Done):**
  * Eventos `bookstack.page.updated` emitidos a RabbitMQ sin bloquear el hilo principal de atención HTTP en el monolito.
  * Procesamiento asíncrono por los consumidores asociados.
  * Redirección automática de mensajes no procesables a la Dead Letter Queue (DLQ).

---

## 2. Estrategia de Reversión (Rollback en < 1 minuto)

Para desactivar cualquier ruta `/v2/*` en caso de falla crítica y redirigir el tráfico inmediatamente de vuelta al monolito *legacy*, se aplica un mecanismo de **conmutación rápida en el API Gateway (Nginx)**:

### Procedimiento Técnico de Rollback:
1. **Conexión a la instancia de Gateway:** Se ingresa al servidor o contenedor Nginx.
2. **Reconfiguración del Upstream:** En el archivo de configuración de rutas (`nginx.conf`), se modifica el bloque de la ruta afectada (ej. `/v2/search`) redirigiéndolo al contenedor del monolito:
   ```nginx
   location /v2/search {
       # Proxy Pass temporal de reversión al monolito legacy
       proxy_pass http://app_monolith:8080/api/legacy/search;
       proxy_set_header Host $host;
       proxy_set_header X-Real-IP $remote_addr;
   }
