# Bloque I – Fase 5: Diseño y Descomposición del Monolito

## 4.1. A1 – Análisis del Monolito y Mapa de Contextos

### Evidencia F5-E1: Inventario de módulos funcionales y tablas
A continuación, se presenta la consulta a la base de datos (SQLite) que lista las tablas/colecciones de PocketBase mapeadas a sus respectivos módulos funcionales.

<img width="1093" height="385" alt="image" src="https://github.com/user-attachments/assets/20517bb6-2f7d-44bd-b638-0ea952755abf" />


---

### Evidencia F5-E2: Matriz de Acoplamiento

Dado que la arquitectura base utiliza PocketBase (SQLite), el acoplamiento se evalúa entre los servicios lógicos consumidos y las colecciones internas del motor. Las tablas compartidas o de uso cruzado determinarán la complejidad de extracción.

| Módulo / Servicio Lógico | Colección / Tabla (PocketBase) | Operaciones (CRUD) | Nivel de Acoplamiento | Justificación del Acoplamiento |
| :--- | :--- | :--- | :--- | :--- |
| **Módulo de Autenticación** | `users`, `_admins` | Lectura (R), Actualización (U) | **Alto** | El servicio de inicio de sesión de la aplicación depende estrictamente de la validación contra estas tablas mediante la API de PocketBase. |
| **Integraciones Sociales** | `_externalAuths` | Creación (C), Lectura (R) | **Medio** | Se acopla solo si se habilita el inicio de sesión con proveedores externos (Google, GitHub, etc.). |
| **Módulo de Formularios / CRM** | `contactos` | Creación (C), Lectura (R), Actualización (U), Borrado (D) | **Alto** | La interfaz de usuario que captura o lista contactos realiza peticiones REST/Realtime directamente hacia esta colección específica. |
| **Core (PocketBase Engine)** | `_collections`, `_migrations`, `_params` | Todas (CRUD) | **Alto (Interno)** | Son tablas de configuración estrictamente acopladas al motor interno de PocketBase para definir el esquema de la base de datos. |

---

### Evidencia F5-E3: Mapa de Contextos (DDD)

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

````

# 4.2. A2 – Registro de Decisiones de Arquitectura (ADR-006 a ADR-010)

A continuación, se documentan las decisiones arquitectónicas para la extracción de microservicios desde el monolito BookStack.

---

## ADR-006: Estrategia de Migración e Integración con el Monolito

* **Estatus:** Aceptado.
* **Contexto:** El monolito BookStack se encuentra en producción y debe continuar operando sin interrupciones mientras se extraen progresivamente funcionalidades hacia nuevos microservicios.
* **Decisión:** Implementar el **Patrón Strangler Fig** integrando los sistemas exclusivamente mediante API REST (para consultas síncronas) y Webhooks (para notificaciones asíncronas), sin realizar modificaciones en el código fuente de BookStack. Un API Gateway (Nginx) actuará como enrutador frontal para desviar el tráfico gradualmente hacia los nuevos servicios (`/v2/*`).
* **Alternativa Descartada:** *Big Bang Rewrite* o modificación del código fuente del monolito. Se descartó porque modificar el código PHP de BookStack aumenta el riesgo de regresiones, rompe la compatibilidad con futuras actualizaciones del producto original y requiere mayor tiempo de desarrollo.
* **Consecuencias:** 
  * (+) Reducción drástica del riesgo de caída del sistema actual.
  * (+) Facilita actualizaciones del monolito sin afectar nuestra arquitectura.
  * (-) Requiere la implementación de un componente intermediario (`event-bridge`) para traducir y enrutar los Webhooks hacia el bus de mensajería.

---

## ADR-007: Selección del Broker de Mensajes

* **Estatus:** Aceptado.
* **Contexto:** Se necesita un sistema de mensajería para entregar de manera confiable los eventos generados por el `event-bridge` hacia los consumidores (servicios de búsqueda, auditoría y exportación), asegurando que no se pierdan mensajes si un servicio se reinicia.
* **Decisión:** Seleccionar **RabbitMQ** (o alternativamente **NATS JetStream** si el consumo de recursos lo requiere) como broker principal para la comunicación asíncrona. Se configurarán colas durables para garantizar la persistencia de los eventos.
* **Alternativa Descartada:** *Redis Streams*. Aunque es muy rápido y consume poca RAM, su modelo de persistencia y enrutamiento avanzado no es tan robusto nativamente como el de RabbitMQ para garantizar "cero pérdida de eventos" ante caídas prolongadas de los consumidores en infraestructuras limitadas.
* **Consecuencias:** 
  * (+) Garantía de entrega (at-least-once) mediante confirmaciones (ACKs).
  * (+) Desacoplamiento total entre la emisión de webhooks y el procesamiento.
  * (-) Incremento en el consumo de memoria RAM de la máquina virtual (VM-2) por el motor de RabbitMQ.

---

## ADR-008: Plataforma de Orquestación

* **Estatus:** Aceptado.
* **Contexto:** La infraestructura proporcionada está fuertemente restringida: máquinas virtuales (VMware Workstation) con apenas 2 vCPU y 4 GB de RAM.
* **Decisión:** Utilizar **Docker Compose** en un único nodo de servicios (VM-2) para gestionar los contenedores de los microservicios y bases de datos. 
* **Alternativa Descartada:** *Docker Swarm* y *k3s/Kubernetes*. Aunque ofrecen alta disponibilidad y características Cloud Native superiores, el *overhead* (consumo de CPU/RAM de los agentes y plano de control) de un clúster Kubernetes (incluso ligero como k3s) saturaría los 4 GB de RAM disponibles, dejando sin recursos a los microservicios y bases de datos.
* **Consecuencias:** 
  * (+) Máximo aprovechamiento de los recursos (RAM/CPU) para las cargas de trabajo reales.
  * (+) Despliegue simple y reproducible.
  * (-) Carencia de alta disponibilidad automática ante la caída del nodo físico; la autorrecuperación se limita al reinicio de contenedores dentro de la misma VM (mediante `restart: always`).

---

## ADR-009: Base de Datos por Servicio

* **Estatus:** Aceptado.
* **Contexto:** Los microservicios a extraer (búsqueda, auditoría, exportación) requieren persistencia propia para funcionar de manera independiente y escalar, evitando el acoplamiento al esquema actual de BookStack.
* **Decisión:** Implementar el patrón **Database per Service**. Queda estrictamente prohibido que los microservicios (S1, S2, S3) realicen conexiones directas a la base de datos MariaDB del monolito. Cada microservicio tendrá su propia base de datos optimizada (ej. Meilisearch para S1, PostgreSQL/MongoDB para S2, Redis/MinIO para S3).
* **Alternativa Descartada:** *Base de datos compartida (Shared Database)*. Se descartó porque acopla los servicios al esquema del monolito, impidiendo su evolución independiente, y genera un punto único de fallo y cuello de botella en el rendimiento.
* **Consecuencias:** 
  * (+) Autonomía total de los equipos y servicios; posibilidad de elegir la tecnología de datos adecuada para cada problema (PoliGlot Persistence).
  * (-) Los datos estarán temporalmente duplicados y se requerirá sincronización asíncrona (eventual consistency) para mantener la integridad.

---

## ADR-010: Consistencia Eventual e Idempotencia

* **Estatus:** Aceptado.
* **Contexto:** Al separar las bases de datos (ADR-009) y usar un broker asíncrono, la actualización del índice de búsqueda y el registro de auditoría ocurren milisegundos o segundos después de la acción en el monolito.
* **Decisión:** Adoptar un modelo de **Consistencia Eventual** para el sistema. Para manejar posibles reintentos de red y evitar datos duplicados, todos los consumidores deben ser **idempotentes**, basándose en un `event_id` único inyectado por el API Gateway o el `event-bridge`. Además, se implementará una **Dead Letter Queue (DLQ)** para enrutar los mensajes fallidos que no puedan procesarse tras varios intentos.
* **Alternativa Descartada:** *Transacciones Distribuidas (Two-Phase Commit / 2PC)*. Se descartó por ser extremadamente complejo de implementar, bloquear recursos durante la transacción y degradar severamente el rendimiento y la disponibilidad del sistema (violando la rapidez requerida).
* **Consecuencias:** 
  * (+) Alta disponibilidad y baja latencia en la escritura principal (el usuario recibe respuesta inmediata sin esperar al índice).
  * (+) Resiliencia ante fallos; los eventos fallidos no bloquean la cola principal.
  * (-) Complejidad adicional en la lógica de los consumidores para garantizar la idempotencia.

---

# 4.3. A3 – Contratos de API y de Eventos

## Política de Versionado
Para garantizar la retrocompatibilidad y no romper a los consumidores existentes (frontend u otros servicios):
* **API REST:** Todas las nuevas rutas extraídas del monolito se exponen bajo el prefijo `/v2/` (ej. `/v2/search`, `/v2/export`).
* **Eventos (Mensajería):** Todos los eventos asíncronos incluyen un campo `version` en su sobre (ej. `"version": "1.0"`).
* **Transición a v3:** Si en el futuro se requiere un cambio disruptivo (*breaking change*), se desplegará una nueva ruta `/v3/` y los eventos se publicarán con `"version": "2.0"`. El API Gateway enrutará el tráfico antiguo a `/v2/` y el nuevo a `/v3/` simultáneamente hasta que los clientes migren, logrando una degradación elegante.

---

## Contratos OpenAPI 3.1 (REST) y AsyncAPI 3.0 (Eventos)
Los esquemas completos con códigos de respuesta (200, 202, 400, 404, 503) y el diseño del sobre de eventos común (`event_id`, `version`, `occurred_at`, `request_id`, `payload`) se encuentran disponibles en el repositorio dentro del directorio `cna/contracts/`:

* **Contrato de Búsqueda:** `cna/contracts/search-svc.openapi.yaml`
* **Contrato de Exportación:** `cna/contracts/export-svc.openapi.yaml`
* **Bus de Eventos:** `cna/contracts/events.asyncapi.yaml`

---

## Política de Versionado (Punto A3)

Para las APIs síncronas (REST), utilizamos versionado en la URI (ej. `/v2/search`). Para los eventos asíncronos, utilizamos el campo `version` dentro del sobre del evento.

**¿Cómo agregar una versión v3 sin romper a los consumidores actuales?**
* **APIs REST:** Se creará un nuevo prefijo de ruta `/v3/` para los endpoints con cambios que rompan compatibilidad. Los endpoints `/v2/` seguirán funcionando en paralelo durante un periodo de transición, permitiendo a los clientes migrar a su propio ritmo.
* **Eventos Asíncronos:** Los nuevos eventos se publicarán con el campo `"version": "3"` en el sobre. Los consumidores existentes que esperan la versión "2" deben estar programados para ignorar versiones desconocidas, o el publicador puede emitir temporalmente ambas versiones en canales separados.

---

## Evidencia F5-E4: Validación de Contratos
Ejecución del linter para validar que los archivos YAML no tienen errores sintácticos o de esquema.

<img width="1024" height="484" alt="image" src="https://github.com/user-attachments/assets/dddb51d0-d74b-40f2-95fa-9d28925c7f01" />

---
# Bloque II – Fase 6: Construcción, Contenerización y Despliegue Distribuido
---

# 5.1. B1 - Infraestructura Multi-Nodo

## F6-E1: Estado de Docker Engine
### Nodo Principal (VM-1 - 192.168.1.25) - Nodo Secundario (VM-2 / nodo2 - 192.168.1.26)
<img width="1455" height="1106" alt="image" src="https://github.com/user-attachments/assets/aab606a7-1c9b-4829-be31-240cd2d6954e" />

## F6-E2: Reglas UFW en VM-2
### Cortafuegos en VM-2 (Regla restrictiva desde VM-1)
<img width="568" height="217" alt="image" src="https://github.com/user-attachments/assets/750ba32c-ec44-4b2f-a4df-67cb678853c5" />

## F6-E3: Certificado X.509 con SAN
Inspección del nuevo certificado X.509 donde se confirma la inclusión de ambas entradas DNS en el campo **Subject Alternative Name (SAN)** (`bookstack.local` y `api.bookstack.local`).
<img width="1138" height="78" alt="image" src="https://github.com/user-attachments/assets/527769b9-92d9-4f9b-a9ff-b29832551a3d" />
