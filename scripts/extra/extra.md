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

### 5.3. B3 – Contenerización con Buenas Prácticas

En esta sección se rediseñó la estrategia de contenerización de los microservicios aplicando un enfoque de mínima superficie de ataque y optimización de recursos, cumpliendo con los siguientes criterios:

*   **Dockerfile Multi-stage:** Se implementó un *builder* temporal para compilar dependencias (`wheels`), dejando una imagen final `python:3.11-slim` limpia y ligera.
*   **Seguridad:** Ejecución con el usuario sin privilegios `appuser` (no-root) y sistema de archivos montado como solo lectura (`read_only: true`), utilizando `/tmp` para escrituras efímeras.
*   **Tolerancia a fallos:** Incorporación de la instrucción `HEALTHCHECK` en Dockerfile, sondeando el endpoint `/healthz` de forma nativa.
*   **Versionamiento:** Despliegue seguro prohibiendo la etiqueta `:latest`. Se utilizó versionamiento semántico ligado al hash del commit (Ej: `1.0.0-eba4c0b`).
*   **Límites (Cgroups):** Contenedores restringidos por diseño a `0.25` CPUs y `128m` de memoria RAM, garantizando que el sumatorio (512MB RAM) no sature el nodo.

<details open>
<summary>📸 <b>Evidencia F6-E9: Tabla de Imágenes y Tamaño</b></summary>
<br>

Listado de los microservicios compilados, mostrando el uso estricto de versionamiento semántico y la reducción de tamaño de las imágenes.

![Evidencia F6-E9](./evidencias/F6-E9-out1.png)

</details>

<details open>
<summary>📸 <b>Evidencia F6-E10: Inspección de Seguridad y Límites (docker inspect)</b></summary>
<br>

Validación mediante `docker inspect` al contenedor `search-svc` demostrando: usuario `appuser`, límite de RAM en bytes (128MB), límite de CPU (0.25), `ReadonlyRootfs: true` y la definición explícita del `HEALTHCHECK`.

![Evidencia F6-E10](./evidencias/F6-E10-out1.png)

</details>
