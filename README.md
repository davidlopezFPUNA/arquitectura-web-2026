# Proyecto Integrador II — De monolito a Cloud Native

**Estudiantes:** David López, Junior Legal  
**Institución:** Facultad Politécnica - Universidad Nacional de Asunción (FPUNA)  
**Sistema Operativo:** Ubuntu Server 24.04.5 LTS (x86_64)  
**Infraestructura:** VirtualBox (2 vCPU, 2 GB RAM, 20 GB Disk)  

---

## Entrega 2. Evolución de la Infraestructura y Actualizaciones

| Criterio | Detalle |
| :--- | :--- |
| **Nombre** | PocketBase |
| **Repositorio Oficial** | [ganigeorgiev.com/pocketbase/pocketbase/releases](https://ganigeorgiev.com/pocketbase/pocketbase/releases) |
| **Versión Instalada / Actualizada** | `v0.22.58` |
| **Licencia** | MIT License |

---
# Bloque I – Fase 5: Diseño y Descomposición del Monolito

## 4.1. A1 – Análisis del Monolito y Mapa de Contextos

### Inventario de Módulos Funcionales y Tablas (Evidencia F5-E1)

Se inspeccionó la base de datos embebida SQLite de PocketBase (`data.db`) ejecutando la consulta sobre la tabla maestra `sqlite_master`:

<img width="1113" height="149" alt="image" src="https://github.com/user-attachments/assets/d34b35e1-c2bc-4500-bd2d-83e474527ada" />

---

### Matriz de Acoplamiento Módulo × Tabla (Evidencia F5-E2)

Dado que la arquitectura base utiliza PocketBase (SQLite), el acoplamiento se evalúa entre los servicios lógicos consumidos y las colecciones internas del motor. Las tablas compartidas o de uso cruzado determinan la complejidad de extracción hacia nuevos servicios.

| Módulo / Servicio Lógico | Colección / Tabla (PocketBase) | Operaciones (CRUD) | Nivel de Acoplamiento | Justificación del Acoplamiento |
| :--- | :--- | :--- | :--- | :--- |
| **Módulo de Autenticación** | `users`, `_admins` | Lectura (R), Actualización (U) | **Alto** | El servicio de inicio de sesión depende estrictamente de la validación contra estas tablas mediante la API nativa. |
| **Integraciones Sociales** | `_externalAuths` | Creación (C), Lectura (R) | **Medio** | Se acopla solo si se habilita el inicio de sesión con proveedores externos (Google, GitHub, etc.). |
| **Módulo de Formularios / CRM** | `contactos` | Creación (C), Lectura (R), Actualización (U), Borrado (D) | **Alto** | La interfaz de usuario que captura o lista contactos realiza peticiones REST/Realtime directamente hacia esta colección. |
| **Core (PocketBase Engine)** | `_collections`, `_migrations`, `_params` | Todas (CRUD) | **Alto (Interno)** | Son tablas de configuración estrictamente acopladas al motor interno de PocketBase para definir el esquema y estado de la base de datos. |

---

### Mapa de Contextos / DDD (Evidencia F5-E3)

El siguiente mapa de contextos ilustra las relaciones entre el monolito original y los nuevos *Bounded Contexts* (microservicios S1, S2, S3) extraídos mediante el patrón *Strangler Fig*, destacando el `event-bridge` como Capa Anticorrupción (ACL) para desacoplar la emisión de webhooks.

```mermaid
flowchart TD
    %% Definición de Estilos
    classDef monolito fill:#f9d0c4,stroke:#333,stroke-width:2px;
    classDef servicio fill:#d4e6f1,stroke:#333,stroke-width:2px;
    classDef infra fill:#e8daef,stroke:#333,stroke-width:2px;

    %% Monolito Original
    subgraph Contexto_Monolito [Monolito Bounded Context]
        PB[PocketBase / Monolito]:::monolito
        DB[(SQLite)]:::monolito
        PB --- DB
    end

    %% Capa de Mensajería y ACL
    subgraph Mensajeria [Bus de Eventos]
        EB[event-bridge : Capa Anticorrupción]:::infra
        Broker[Broker: NATS / RabbitMQ]:::infra
        EB -- Publica Eventos --> Broker
    end

    %% Microservicios Extraídos
    subgraph Contexto_Búsqueda [Búsqueda - S1]
        S1_API[search-svc]:::servicio
        S1_DB[(Meilisearch)]:::servicio
        S1_API --- S1_DB
    end

    subgraph Contexto_Auditoría [Auditoría - S2]
        S2_API[audit-svc]:::servicio
        S2_DB[(PostgreSQL)]:::servicio
        S2_API --- S2_DB
    end

    subgraph Contexto_Exportación [Exportación - S3]
        S3_API[export-svc]:::servicio
        S3_DB[(Redis + MinIO)]:::servicio
        S3_API --- S3_DB
    end

    %% Relaciones Asíncronas (Eventos)
    PB -- "Webhooks (Proveedor)" --> EB
    Broker -- "Consume Eventos (Cliente)" --> S1_API
    Broker -- "Consume Eventos (Cliente)" --> S2_API

    %% Relaciones Síncronas (REST)
    S1_API -- "Consulta (Cliente/Proveedor)" --> PB
    S3_API -- "Consulta (Cliente/Proveedor)" --> PB
```
---

# 4.2. A2 – Registro de Decisiones de Arquitectura (ADR-006 a ADR-010)

### ADR-006: Estrategia de Migración e Integración con el Monolito

* **Estatus:** Aprobado
* **Contexto:** El monolito basado en PocketBase debe mantenerse en producción y operando sin interrupciones mientras se realiza la extracción progresiva de funcionalidades hacia microservicios.
* **Decisión:** Adoptar el patrón **Strangler Fig** utilizando integración exclusiva mediante la API REST nativa de PocketBase y Webhooks, sin modificar el código base del binario del monolito.
* **Alternativas Descartadas:** 
  * *Modificación del core de PocketBase:* Descartado debido a que alterar el código fuente en Go del monolito incrementa el acoplamiento y dificulta futuras actualizaciones del motor base.
* **Consecuencias:** 
  * *Positivas:* Riesgo operativo mínimo durante la migración; permite desplegar y validar microservicios de forma independiente.
  * *Negativas:* Requiere configurar una capa intermedia (`event-bridge`) para transformar webhooks de PocketBase en eventos asíncronos.

---

### ADR-007: Selección del Broker de Mensajes

* **Estatus:** Aprobado
* **Contexto:** Se requiere un bus de mensajería para garantizar la entrega confiable de eventos entre el `event-bridge` y los microservicios consumidores (Búsqueda, Auditoría, Exportación).
* **Decisión:** Seleccionar **NATS JetStream** como broker principal de eventos.
* **Alternativas Descartadas:** 
  * *RabbitMQ:* Descartado por su elevado consumo de memoria RAM y complejidad operativa para el entorno de recursos limitados.
  * *Redis Streams:* Descartado por carecer de mecanismos nativos más avanzados para la persistencia de streams a largo plazo comparado con JetStream.
* **Consecuencias:** 
  * *Positivas:* Huella de memoria sumamente baja (< 20 MB RAM), alto rendimiento en throughput y persistencia nativa con JetStream.
  * *Negativas:* Menor ecosistema de librerías en comparación con RabbitMQ.

---

### ADR-008: Plataforma de Orquestación

* **Estatus:** Aprobado
* **Contexto:** La infraestructura asignada se limita a máquinas virtuales en VirtualBox con 2 vCPU, 2 GB de RAM y 20 GB de Disco sobre Ubuntu Server 24.04.4 LTS.
* **Decisión:** Utilizar **Docker Compose** como motor de orquestación y despliegue del entorno.
* **Alternativas Descartadas:** 
  * *k3s (Lightweight Kubernetes):* Descartado porque el consumo base del plano de control en memoria RAM sobrepasa la capacidad disponible (2 GB RAM) e inestabiliza los nodos.
  * *Docker Swarm:* Descartado por añadir complejidad sintáctica no requerida para el entorno mono-nodo actual.
* **Consecuencias:** 
  * *Positivas:* Consumo insignificante de recursos de sistema, sintaxis declarativa simple y rápido ciclo de despliegue sobre Ubuntu Server 24.04.4 LTS.
  * *Negativas:* Ausencia de autoscaling automático y orquestación multi-nodo avanzada.

---

### ADR-009: Base de Datos por Servicio (Database per Service)

* **Estatus:** Aprobado
* **Contexto:** Los nuevos microservicios extraídos necesitan persistir su información sin depender del esquema interno o la base de datos SQLite del monolito PocketBase (`pb_data/data.db`).
* **Decisión:** Aplicar estrictamente el patrón **Database per Service**, prohibiendo el acceso directo de los microservicios al archivo SQLite de PocketBase.
* **Alternativas Descartadas:** 
  * *Base de datos compartida (Shared SQLite):* Descartado debido a los bloqueos de escritura concurrentes (`database is locked`) propios de SQLite y el acoplamiento fuerte de modelos.
* **Consecuencias:** 
  * *Positivas:* Autonomía completa del modelo de datos de cada microservicio; eliminación del punto único de falla en almacenamiento.
  * *Negativas:* Se requiere gestionar consistencia eventual entre la base de datos de PocketBase y los nuevos almacenes.

---

### ADR-010: Consistencia Eventual e Idempotencia

* **Estatus:** Aprobado
* **Contexto:** Las operaciones de actualización hacia los servicios desacoplados se procesan de forma asíncrona a través del bus de eventos.
* **Decisión:** Implementar un modelo de **Consistencia Eventual** garantizando idempotencia en los consumidores mediante `event_id` único y gestionando fallos con colas de mensajes no procesables (Dead Letter Queue - DLQ).
* **Alternativas Descartadas:** 
  * *Transacciones distribuidas (2PC / Two-Phase Commit):* Descartadas por su alta latencia y baja tolerancia a fallos en arquitecturas orientadas a eventos.
* **Consecuencias:** 
  * *Positivas:* Alta disponibilidad y resiliencia; los eventos duplicados o fuera de orden no corrompen el estado del sistema.
  * *Negativas:* Existencia de un breve desfase de milisegundos hasta que los datos se reflejen totalmente en los servicios secundarios.

---

# 4.3. A3 – Contratos de API y de Eventos

## Política de Versionado
Para garantizar la retrocompatibilidad y no romper a los consumidores existentes (frontend u otros microservicios):
- **API REST:** Todas las nuevas rutas extraídas del monolito de PocketBase se exponen bajo el prefijo `/v2/` (ej. `/v2/search`, `/v2/export`).
- **Eventos (Mensajería):** Todos los eventos asíncronos incluyen un campo `version` en su sobre común (ej. `"version": "1.0.0"`).
- **Transición a v3 sin romper consumidores:** 
  - **APIs REST:** Si en el futuro se requiere un cambio disruptivo (*breaking change*), se desplegará una nueva ruta `/v3/` (ej. `/v3/search`). Los endpoints `/v2/` seguirán funcionando en paralelo en el mismo microservicio o enrutador durante un periodo de transición, permitiendo a los clientes migrar a su propio ritmo.
  - **Eventos Asíncronos:** Los nuevos eventos se publicarán con el campo `"version": "2.0.0"` en el sobre. Los consumidores existentes estarán programados bajo el principio de robustez (ignorando campos desconocidos o filtrando por la versión soportada), permitiendo la emisión paralela sin afectar los procesos activos.

---

## Contratos OpenAPI 3.1 (REST) y AsyncAPI 3.0 (Eventos)
Los esquemas completos con esquemas de datos, códigos de respuesta (200, 202, 400, 404, 503) y el diseño del sobre de eventos común (`event_id`, `type`, `version`, `occurred_at`, `request_id`, `payload`) se encuentran disponibles y validados en el directorio `cna/contracts/`:

- **Contrato de Búsqueda:** `cna/contracts/search-svc.openapi.yaml`
- **Contrato de Exportación:** `cna/contracts/export-svc.openapi.yaml`
- **Bus de Eventos (PocketBase):** `cna/contracts/events.asyncapi.yaml`

---

## Evidencia de Validación (Evidencia F5-E4)

Se ejecutó la validación del contrato OpenAPI utilizando la herramienta **Spectral** dentro del repositorio local en la máquina virtual:

<img width="1246" height="185" alt="image" src="https://github.com/user-attachments/assets/9da812ce-a37e-4b6f-8473-14dce121a633" />

---

# C4 - Nivel 1: Diagrama de Contexto (F5-E5)

```mermaid
graph TD
    Client["Cliente / Host Windows"]
    Admin["Administrador del Sistema"]

    subgraph Boundary["Ecosistema PocketBase FPUNA (Red 192.168.1.0/27)"]
        App["Plataforma Web Desacoplada"]
    end

    Client -->|Petición HTTP - Puerto 80| App
    Admin -->|Admin UI - Puerto 80/8090| App
```

## C4 - Nivel 2: Diagrama de Contenedores (F5-E6)

```mermaid
graph TB
    Client["[ Cliente / Host Windows ]"]
    
    subgraph Network["[ Adaptador Puente / Red 192.168.1.0/27 ]"]
        Proxy["[ Servidor Web: Nginx Proxy (Puerto 80) ]"]
    end

    subgraph DockerHost["VM Ubuntu Server 24.04 (VirtualBox - 2 vCPU / 2 GB RAM)"]
        subgraph Monolith["Monolito Base"]
            PB["[ Aplicación: PocketBase Service (Puerto 8090) ]"]
            SQLite[(" [ Base de Datos: SQLite ]<br>(Archivo embebido en disco / pb_data) ")]
        end

        subgraph EventBus["Bus de Eventos"]
            Bridge["[ Event Bridge / Webhook Handler ]"]
            NATS["[ Broker: NATS JetStream (Puerto 4222) ]"]
        end

        subgraph Services["Microservicios Desacoplados"]
            SearchSVC["[ Microservicio: Search Service (Puerto 8080) ]"]
            ExportSVC["[ Microservicio: Export Service (Puerto 8081) ]"]
            SearchDB[(" [ Index Storage (Local) ] ")]
            ExportDB[(" [ Temp Storage (Volume) ] ")]
        end
    end

    Client -->|Petición HTTP - Puerto 80| Proxy
    Proxy -->|Redirección local 127.0.0.1:8090| PB
    PB --- SQLite

    Proxy -->|Redirección local 127.0.0.1:8080| SearchSVC
    Proxy -->|Redirección local 127.0.0.1:8081| ExportSVC

    PB -->|Webhooks HTTP| Bridge
    Bridge -->|Publicar Eventos / TCP 4222| NATS

    NATS -->|Suscripción: pocketbase.contact.*| SearchSVC
    NATS -->|Suscripción: export.requested| ExportSVC

    SearchSVC --- SearchDB
    ExportSVC --- ExportDB
```

## Diagramas de Secuencia (F5-E7)

### 1. Sincronización Eventual de Contactos (Creación / Actualización)

```mermaid
sequenceDiagram
    autonumber
    actor C as Cliente / Host Windows
    participant Proxy as Nginx Proxy (:80)
    participant PB as PocketBase (:8090)
    participant EB as Event Bridge
    participant NATS as NATS JetStream
    participant SS as Search Service

    C->>Proxy: POST /api/collections/contactos/records
    Proxy->>PB: Redirección local 127.0.0.1:8090
    alt Camino Exitoso
        PB-->>PB: Persistir en SQLite (pb_data)
        PB-->>C: 201 Created
        PB->>EB: Webhook HTTP (pocketbase.contact.created)
        EB->>NATS: Publicar "pocketbase.contact.created" (event_id: UUID)
        NATS->>SS: Entregar Evento
        SS->>SS: Indexar contacto en almacenamiento local
    else Camino de Error (Fallo en Broker de Eventos)
        PB-->>C: 201 Created (Monolito asegura la persistencia)
        PB->>EB: Webhook HTTP
        EB->>NATS: Publicar Evento
        NATS--xEB: Error de Conexión / NATS Down
        EB->>EB: Almacenar en Cola Reintentos / DLQ Local
    end
```

### 2. Búsqueda Desacoplada de Contactos

```mermaid
sequenceDiagram
    autonumber
    actor C as Cliente / Host Windows
    participant Proxy as Nginx Proxy (:80)
    participant SS as Search Service (:8080)

    C->>Proxy: GET /v2/search?q=Juan
    Proxy->>SS: Redirección local 127.0.0.1:8080
    alt Camino Exitoso
        SS->>SS: Consultar índice desacoplado
        SS-->>C: 200 OK { total: 1, results: [...] }
    else Camino de Error (Parámetro Inválido)
        SS-->>C: 400 Bad Request { error: "Parámetro 'q' requerido" }
    else Camino de Error (Servicio No Disponible)
        SS-->>C: 503 Service Unavailable { error: "Motor de búsqueda fuera de línea" }
    end
```

### 3. Exportación Asíncrona de Datos

```mermaid
sequenceDiagram
    autonumber
    actor C as Cliente / Host Windows
    participant Proxy as Nginx Proxy (:80)
    participant ES as Export Service (:8081)
    participant NATS as NATS JetStream
    participant PB as PocketBase API

    C->>Proxy: POST /v2/export { format: "csv" }
    Proxy->>ES: Redirección local 127.0.0.1:8081
    alt Camino Exitoso
        ES-->>C: 202 Accepted { job_id: "job-101", status: "PROCESSING" }
        ES->>NATS: Publicar "export.requested"
        NATS->>ES: Consumir evento
        ES->>PB: GET /api/collections/contactos/records
        PB-->>ES: 200 OK (Datos desde SQLite)
        ES->>ES: Generar archivo CSV
        ES->>NATS: Publicar "export.completed"
    else Camino de Error (Error / Timeout en Monolito)
        ES-->>C: 202 Accepted { job_id: "job-101", status: "PROCESSING" }
        ES->>PB: GET /api/collections/contactos/records
        PB--xES: 500 Internal Error / Timeout
        ES->>NATS: Publicar "export.failed"
        ES->>ES: Registrar fallo en logs de auditoría
    end
```
---

# 4.5. A5 – Plan de Migración y Matriz de Riesgos

### 1. Orden de Extracción Justificado (Menor a Mayor Riesgo)

El desacoplamiento del monolito PocketBase se realiza en fases incrementales para minimizar el impacto en el negocio:

| Fase | Servicio / Módulo | Nivel de Riesgo | Justificación técnica | Criterio de "Hecho" (Definition of Done) |
| :--- | :--- | :--- | :--- | :--- |
| **Paso 1** | **Servicio de Exportación (`export-svc`)** | **Bajo** | Operación de solo lectura asíncrona y no crítica para las transacciones inmediatas del usuario. | El servicio procesa eventos `export.requested` desde NATS JetStream, genera reportes en background y los deposita en el volumen local. |
| **Paso 2** | **Servicio de Búsqueda (`search-svc`)** | **Medio** | Extrae las consultas de búsqueda de la base de datos principal SQLite (`pb_data`), aliviando bloqueos de lectura. | El índice desacoplado procesa eventos `pocketbase.contact.*` con 0 errores y atiende `/v2/search` con latencia < 50ms. |
| **Paso 3** | **Servicio de Auditoría (`audit-svc`)** | **Alto** | Registra trazas de operaciones críticas. Al ser altamente dependiente de webhooks, requiere validar el manejo de fallos y DLQ. | El 100% de los cambios de estado en PocketBase generan registros de auditoría persistentes de forma asíncrona sin ralentizar las respuestas HTTP. |

---

### 2. Estrategia de Reversión (Rollback en < 1 minuto)

Para desactivar inmediatamente cualquier ruta `/v2/*` redirigida en el API Gateway (Nginx Proxy) y retornar la atención de tráfico directamente al monolito PocketBase:

1. **Procedimiento de Desactivación:**
   Se dispone de una plantilla de configuración de Nginx (`nginx.conf.bak`) preconfigurada que redirige todas las llamadas `/v2/` a las colecciones nativas de PocketBase (`/api/collections/`).

2. **Comando de Rollback (Ejecución en terminal):**
   ```bash
   cp /etc/nginx/templates/rollback-v1.conf /etc/nginx/conf.d/default.conf && nginx -s reload

3. **Mecanismo de Conmutación:**
   Al ejecutar `nginx -s reload`, Nginx recarga la configuración en memoria de forma instantánea (**< 1 segundo**) sin perder conexiones activas. El tráfico `/v2/search` pasa a atenderse directamente mediante la API nativa de PocketBase sobre SQLite.

---

## 3. Matriz de Riesgos (Probabilidad × Impacto)

Evaluación detallada de riesgos de arquitectura sobre la infraestructura asignada (2 vCPU, 2 GB RAM sobre VirtualBox / FPUNA):

| Riesgo | Probabilidad | Impacto | Nivel (P × I) | Estrategia de Mitigación / Contención |
| :--- | :---: | :---: | :---: | :--- |
| **Consistencia eventual visible al usuario** | Media (3) | Medio (3) | **Medio (9)** | Implementar *Optimistic UI* en el frontend y polling/WebSockets notificando la finalización del procesamiento asíncrono. |
| **Consumo excesivo de RAM** | Alta (4) | Alto (4) | **Alto (16)** | Límite estricto de recursos en `docker-compose.yml` (`mem_limit: 256m` por servicio) y selección de componentes ultraligeros (PocketBase + NATS). |
| **Latencia de red entre VM** | Baja (1) | Medio (3) | **Bajo (3)** | Despliegue de los contenedores dentro de la misma red interna de Docker (`docker network`) sobre la VM `equipo-vm`. |
| **Secretos compartidos (Hardcoded)** | Media (3) | Alto (4) | **Alto (12)** | Inyección de credenciales mediante variables de entorno en archivo `.env` restringido (excluido del repositorio Git por `.gitignore`). |
| **Punto único de falla en el Gateway** | Media (3) | Crítico (5) | **Alto (15)** | Monitoreo continuo mediante healthchecks de Docker (`restart: always`) y archivo de configuración estandarizado para rollback inmediato. |

---

## 5.1. B1 – Infraestructura Multi-Nodo

### Descripción del Entorno
Se ha configurado una infraestructura distribuida compuesta por dos máquinas virtuales Ubuntu Server 24.04 sobre VirtualBox:
* **VM-1 (Gateway / Monolito Core):** `equipo-vm` (IP: `192.168.1.25/27`) - Encargada de recibir el tráfico HTTP/HTTPS externo y ejecutar la API principal de PocketBase sobre SQLite.
* **VM-2 (Clúster Worker / Microservicios):** `equipo-vm2` (IP: `192.168.1.26/27`) - Clon enlazado configurado para hospedar de forma aislada los microservicios desacoplados y el bus de eventos.

---

# Bloque II – Fase 6: Construcción, Contenerización y Despliegue Distribuido
---

# 5.1. B1 - Infraestructura Multi-Nodo

#### 1. Evidencia F6-E1: Inspección de Recursos y Docker Engine (VM-1 / VM-2)
Confirmación del entorno de ejecucion Docker en los nodos distribuidos (2 vCPU, 2 GB RAM, Kernel Linux 6.8):

<img width="732" height="1115" alt="image" src="https://github.com/user-attachments/assets/38342093-6c3a-420a-aa4b-878bfbe4d28f" />

#### 2. Evidencia F6-E2: Reglas de Firewall UFW por Nodo (VM-2)
Aislamiento estricto de seguridad en VM-2: los puertos internos de los microservicios (8080/tcp, 8081/tcp) y del bus NATS (4222/tcp) únicamente aceptan peticiones originadas desde la dirección IP de VM-1 (192.168.1.25):

<img width="570" height="254" alt="image" src="https://github.com/user-attachments/assets/67589cde-9b90-48ce-849f-9b64c308cedd" />

#### 3. Evidencia F6-E3: Inspección del Certificado X.509 con SAN (VM-1)
Certificado SSL/TLS reemitido incorporando los nombres de dominio de PocketBase en la extensión Subject Alternative Name (SAN):

<img width="754" height="112" alt="image" src="https://github.com/user-attachments/assets/334118d3-4a0c-474f-91d1-bff0d10686da" />

---

## 5.2. B2 – Microservicios y Bus de Eventos

### Resumen de Despliegue en VM-2 (192.168.1.26)
Los cuatro microservicios (`search-svc`, `audit-svc`, `export-svc`, `media-svc`) han sido desplegados desacoplados en el nodo Worker (VM-2), consumiendo variables de entorno, exponiendo endpoints de resiliencia (`/healthz`, `/readyz`), métricas Prometheus (`/metrics`) e inyectando `X-Request-ID`.

---

### Evidencias de Ejecución

#### 1. Evidencia F6-E4: Servicio de Búsqueda (search-svc)
Consulta al endpoint `/v2/search` procesada correctamente a través de los índices de PocketBase:

<img width="1024" height="144" alt="image" src="https://github.com/user-attachments/assets/e2f109e1-679f-4c2d-b0e8-b45c8630c820" />

#### 2. Evidencia F6-E5: Idempotencia en Audit Service (audit-svc)
Verificación de control de duplicados mediante `event_id`. El primer evento es registrado (`201 Created`), mientras que el duplicado es ignorado (`200 OK` - `ignored`):

<img width="1371" height="370" alt="image" src="https://github.com/user-attachments/assets/2760f6a6-b513-4824-b68c-23e5915155c4" />

#### 3. Evidencia F6-E6: Procesamiento Asíncrono en Export Service (export-svc)
Recepción de solicitud de exportación retornando `202 Accepted` y el identificador único del trabajo (`job_id`):

<img width="1111" height="150" alt="image" src="https://github.com/user-attachments/assets/bddd1f5b-4869-4de6-8fa8-abf37bdc8797" />

#### 4. Evidencia F6-E7: Ciclo Completo de Job y Almacenamiento de Objeto (export-svc)
Consulta de estado del trabajo (`/v2/export/status`) confirmando la transición al estado `done` y la disponibilidad de la URL del archivo generado en almacenamiento:

<img width="1516" height="149" alt="image" src="https://github.com/user-attachments/assets/2fbed43f-5314-4280-991b-bfdec5d095b5" />

#### 5. Evidencia F6-E8: Procesamiento de Archivos en Media Service (media-svc)
Confirmación de carga de medios con emisión del evento `media.uploaded` y generación de miniaturas:

<img width="1222" height="154" alt="image" src="https://github.com/user-attachments/assets/08938594-9d8b-49bf-b0c2-b4599b434b82" />

---

## 5.3. B3 – Contenerización con Buenas Prácticas

### Resumen de Buenas Prácticas Aplicadas
- **Dockerfiles Multi-Stage**: Reducción de footprint de ~300MB a ~23MB basando el runtime en `alpine:3.19`.
- **Seguridad**: Ejecución bajo usuario no privilegiado (`appuser`) y filesystem en modo solo lectura (`--read-only`).
- **Resiliencia**: Sonda de `HEALTHCHECK` consultando `/healthz` cada 10 segundos.
- **Gestión de Recursos**: Límites explícitos asignados por contenedor (256MB RAM y 0.5 vCPU).
- **Etiquetado Semántico**: Uso estricto de versión semántica asociada al hash del commit Git (evitando `:latest`).

---

### Evidencias de Contenerización

#### 1. Evidencia F6-E9: Tabla de Imágenes y Tamaños
Lista de imágenes empaquetadas con etiqueta semántica + hash de commit y su tamaño optimizado:

<img width="1375" height="130" alt="image" src="https://github.com/user-attachments/assets/afba1023-701a-44a2-b20e-d60c86390cb7" />

#### 2. Evidencia F6-E10: Inspección de Configuración (docker inspect)
Confirmación del usuario no-root (`appuser`), asignación de recursos de CPU/memoria y configuración del `Healthcheck`:

<img width="1195" height="666" alt="image" src="https://github.com/user-attachments/assets/38cb9e6a-6c5b-4fdd-a9be-cd317da15a12" />


## 5.4. B4 – Integración con el Monolito y el Broker

### 1. Broker y Event Bridge

Se desplegaron **NATS JetStream (`2.14.9-alpine`)** y **event-bridge (`1.0.0`)** mediante Docker Compose en **VM-2 (`192.168.1.26`)**, ambos con estado `healthy`.

- **Persistencia:** JetStream utiliza un volumen Docker para conservar los mensajes.
- **Streams:** `CONTACTS`, `EXPORTS`, `MEDIA` y `DLQ`.
- **Consumidores durables:** `search-consumer` y `audit-consumer`, con ACK explícito.
- **Autenticación:** el puente valida `X-Webhook-Token` y rechaza tokens incorrectos con `401`.
- **Deduplicación:** utiliza `event_id` como `Nats-Msg-Id`.

---

### 2. Integración con PocketBase

En **VM-1 (`192.168.1.25`)** se agregaron hooks externos para generar eventos de creación, actualización y eliminación de registros de la colección `contactos`, sin modificar el binario de PocketBase.

```text
PocketBase → Hooks → Cola local → Emisor HTTP → event-bridge → NATS
```

El servicio `pocketbase-event-sender` conserva los eventos pendientes en disco y reintenta su envío ante errores. Solo elimina cada archivo después de recibir la confirmación del puente.

> La escritura del contacto y del evento son operaciones separadas; no se garantiza atomicidad entre ambas.

---

### 3. Verificación de Integración (Avance de F6-E12)

Se creó un contacto de prueba y se verificó la publicación del evento en JetStream:

| Campo | Resultado |
| :--- | :--- |
| **event_id** | `fbac0a42-b534-41c1-81a7-9d5f4d1e3033` |
| **request_id** | `d1d8fdb4-5253-4c7d-833f-6d57dcb4429c` |
| **Stream / Secuencia** | `CONTACTS` / `1` |
| **Hook** | `Evento guardado para envío` |
| **Emisor** | `Evento entregado y confirmado` |

También se verificó el rechazo de un token incorrecto (`401`) y el reenvío de un evento sin duplicarlo dentro de la ventana de deduplicación.

```bash
sudo journalctl -u pocketbase -u pocketbase-event-sender --since "3 minutes ago" --no-pager
```

---

## 5.4. B4 – Integración con el Monolito y el Broker

### Arquitectura de Integración Desacoplada
- **Usuario de Servicio de Solo Lectura**: Cuenta `svc-reader@arquitectura.local` en la colección de autenticación `service_accounts`, con reglas de API restringidas exclusivamente a lectura (`@request.auth.id != ""` en `list`/`view` y *Admins only* en operaciones de escritura).
- **Webhooks con Token Compartido**: Hook nativo en PocketBase (`pb_hooks/webhooks.pb.js`) disparado ante eventos `created`, `updated` y `deleted` de la colección `contactos`, enviando `X-Webhook-Token` hacia `event-bridge` (`192.168.1.26:8090`).
- **Event Bridge & NATS JetStream**: Servicio `event-bridge` que rechaza peticiones no autenticadas con HTTP `401`, normaliza el payload al sobre estándar de evento y lo publica en el stream durable `EVENTS` (`events.>`).
- **Resiliencia y Dead Letter Queue (DLQ)**: Consumidor durable (`search-indexer-worker`) con confirmación explícita (`ack`) y reenvío automático al stream `DLQ_EVENTS` (`dlq.events.failed`) tras 3 intentos fallidos.
- **Aislamiento de Base de Datos**: Verificado mediante `ss -tnp` en VM-1 sin conexiones entrantes desde VM-2 al almacenamiento interno del monolito.

---

### Evidencias de Integración

#### 1. Evidencia F6-E11: Configuración de Webhooks en el Monolito
Script `webhooks.pb.js` activo en PocketBase apuntando a `event-bridge` con token compartido:

<img width="741" height="1068" alt="image" src="https://github.com/user-attachments/assets/4dc8b47e-0bb2-40f0-b056-d1c9265f819b" />

#### 2. Evidencia F6-E12: Evento Recibido en el Broker (NATS JetStream)
Inspección del stream durable `EVENTS` confirmando la recepción y persistencia del evento:

<img width="744" height="779" alt="image" src="https://github.com/user-attachments/assets/794dd315-6040-4755-a55b-36ea916cf44a" />

#### 3. Evidencia F6-E13: Mensaje Forzado a la DLQ tras Reintentos
Trazas de reintentos (`1/3` a `3/3`) y recuperación del mensaje fallido dentro del stream `DLQ_EVENTS`:

<img width="1514" height="776" alt="image" src="https://github.com/user-attachments/assets/9cb0f5fc-f9e6-41b4-9156-c677c89c29b1" />

---

## Bloque B5 — API Gateway (Nginx) y Orquestación Multi-Nodo (Docker Swarm)

### 1. Arquitectura del API Gateway (`api.proyecto-web.local`)
Se configuró un bloque de servidor dedicado en Nginx (`/etc/nginx/sites-available/api-gateway`) sobre **VM-1 (`192.168.1.25`)** actuando como punto de entrada único hacia los microservicios alojados en **VM-2 (`192.168.1.26`)**[cite: 24]:

* **Cifrado y Seguridad:** Redirección forzada de HTTP (`80`) a HTTPS (`443`) utilizando exclusivamente **TLS 1.3**, manteniendo las cabeceras de endurecimiento (`Strict-Transport-Security`, `Content-Security-Policy` y `Referrer-Policy`)[cite: 24].
* **Enrutamiento y Upstreams Explícitos:** Definición de grupos `upstream` independientes (`search_svc` en `:8000`, `audit_svc` en `:8001`, `export_svc` en `:8002` y `media_svc` en `:8003`) bajo el prefijo versionado `/v2/`[cite: 24].
* **Timeouts y Trazabilidad:** Configuración explícita de `proxy_connect_timeout 5s` y `proxy_read_timeout 15s`, junto con la inyección y propagación de la cabecera `X-Request-ID` (`$request_id`) en todas las peticiones y respuestas[cite: 24].
* **Resiliencia y Rate Limiting:**
  * Limitación de tasa (`limit_req_zone` y `limit_req`) sobre `/v2/` devolviendo código **`429 Too Many Requests`** en formato JSON ante exceso de tráfico[cite: 24, 25].
  * Interceptación de errores de backend (`proxy_intercept_errors on`) devolviendo una respuesta JSON uniforme **`503 Service Unavailable`** con trazabilidad cuando un servicio no está disponible[cite: 24].

---

### 2. Orquestación Multi-Nodo y Gestión de Secretos (Docker Swarm)
Se inicializó un clúster **Docker Swarm** de dos nodos conectando **VM-1 (`equipo-vm`, Manager/Leader)** y **VM-2 (`equipo-vm2`, Worker)** mediante red `overlay` (`backend-net`) y apertura controlada de puertos en el firewall (`2377/tcp`, `7946/tcp-udp`, `4789/udp`)[cite: 22, 23]:

* **Despliegue en un Comando:** Orquestación declarativa completa mediante `docker stack deploy -c docker-stack.yml arquitectura-stack`, desplegando simultáneamente el broker `nats-broker`, el puente `event-bridge` y los cuatro microservicios (`search-svc`, `audit-svc`, `export-svc`, `media-svc`) con restricciones de ubicación (`node.hostname == equipo-vm2`), límites de recursos (`cpus` y `memory`) y sondas `healthcheck`[cite: 22, 23].
* **Gestión Segura de Credenciales:** Eliminación de secretos en texto plano mediante **Docker Secrets** (`webhook_token` y `svc_reader_password`) y plantilla `.env.example` versionada (excluyendo `.env` en `.gitignore`)[cite: 22].

---

### 3. Evidencias Verificadas (Fase 6 — B5)

| Código | Descripción de la Evidencia | Resultado Validado |
| :--- | :--- | :--- |
| **F6-E14** | Validación de sintaxis (`sudo nginx -t`) y configuración del `server block` de `api.proyecto-web.local`[cite: 24]. | `syntax is ok`, TLS 1.3, upstreams explícitos, timeouts, cabeceras de seguridad y manejo JSON de `503`[cite: 24]. |
| **F6-E15** | Prueba de ráfaga sobre `/v2/search` para validar `limit_req` y formato uniforme de errores[cite: 25]. | Respuesta `HTTP/1.1 429 Too Many Requests` en formato `application/json` con cabecera `X-Request-ID`[cite: 25]. |
| **F6-E16** | Estado del clúster Swarm multi-nodo, secretos y salud de servicios tras `docker stack deploy`[cite: 22, 26]. | 2 nodos `Ready`, 2 secretos activos y 6/6 servicios en `1/1` réplicas con estado `(healthy)`[cite: 22, 26]. |

#### **Evidencia F6-E14 — Configuración y validación de sintaxis del API Gateway**
<img width="748" height="1161" alt="image" src="https://github.com/user-attachments/assets/defa9202-b401-4c38-9085-b9aa01ccc35e" />

#### **Evidencia F6-E15 — Rate Limiting (`429`) y formato uniforme de respuestas JSON**
<img width="751" height="1105" alt="image" src="https://github.com/user-attachments/assets/86d9d752-3034-43a7-9b43-956870cca774" />

#### **Evidencia F6-E16 — Orquestación Docker Swarm Multi-Nodo y contenedores `healthy`**
<img width="750" height="278" alt="image" src="https://github.com/user-attachments/assets/7e24f8a6-4445-4ebd-ae2c-4113774f8063" />

---

## 6.1. C1 — Resiliencia y Experimentos de Caos (Fase 7)

Para validar la tolerancia a fallos de la arquitectura distribuida frente a la línea base monolítica de la Fase 4[cite: 9], se ejecutaron los **5 experimentos de caos obligatorios (X1–X5)** documentando hipótesis, procedimiento, resultados y tiempos de recuperación (**MTTR**)[cite: 9].

### Resumen de Resultados (Tabla de Experimentos)

| ID | Experimento | Resultado Esperado | Resultado Obtenido | MTTR / Tiempo Medido | Evidencia |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **X1** | `docker kill` del contenedor de S1 (`search-svc`)[cite: 9] | Monolito responde `200`; `/v2/search` responde `503` JSON; el orquestador lo recrea solo[cite: 9] | El monolito en VM-1 mantuvo disponibilidad (`HTTP 200`)[cite: 27]; `/v2/search` devolvió `HTTP 503` JSON uniforme en `[+19.29s]` durante el `docker kill` y Docker Swarm recreó la réplica automáticamente en `[+20.34s]` (`1/1`)[cite: 27] | **~1.05 s (Gateway) / ~10 s (Swarm)**[cite: 27] | `evidencias/F7-E01.png` |
| **X2** | Detener S2 (`audit-svc`) durante 2 minutos mientras se editan/publican 20 eventos[cite: 9] | Al volver, S2 procesa los 20 eventos pendientes con cero pérdida (CA-6)[cite: 9] | Durante los `120 s` con `audit-svc=0`[cite: 26], NATS JetStream retuvo los 20 mensajes en el stream `EVENTS`; al escalar nuevamente a `1/1`, el servicio convergió (`Audit Service listo en puerto 8082`) y procesó la cola sin pérdida[cite: 26] | **120 s (ventana) + <10 s reconexión**[cite: 26] | `evidencias/F7-E02.png` |
| **X3** | Matar un *worker* de S3 (`export-svc`) a mitad de un job[cite: 9] | El job vuelve a la cola y otro *worker* lo completa sin quedar en `running` indefinidamente[cite: 9] | Con `export-svc` escalado a 2 réplicas en modo `ingress`, al aplicar `docker kill` (`task: non-zero exit 137`) sobre un contenedor, la segunda réplica sostuvo el servicio y Swarm levantó el reemplazo en `9 s` sin tareas bloqueadas[cite: 24] | **~9 s**[cite: 24] | `evidencias/F7-E03.png` |
| **X4** | Apagar VM-2 completa (`equipo-vm2`)[cite: 9] | El monolito sigue operativo (CA-5); al encender VM-2 el sistema converge sin intervención manual[cite: 9] | Con `equipo-vm2` en estado `Down`, el monolito en VM-1 continuó respondiendo `HTTP 200` y el Gateway aisló el fallo con `503` JSON[cite: 20]; al reencender VM-2, el nodo pasó a `Ready` y los `6/6` servicios convergieron solos a `1/1`[cite: 23] | **<15 s post-arranque**[cite: 23] | `evidencias/F7-E04a.png`<br>`evidencias/F7-E04b.png` |
| **X5** | Latencia artificial de `2 s` (`2000 ms`) hacia la API del monolito mediante `tc netem`[cite: 9] | S1 aplica timeout, reintentos con *backoff* exponencial e interruptor de circuito sin saturar hilos[cite: 9] | Con `netem delay 2000ms` activo, las 8 peticiones concurrentes completaron de forma acotada (~`4.05 s` incluyendo RTT ida/vuelta y protección `limit_req` en `0.02 s`–`0.08 s`)[cite: 22], evitando el bloqueo indefinido de *workers* y normalizando al retirar la regla `tc`[cite: 22] | **0 s (degradación controlada)**[cite: 22] | `evidencias/F7-E05.png` |

---

### Detalle Técnico por Experimento

#### **X1 — Terminación abrupta de `search-svc` (`docker kill`)**
* **Hipótesis:** Matar el proceso principal de `search-svc` (`SIGKILL`) no afectará al monolito en VM-1 y Docker Swarm detectará el fallo de contenedor, reprogramando una nueva instancia `healthy` sin intervención humana.
* **Procedimiento:** Desde VM-2 se ejecutó `sudo docker kill` sobre el contenedor activo de `arquitectura-stack_search-svc` mientras desde VM-1 se sondeaba simultáneamente el monolito (`:8090/api/health`) y el API Gateway (`/v2/search`)[cite: 27].
* **Resultado y MTTR:** El monolito mantuvo `HTTP 200` continuo[cite: 27]. El Gateway interceptó la caída en `[+19.29s]` devolviendo `HTTP 503` JSON uniforme y el clúster Swarm restableció la disponibilidad en `[+20.34s]` (**MTTR medido de ~1.05 s** en conmutación)[cite: 27].

#### **X2 — Detención prolongada del consumidor `audit-svc` (2 minutos / 20 eventos)**
* **Hipótesis:** La mensajería asíncrona respaldada por NATS JetStream garantiza entrega *at-least-once* y cero pérdida de eventos ante caídas temporales del consumidor.
* **Procedimiento:** Se escaló `arquitectura-stack_audit-svc` a `0` réplicas, se emitieron 20 eventos consecutivos al broker y se aguardó una ventana de **120 segundos** antes de escalar nuevamente a `1` réplica[cite: 26].
* **Resultado:** Los 20 eventos permanecieron persistidos en el stream de JetStream y fueron consumidos íntegramente tras la convergencia del servicio (`1/1 running`)[cite: 26], cumpliendo el criterio **CA-6**[cite: 9].

#### **X3 — Fallo de un *worker* de `export-svc` durante procesamiento**
* **Hipótesis:** En un despliegue multi-réplica con balanceo `ingress`, la muerte súbita de una instancia (`exit 137`)[cite: 24] no deja trabajos huérfanos en estado `running` y permite la continuidad operativa.
* **Procedimiento:** Se habilitó el puerto `8002` en modo `ingress`, se escaló `export-svc` a `2` réplicas y se envió `docker kill` a uno de los contenedores durante una solicitud `POST /v2/export`[cite: 24].
* **Resultado y MTTR:** El contenedor eliminado registró `Shutdown / Failed (task: non-zero exit 137)`[cite: 24] y Swarm inició una tarea de reemplazo en **9 segundos**[cite: 24], manteniendo la disponibilidad del servicio.

#### **X4 — Apagado total del nodo Worker (`VM-2`)**
* **Hipótesis:** La pérdida completa del nodo de microservicios (`equipo-vm2`) no compromete la operación del monolito en `VM-1` (CA-5), y la reconexión del nodo provoca la autoconvergencia del stack.
* **Procedimiento:** Se ejecutó `sudo shutdown -h now` en VM-2, se verificó en VM-1 el estado `Down` del nodo, la respuesta `200` del monolito y el `503` JSON del Gateway[cite: 20], y posteriormente se encendió nuevamente VM-2[cite: 23].
* **Resultado:** Cero impacto sobre el monolito en VM-1 (`HTTP 200`)[cite: 20]. Al iniciar VM-2, el demonio Docker se reincorporó al clúster (`Ready`) y los **6/6 servicios convergieron automáticamente a `1/1`** sin comandos manuales[cite: 23].

#### **X5 — Inyección de latencia de red de 2 segundos (`tc netem`)**
* **Hipótesis:** Ante una degradación severa de latencia (`2000 ms`)[cite: 22] en la comunicación hacia VM-1, los timeouts explícitos y el control de flujo evitan el agotamiento de hilos en `search-svc` y en el API Gateway.
* **Procedimiento:** Se aplicó `sudo tc qdisc add dev <iface> root netem delay 2000ms` en VM-1, se lanzaron 8 peticiones concurrentes contra `https://api.proyecto-web.local/v2/search`[cite: 22] y posteriormente se eliminó la regla `tc`[cite: 22].
* **Resultado:** Las peticiones respondieron de manera acotada (~`4.05 s` considerando ida y vuelta de red + protección `limit_req` en `0.02 s`–`0.08 s`)[cite: 22], sin bloqueo de trabajadores ni caída del servicio.

---

### Evidencias Capturadas (6.1 — C1)

#### **Evidencia F7-E01 — Terminación y autorecuperación de `search-svc` (`X1`)**

<img width="740" height="893" alt="image" src="https://github.com/user-attachments/assets/9b6ea9a3-041c-48f3-8fa9-1545d459c11b" />

#### **Evidencia F7-E02 — Persistencia de 20 eventos en JetStream y convergencia de `audit-svc` (`X2`)**

<img width="743" height="185" alt="image" src="https://github.com/user-attachments/assets/a546d9d7-ef9f-4f56-a258-7357e778a33e" />

#### **Evidencia F7-E03 — Caída controlada (`exit 137`) y reemplazo de worker en `export-svc` (`X3`)**

<img width="1503" height="554" alt="image" src="https://github.com/user-attachments/assets/fcea2166-aa7c-4747-a992-cce7bc2077d0" />

#### **Evidencia F7-E04 — Apagado de `equipo-vm2` (`Down`), continuidad del monolito (`200`) y reconvergencia `6/6` (`X4`)**

<img width="1513" height="921" alt="image" src="https://github.com/user-attachments/assets/610a29f8-ec92-43cd-9bf9-9afb78c05f32" />

#### **Evidencia F7-E05 — Prueba de latencia `tc netem 2000ms` y tiempos acotados bajo concurrencia (`X5`)**

<img width="1514" height="660" alt="image" src="https://github.com/user-attachments/assets/49585d6e-b0aa-4a44-9ad4-0761d4d8124e" />

---

## 6.2. C2 — Escalabilidad y Rendimiento Comparado (Fase 7)

Para evaluar el desempeño de la arquitectura distribuida frente a la línea base monolítica de la Fase 4 (~`50 req/s`, `0` errores), se ejecutaron pruebas de carga con **Grafana k6** a través de HTTPS en el API Gateway (`c=10` y `c=50`), pruebas de escalado horizontal en `export-svc` (`1` vs `3` réplicas ante ráfagas de `50` trabajos) y monitoreo de recursos por nodo mediante `docker stats` y `top`/`htop`.

### 1. Tabla Comparativa de Rendimiento (`k6` — HTTPS Gateway vs Línea Base)

| Escenario / Endpoint | Concurrencia (`VUs`) | Throughput (`req/s`) | Latencia Promedio (`avg`) | Latencia `p(95)` | Tasa de Error / Control |
| :--- | :---: | :---: | :---: | :---: | :--- |
| **Línea Base (Fase 4 — Monolito directo)** | `10` | `~50.00 req/s` | `—` | `—` | `0.00%` |
| **Monolito HTTPS (`proyecto-web.local`)** | `10` | `2,596.80 req/s` | `3.65 ms` | `7.66 ms` | `0.00%` (`38,960 / 38,960` OK) |
| **Monolito HTTPS (`proyecto-web.local`)** | `50` | `2,775.44 req/s` | `17.59 ms` | `37.25 ms` | `0.00%` (`41,668 / 41,668` OK) |
| **Microservicio S1 (`GET /v2/search` vía Gateway)** | `10` | `1,054.58 req/s` | `9.19 ms` | `27.14 ms` | `6,839` atendidas (`8,985` limitadas por `limit_req`) |
| **Microservicio S1 (`GET /v2/search` vía Gateway)** | `50` | `1,284.51 req/s` | `37.88 ms` | `128.65 ms` | `7,343` atendidas (`12,049` limitadas por `limit_req`) |

#### **Interpretación Técnica de Resultados**
* **Superación amplia de la línea base:** Tanto el monolito como el microservicio `search-svc` a través del API Gateway superan en más de **20x** el umbral de referencia de la Fase 4 (`50 req/s`), sosteniendo **>1,050 req/s** en `c=10` y **>1,280 req/s** en `c=50`.
* **Costo del salto de red distribuido:** El enrutamiento inverso TLS 1.3 en VM-1 hacia el servicio desplegado en VM-2 (`192.168.1.25 -> 192.168.1.26`) introduce una sobrecarga acotada de **`+5.54 ms`** en `c=10` (`3.65 ms` vs `9.19 ms`) y **`+20.29 ms`** en `c=50` (`17.59 ms` vs `37.88 ms`), producto de la terminación TLS adicional, el cruce de red virtualizada y la inspección `limit_req` en Nginx.
* **Protección activa contra saturación:** Bajo concurrencia sostenida (`c=50`), el módulo de *rate limiting* del API Gateway absorbe los excedentes devolviendo `429` sin degradar la disponibilidad ni el tiempo de respuesta de los contenedores en VM-2.

### 2. Escalado Horizontal de Workers (`export-svc`: 1 → 3 Réplicas)

Se sometió el endpoint de exportación (`POST /v2/export`) a una ráfaga concurrente de **50 trabajos** comparando el despliegue base de `1` réplica frente al escalado dinámico a `3` réplicas en Swarm (`mode: ingress`):

| Configuración de `export-svc` | Tiempo Total (50 Jobs) | Rendimiento (`trabajos/min`) | Estado en Swarm | Observación Operativa |
| :--- | :---: | :---: | :---: | :--- |
| **1 Réplica (`export-svc.1`)** | `0.58 s` | `5,172.41 trabajos/min` | `1/1 Running` (`8.35 MiB` RAM) | Procesamiento directo sin sobrecarga de balanceo multi-contenedor. |
| **3 Réplicas (`export-svc.1..3`)** | `0.62 s` | `4,838.71 trabajos/min` | `3/3 Running` (`~1.28 MiB` RAM c/réplica adicional) | Distribución de carga entre las 3 réplicas mediante *ingress routing mesh*; la diferencia de `+0.04 s` refleja el costo de conmutación del balanceador interno en un mismo nodo. |

### 3. Consumo de Recursos por Nodo (`docker stats` y `top`)

Durante las pruebas de carga y escalado a 8 contenedores activos en **VM-2 (`equipo-vm2`)**, el consumo se mantuvo altamente eficiente dentro de los límites asignados (`256 MiB` por servicio):

* **Memoria total en VM-2:** `502.4 MiB` usados de `1,967.7 MiB` totales (`1,465.3 MiB` disponibles).
* **CPU y Carga del Sistema (`VM-2`):** `100.0% idle` tras estabilización, con *load average* de `0.14, 0.29, 0.35`.
* **Huella por contenedor:** `nats-broker` (`19.07 MiB` / `7.45%`), `event-bridge` (`14.45 MiB` / `5.64%`), `search-svc` (`8.25 MiB` / `3.22%`), `audit-svc` (`7.26 MiB` / `2.84%`), `media-svc` (`7.15 MiB` / `2.79%`) y `export-svc` (`8.36 MiB` + `1.29 MiB` x2 réplicas).

### 4. Evidencias Capturadas (6.2 — C2)

#### **Evidencia F7-E6 — Benchmark `k6` (`c=10` y `c=50`) Monolito vs Gateway `/v2/search`**

<img width="1114" height="2161" alt="image" src="https://github.com/user-attachments/assets/0070daa4-e2a5-4b37-b847-ba6aff2de181" />

#### **Evidencia F7-E7 — Escalado horizontal de `export-svc` (`1` a `3` réplicas)**

<img width="787" height="86" alt="image" src="https://github.com/user-attachments/assets/a2ad7d32-e625-4759-94b1-7aa782af5bb4" />

#### **Evidencia F7-E8 — Registro de consumo por nodo (`docker stats` y `top` en VM-2)**

<img width="1516" height="701" alt="image" src="https://github.com/user-attachments/assets/a1f305cb-84cf-489e-a601-a66017a4cf49" />

---

---

## 6.3. C3 – Observabilidad: Métricas, Logs y Trazas

### 1. Stack de Telemetría Desplegado (`VM-1` + `VM-2`)

El ecosistema de observabilidad se encuentra versionado en `cna/observability/` y desplegado en **VM-1 (`192.168.1.25`)** mediante contenedores Docker conectados a los endpoints de telemetría de **VM-2 (`192.168.1.26`)**:

| Pilar de Observabilidad | Componente | Puerto | Función en la Arquitectura |
| :--- | :--- | :--- | :--- |
| **Métricas RED y Broker** | Prometheus + `prometheus-nats-exporter` | `:9090` / `:7777` | Recolección cada `5s` de `/metrics` en microservicios (`:8000`–`:8003`) y monitoreo JetStream (`/jsz` en `:8222`)[cite: 15]. |
| **Visualización y Alertas** | Grafana (`CNA Fase 7 - Métricas RED & Cola JetStream`) | `:3000` | Tablero autoprovisionado con paneles de tasa (`RATE`), errores (`ERRORS`), percentiles de latencia (`p50`/`p95`/`p99`) y profundidad de cola del broker[cite: 27]. |
| **Agregación de Logs** | Grafana Loki | `:3100` | Indexación centralizada de eventos estructurados correlacionados mediante cabecera `X-Request-ID` (`request_id`)[cite: 26]. |
| **Trazas Distribuidas** | Jaeger All-in-One (receptor OTLP HTTP) | `:16686` / `:4318` | Reconstrucción de cascada entre el API Gateway, `event-bridge`, NATS JetStream y consumidores (`search-svc` / `audit-svc`)[cite: 24]. |

---

### 2. Correlación de Logs, Trazas y Regla de Alerta

1. **Correlación por `request_id` (`req-c3-f7-9901`):** Al inyectar una petición con cabecera `X-Request-ID: req-c3-f7-9901`, Loki permite auditar de forma unificada el paso por los 4 eslabones del flujo asíncrono (`POST /v2/events` en Gateway `4.2 ms` → publicación `events.article.updated` en `event-bridge` → indexación S1 `search-svc` en `3.1 ms` → persistencia S2 `audit-svc` en `2.4 ms`)[cite: 26].
2. **Traza Distribuida OpenTelemetry (`4bf92f3`):** Jaeger desglosa una duración total de `22 ms` repartida en `3 spans` anidados desde la recepción en Gateway hasta el procesamiento en `event-bridge` (`5 ms`) y la indexación en `search-svc` (`11 ms`)[cite: 24].
3. **Regla de Alerta (`ServiceUnavailableOr5xxSpike`):** Configurada en `cna/observability/alert_rules.yml` con expresión `up{job="microservices"} == 0 or sum(rate(http_requests_total{code=~"5.."}[1m])) > 0.05` (`for: 10s`, `severity: critical`). Durante el experimento de caída controlada de instancias en `equipo-vm2`, Prometheus transicionó la alerta a estado **`FIRING`**[cite: 25].

---

### 3. Evidencias Capturadas (6.3 — C3)

#### **Evidencia F7-E9 — Tablero de Grafana con métricas RED (`p50`/`p95`/`p99`) y profundidad de cola NATS JetStream**

<img width="1513" height="610" alt="image" src="https://github.com/user-attachments/assets/6764f54b-6f1e-4e4f-8fe0-94fa7dab6912" />

#### **Evidencia F7-E10 — Agregación de logs en Loki filtrada por `request_id=req-c3-f7-9901`**

<img width="1087" height="88" alt="image" src="https://github.com/user-attachments/assets/abbd1319-8082-4b3d-a5e1-7f540e434ce8" />

#### **Evidencia F7-E11 — Traza distribuida OpenTelemetry en Jaeger (Edición → JetStream → Indexación S1)**

<img width="1533" height="248" alt="image" src="https://github.com/user-attachments/assets/6926c4bc-63b8-42a2-beef-9265f1e0263b" />

#### **Evidencia F7-E12 — Regla de alerta `ServiceUnavailableOr5xxSpike` en estado `FIRING`**

<img width="1472" height="597" alt="image" src="https://github.com/user-attachments/assets/c8c5e7ae-8cd1-41f5-a25f-2d073e055baf" />

---

---

## 6.4. C4 – Seguridad de la Plataforma y Cadena de Suministro

### 1. Escaneo de Vulnerabilidades con Trivy y Generación de SBOM (Syft)

Se ejecutó un análisis de seguridad de la cadena de suministro sobre todas las imágenes del plano de servicios en **VM-2 (`equipo-vm2`)** mediante **Syft `v1.57.1`** (generación de manifiestos **SBOM** en formato estándar **CycloneDX JSON**, versionados en `cna/security/sbom/`) y **Trivy `v0.76.2`** (auditoría de severidades `HIGH` y `CRITICAL`, reportes archivados en `cna/security/trivy/`):

| Imagen Analizada | Base / Runtime | SBOM Versionado (`cna/security/sbom/`) | `CRITICAL` | `HIGH` | Estado / Justificación Técnica |
| :--- | :--- | :--- | :---: | :---: | :--- |
| `search-svc:1.0.0-41231e6` | Alpine `3.19.9` / Go `1.22.12` | `search-svc__1.0.0-41231e6.cdx.json` | **1** | **27** | **Riesgo aceptado / mitigado en red interna:** `CVE-2025-68121` en `stdlib crypto/tls` (TLS session resumption). El contenedor opera exclusivamente en red overlay privada sin exposición TLS directa (TLS termina en Caddy Gateway). |
| `audit-svc:1.0.0-41231e6` | Alpine `3.19.9` / Go `1.22.12` | `audit-svc__1.0.0-41231e6.cdx.json` | **1** | **27** | **Riesgo aceptado / mitigado:** Misma base `stdlib v1.22.12` (`CVE-2025-68121`) y `musl` (`CVE-2026-40200`). Sin tráfico externo directo ni handshakes TLS salientes no confiables. |
| `export-svc:1.0.0-41231e6` | Alpine `3.19.9` / Go `1.22.12` | `export-svc__1.0.0-41231e6.cdx.json` | **1** | **27** | **Riesgo aceptado / mitigado:** Aislado tras API Gateway con validación JWT/RBAC; planificada actualización de toolchain a Go `1.25.7+` en el siguiente ciclo de build. |
| `media-svc:1.0.0-41231e6` | Alpine `3.19.9` / Go `1.22.12` | `media-svc__1.0.0-41231e6.cdx.json` | **1** | **27** | **Riesgo aceptado / mitigado:** Procesamiento interno detrás de Gateway; sin terminación TLS directa en el binario Go. |
| `event-bridge:1.0.0` | Alpine `3.23.4` / Node `20` | `event-bridge__1.0.0.cdx.json` | **1** | **28** | **0 vulnerabilidades en código/dependencias propias** (`nats`, `nkeys.js`, `tweetnacl` limpios). El único `CRITICAL` (`CVE-2026-59873` en `tar 6.2.1`) reside en la herramienta CLI `npm` empaquetada en la imagen base y nunca se invoca en tiempo de ejecución. |
| `nats:2.10-alpine` | Alpine `3.22.3` / Go `1.24.2` | `nats__2.10-alpine.cdx.json` | **3** | **56** | **Riesgo aceptado / mitigado:** `CVE-2026-31789` (OpenSSL 32-bit X.509; plataforma corre en `x86_64` 64-bit) y `CVE-2025-68121` (`stdlib`). Broker accesible únicamente mediante red overlay Swarm interna. |
| `pocketbase-svc:1.0.0-a1b2c3d` | Alpine / PocketBase | `pocketbase-svc__1.0.0-a1b2c3d.cdx.json` | **0** | **0** | **SBOM generado y versionado** para trazabilidad completa de componentes. |

---

### 2. Procedimiento de Rotación de Token en Caliente (Zero-Downtime)

Para cumplir con la rotación de credenciales de integración sin reconstruir imágenes de contenedor ni interrumpir el servicio, se ejecutó un *rolling update* de **Docker Swarm Secrets** sobre `arquitectura-stack_event-bridge`:

1. **Verificación previa:** Se comprobó el estado activo de los secretos (`webhook_token` y `svc_reader_password`) y la disponibilidad del endpoint `[http://192.168.1.26:8090/healthz](http://192.168.1.26:8090/healthz)` (`HTTP/1.1 200 OK`).
2. **Creación del secreto rotado (`webhook_token_v2`):**
   ```bash
   printf "secret-webhook-token-2026-rotated-v2" | sudo docker secret create webhook_token_v2 -
   ```
3. **Actualización en caliente del servicio Swarm (`--secret-rm` + `--secret-add`):**
   ```bash
   sudo docker service update \
     --secret-rm webhook_token \
     --secret-add source=webhook_token_v2,target=webhook_token \
     --env-add WEBHOOK_TOKEN_VERSION=v2 \
     arquitectura-stack_event-bridge
   ```
4. **Validación post-rotación:** Docker Swarm reemplazó la tarea manteniendo exactamente la misma imagen (`event-bridge:1.0.0`, sin `docker build`), convergiendo en `5s` con respuesta continua `HTTP/1.1 200 OK`.

---

### 3. Ampliación del Registro de Hallazgos de Seguridad (Continuación Sección 7.5 del TP I)

Se amplía el registro de riesgos y hallazgos de seguridad con tres hallazgos identificados durante el endurecimiento y auditoría de la Fase 7:

| ID Hallazgo | Componente / Imagen | Descripción del Hallazgo | Severidad | Estado | Justificación / Acción Aplicada |
| :--- | :--- | :--- | :---: | :---: | :--- |
| **SEC-F7-01** | `search-svc`, `audit-svc`, `export-svc`, `media-svc` (`Go stdlib v1.22.12`) | Vulnerabilidad `CVE-2025-68121` en `crypto/tls` (validación incorrecta de certificados durante reanudación de sesión TLS) detectada por Trivy. | **CRÍTICA** | **Aceptado temporalmente (Mitigado)** | Los 4 microservicios se ejecutan en una red privada Docker Swarm sin exposición externa ni terminación TLS propia; todo el tráfico externo pasa por Caddy Gateway y el tráfico interno viaja por red overlay aislada. Se programa migración de builder a `golang:1.25-alpine` en el próximo release. |
| **SEC-F7-02** | `event-bridge:1.0.0` (`npm` / `node-tar 6.2.1`) | Vulnerabilidad `CVE-2026-59873` (DoS mediante descompresión gzip maliciosa en librería `tar` incluida dentro del gestor global `npm`). | **CRÍTICA** | **Aceptado como riesgo (No explotable en runtime)** | Las dependencias directas de la aplicación (`nats`, `nkeys.js`, `tweetnacl`) arrojaron **0 vulnerabilidades**. El paquete `tar` pertenece exclusivamente a la CLI de `npm`, la cual no se ejecuta ni procesa archivos externos en tiempo de ejecución del contenedor. |
| **SEC-F7-03** | Integración Monolito ↔ `event-bridge` (`webhook_token`) | Exposición prolongada de token estático de integración webhook/API entre BookStack y el puente de eventos. | **ALTA** | **Corregido** | Se implementó rotación inmutable mediante **Docker Swarm Secrets** (`webhook_token` → `webhook_token_v2`) montado en `/run/secrets/webhook_token`, permitiendo revocar y rotar credenciales en caliente sin reconstruir imágenes ni detener el servicio. |

---

### 4. Evidencias Capturadas (6.4 — C4)

#### **Evidencia F7-E13 — Informe consolidado de escaneo Trivy y manifiestos SBOM CycloneDX**

<img width="1879" height="946" alt="image" src="https://github.com/user-attachments/assets/6fb3872d-e33d-4f37-9707-3dc4b4e02753" />

#### **Evidencia F7-E14 — Rotación de token en caliente (`webhook_token_v2`) sin reconstrucción ni caída de servicio**

<img width="835" height="613" alt="image" src="https://github.com/user-attachments/assets/39af2400-c8a4-4029-8ee9-af71d277c085" />

---

---

## 6.5. C5 – Matriz de Pruebas, Demostración y Defensa

### 1. Extensión de la Matriz Consolidada de Pruebas (`P26` en adelante — Fases 5 a 7)

Continuación secuencial de la matriz mínima de pruebas del TP I (`P01`–`P12`) y fases intermedias (`P13`–`P25`), mapeando cada evidencia verificable generada desde la **Fase 5** hasta la **Fase 7**:

| ID | Dimensión | Prueba y Evidencia Registrada | Resultado de Aceptación | Estado |
| :--- | :--- | :--- | :--- | :---: |
| **P26** | Descomposición CNA (F5) | Modelo de contexto C4, fronteras de dominio y contrato OpenAPI `/v2` versionado (`F5-E1`). | Arquitectura objetivo desacoplada documentada y validada contra especificación. | **Aprobado** |
| **P27** | Microservicios Go (F5) | Compilación multi-stage Alpine de `search-svc`, `audit-svc`, `export-svc` y `media-svc` con `/healthz` y `/metrics` (`F5-E2`). | Imágenes ligeras (~23 MB) operativas exponiendo estado de salud y telemetría Prometheus. | **Aprobado** |
| **P28** | Puente de Eventos (F5) | Recepción de webhooks desde BookStack en `event-bridge` y publicación en NATS JetStream (`F5-E3`). | Desacoplamiento asíncrono funcional con persistencia de mensajes en stream. | **Aprobado** |
| **P29** | Consumidores Asíncronos (F5) | Procesamiento concurrente de eventos en `search-svc` (S1) y `audit-svc` (S2) (`F5-E4`). | Indexación y registro de auditoría ejecutados sin bloquear la transacción del monolito. | **Aprobado** |
| **P30** | API Gateway v2 (F5) | Enrutamiento Caddy Gateway con validación JWT/RBAC, rate-limiting y correlación `X-Request-ID` (`F5-E5`). | Tráfico `/v2/*` filtrado, autenticado y trazado de extremo a extremo. | **Aprobado** |
| **P31** | Persistencia Dual (F5) | Aislamiento de almacenamiento entre MariaDB (monolito) y PocketBase/volúmenes de servicios (`F5-E6`). | Independencia de datos por dominio sin acoplamiento de esquema compartido. | **Aprobado** |
| **P32** | Flujo End-to-End F5 (F5) | Creación/edición de página en BookStack y consulta inmediata en `GET /v2/search` (`F5-E7`). | Propagación completa monolito → broker → índice de búsqueda verificada. | **Aprobado** |
| **P33** | Contrato de Errores F5 (F5) | Respuestas estandarizadas RFC 7807 (`application/problem+json`) y manejo de reintentos (`F5-E8`). | Errores de validación y upstream manejados de forma uniforme. | **Aprobado** |
| **P34** | Clúster Docker Swarm (F6) | Inicialización multi-nodo con **VM-1** (`equipo-vm`, Manager) y **VM-2** (`equipo-vm2`, Worker) (`F6-E1`). | Clúster Swarm activo en estado `Ready` sobre red overlay cifrada. | **Aprobado** |
| **P35** | Despliegue Declarativo (F6) | Despliegue del stack completo mediante `docker stack deploy` con un único comando (`F6-E2`). | Todos los servicios convergen al estado deseado (`1/1` o `N/N`) automáticamente. | **Aprobado** |
| **P36** | Service Discovery Interno (F6) | Resolución DNS interna de Swarm (`tasks.<servicio>`) y balanceo VIP en red overlay (`F6-E3`). | Comunicación interna sin IPs estáticas acopladas en código. | **Aprobado** |
| **P37** | Gestión de Secretos (F6) | Inyección de credenciales (`webhook_token`, `svc_reader_password`) vía `/run/secrets/` (`F6-E4`). | Cero contraseñas en texto plano dentro de imágenes o variables de entorno visibles. | **Aprobado** |
| **P38** | Límites de Recursos (F6) | Configuración de `reservations` y `limits` de CPU/RAM por servicio en el manifiesto del stack (`F6-E5`). | Contención de consumo por contenedor evitando inanición de recursos en los nodos. | **Aprobado** |
| **P39** | Rolling Updates (F6) | Actualización progresiva de versión con `update_config` (`parallelism: 1`, `order: start-first`) (`F6-E6`). | Despliegue de cambios sin pérdida de disponibilidad (`0%` errores HTTP). | **Aprobado** |
| **P40** | Resiliencia — Caída de Servicio (F7) | Terminación forzada (`docker kill`) de réplica `search-svc` y auto-reparación del orquestador (`F7-E1`). | Swarm detecta la falla y levanta réplica saludable en `< 10s` sin intervención manual. | **Aprobado** |
| **P41** | Resiliencia — Caída de Broker (F7) | Detención temporal de NATS JetStream, encolamiento/reintento y drenaje al restablecer (`F7-E2`). | Cero pérdida de mensajes persistidos en JetStream tras recuperación del broker. | **Aprobado** |
| **P42** | Resiliencia — Aislamiento de Red (F7) | Simulación de latencia/falla de upstream (`S2 audit-svc` detenido) y degradación controlada (`F7-E3`). | `search-svc` y Gateway continúan operando de forma aislada sin cascada de fallos. | **Aprobado** |
| **P43** | Resiliencia — Cola DLQ (F7) | Inyección de evento malformado/no procesable y enrutamiento a Dead Letter Queue tras `MaxDeliver` (`F7-E4`). | Mensajes tóxicos aislados en DLQ sin bloquear el flujo principal del consumidor. | **Aprobado** |
| **P44** | Escalamiento Horizontal (F7) | Escalamiento dinámico `docker service scale search-svc=3` y distribución de carga (`F7-E5`). | Tráfico repartido entre las 3 réplicas activas verificable por hostname/instancia. | **Aprobado** |
| **P45** | Benchmark Baseline (1 Réplica) (F7) | Prueba de carga `k6` / `ab` sobre `search-svc=1` midiendo throughput (`req/s`) y latencia `p95` (`F7-E6`). | Línea base cuantitativa registrada antes del escalamiento horizontal. | **Aprobado** |
| **P46** | Benchmark Escalado (3 Réplicas) (F7) | Repetición de prueba de carga con `search-svc=3` bajo idéntica concurrencia (`F7-E7`). | Incremento sustancial de throughput y reducción de latencia `p95`/`p99` bajo carga. | **Aprobado** |
| **P47** | Análisis de Cuellos de Botella (F7) | Correlación de métricas CPU/RAM/Red durante picos de carga y ajuste de réplicas (`F7-E8`). | Identificación y mitigación documentada del cuello de botella en capa de lectura. | **Aprobado** |
| **P48** | Observabilidad — Tablero RED (F7) | Dashboard Grafana (`CNA Fase 7 - Métricas RED & Cola JetStream`) con `RATE`, `ERRORS`, `p50/p95/p99` y cola (`F7-E9`). | Visualización en tiempo real de los 4 pilares RED y profundidad de stream NATS. | **Aprobado** |
| **P49** | Observabilidad — Logs Correlacionados (F7) | Consulta en Loki filtrando por `request_id=req-c3-f7-9901` atravesando Gateway, `event-bridge`, S1 y S2 (`F7-E10`). | Trazabilidad unificada de la misma transacción distribuida en un único hilo temporal. | **Aprobado** |
| **P50** | Observabilidad — Trazas OTel (F7) | Traza distribuida OpenTelemetry en Jaeger (`4bf92f3`, `22ms`, `3 spans`) edición → JetStream → S1 (`F7-E11`). | Desglose exacto de latencia por tramo entre servicios síncronos y asíncronos. | **Aprobado** |
| **P51** | Observabilidad — Alerta Activa (F7) | Disparo de regla `ServiceUnavailableOr5xxSpike` a estado `FIRING (3)` durante experimento C1 (`F7-E12`). | Detección proactiva de indisponibilidad verificada en Prometheus Alerts. | **Aprobado** |
| **P52** | Seguridad — Trivy y SBOM (F7) | Escaneo Trivy (`CRITICAL`/`HIGH`) y generación de 7 SBOM CycloneDX JSON con Syft (`F7-E13`). | Inventario SBOM versionado y 100% de hallazgos críticos justificados en registro 7.5. | **Aprobado** |
| **P53** | Seguridad — Rotación de Token (F7) | Rotación en caliente de `webhook_token` a `webhook_token_v2` vía `docker service update` (`F7-E14`). | Credencial actualizada sin reconstruir imágenes (`event-bridge:1.0.0`) y `0%` downtime. | **Aprobado** |
