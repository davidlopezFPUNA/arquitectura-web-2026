# Registros de Decisiones Arquitectónicas (ADR-006 a ADR-010)

Continuación de los registros arquitectónicos del TP I (`ADR-001` a `ADR-005`) para la evolución hacia Cloud Native Architecture (Fases 5, 6 y 7).

---

## ADR-006: Descomposición Modular del Monolito mediante Patrón Strangler Fig y API Gateway v2
- **Estado:** Aceptado
- **Contexto:** BookStack concentraba gestión documental, búsqueda, auditoría, exportación y procesamiento multimedia en un único proceso monolítico, dificultando el escalamiento independiente de operaciones de lectura/búsqueda.
- **Decisión:** Mantener BookStack como núcleo documental (`/`) y extraer los dominios de búsqueda (`search-svc`), auditoría (`audit-svc`), exportación (`export-svc`) y multimedia (`media-svc`) como microservicios autónomos en Go detrás de **Caddy API Gateway** bajo el prefijo versionado `/v2/*`.
- **Consecuencias:** Permite desplegar y escalar cada dominio de forma independiente manteniendo retrocompatibilidad total con rutas legadas.

## ADR-007: Mensajería Asíncrona con NATS JetStream y Puente de Eventos (`event-bridge`)
- **Estado:** Aceptado
- **Contexto:** La invocación síncrona HTTP desde el monolito hacia múltiples consumidores introducía acoplamiento temporal y riesgo de pérdida de eventos ante caídas parciales.
- **Decisión:** Implementar **NATS JetStream** como broker de mensajería persistente con streams durables, acknowledgements explícitos (`AckExplicit`), reintentos con backoff y cola de mensajes fallidos (**DLQ**), alimentado por un adaptador desacoplado (`event-bridge` en Node.js) que recibe webhooks autenticados desde BookStack.
- **Consecuencias:** Desacoplamiento temporal completo, absorción de picos de carga y cero pérdida de eventos ante indisponibilidad temporal de consumidores (`S1`/`S2`).

## ADR-008: Orquestación Multi-Nodo con Docker Swarm y Gestión Inmutable de Secretos
- **Estado:** Aceptado
- **Contexto:** Se requiere desplegar la plataforma en un clúster de dos nodos (`VM-1` Manager y `VM-2` Worker) con descubrimiento de servicios interno, escalamiento horizontal, actualizaciones sin caída y rotación segura de credenciales.
- **Decisión:** Utilizar **Docker Swarm** con red overlay cifrada, DNS interno (`tasks.<servicio>`), políticas de actualización `start-first` y **Docker Swarm Secrets** (`/run/secrets/*`) para gestionar tokens (`webhook_token`) y contraseñas.
- **Consecuencias:** Despliegue declarativo en un comando (`docker stack deploy`), auto-reparación ante caída de contenedores y rotación de secretos en caliente sin reconstruir imágenes.

## ADR-009: Observabilidad Integral basada en Pilares RED, Logs Correlacionados y Trazas OpenTelemetry
- **Estado:** Aceptado
- **Contexto:** En una arquitectura distribuida híbrida (síncrona/asíncrona), diagnosticar degradaciones o fallos parciales requiere correlacionar métricas, registros y trazas entre nodos.
- **Decisión:** Desplegar un stack unificado en `VM-1` compuesto por **Prometheus** + `nats-exporter` (métricas RED y profundidad de cola JetStream + reglas de alerta), **Loki + Promtail** (logs estructurados correlacionados por `X-Request-ID`), **Jaeger** (trazas distribuidas OTLP/W3C) y **Grafana** como consola central.
- **Consecuencias:** Detección proactiva de incidentes (`ServiceUnavailableOr5xxSpike`), trazabilidad de extremo a extremo y diagnóstico en segundos.

## ADR-010: Endurecimiento de Cadena de Suministro mediante SBOM CycloneDX y Auditoría Trivy
- **Estado:** Aceptado
- **Contexto:** El uso de imágenes base y librerías externas introduce riesgos de vulnerabilidades conocidas (`CVE`) que deben inventariarse y gobernarse antes de producción.
- **Decisión:** Incorporar **Syft** para generar manifiestos **SBOM** en formato estándar **CycloneDX JSON** versionados en el repositorio (`cna/security/sbom/`) y **Trivy** para auditar vulnerabilidades `HIGH`/`CRITICAL` (`cna/security/trivy/`), documentando mitigaciones de red y aceptaciones de riesgo en el registro de seguridad.
- **Consecuencias:** Trazabilidad completa de componentes de software y control auditado de vulnerabilidades sin bloquear entregas por hallazgos no explotables en runtime.
