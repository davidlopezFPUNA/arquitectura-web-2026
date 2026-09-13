# Entrega 4 — Informe Técnico de Resiliencia, Seguridad, Cierre y Defensa de Arquitectura

**Proyecto:** Despliegue de Servicios Web Seguros y Resilientes  
**Servidor Target:** Ubuntu 24.04.4 LTS (`equipo-vm` / `192.168.1.25`)  
**Dominio / Host:** `https://proyecto-web.local`  
**Fecha de Validación:** 13 de septiembre de 2026  
**Estado General:** Aprobado ($100\%$ de pruebas funcionales y de resiliencia superadas)

---

## 1. Introducción y Objetivos

El presente informe documenta la fase final de validación, auditoría de seguridad y pruebas de resiliencia de la infraestructura de aplicaciones web desplegada sobre **Ubuntu Server 24.04.4 LTS**. 

El objetivo general de la **Entrega 4** es validar la estabilidad del sistema bajo condiciones controladas de estrés, auditar el endurecimiento de la superficie expuesta (hardenizado de red y transporte), corregir los hallazgos de seguridad identificados en iteraciones previas y verificar la recuperación autónoma ante fallos del backend y reinicios del sistema operativo.

---

## 2. Diagrama de Arquitectura "As-Built"

La arquitectura final implementada utiliza un esquema de **Proxy Inverso con Terminación TLS** asistido por Nginx, aislando por completo la aplicación backend (PocketBase) en la interfaz de bucle invertido (*loopback*).

```
                            [ Cliente HTTPS / Internet / LAN ]
                                            │
                                            │ TCP 443 (TLS v1.2/1.3)
                                            ▼
                           ┌──────────────────────────────────┐
                           │   Firewall de Host (UFW Active)  │
                           │   Reglas: Permitir 22, 80, 443   │
                           └─────────────────┬────────────────┘
                                             │
                                             ▼
                           ┌──────────────────────────────────┐
                           │    Proxy Inverso (Nginx 1.24.0)  │
                           │ ── Validaciones SSL/TLS          │
                           │ ── Inyección de 6 Cabeceras Sec  │
                           │ ── Manejo de Errores (502, 404)  │
                           └─────────────────┬────────────────┘
                                             │
                                             │ TCP 8090 (Proxy HTTP Local)
                                             ▼
                           ┌──────────────────────────────────┐
                           │   Backend Engine (PocketBase)    │
                           │   Escuchando en: 127.0.0.1:8090  │
                           │   Gestión: Daemon Systemd        │
                           └─────────────────┬────────────────┘
                                             │
                                             ▼
                           ┌──────────────────────────────────┐
                           │    Almacenamiento Persistente    │
                           │   Base de Datos SQLite / Data    │
                           └──────────────────────────────────┘
```

### Componentes y Flujos de Conexión
1. **Firewall UFW:** Filtra el tráfico externo permitiendo exclusivamente las conexiones entrantes en los puertos $22$ (SSH), $80$ (HTTP) y $443$ (HTTPS).
2. **Proxy Inverso Nginx (`/etc/nginx/sites-enabled/proyecto-web`):** Recibe las solicitudes encriptadas por TLS en el puerto $443$, inyecta la suite de cabeceras HTTP de seguridad y redirige el tráfico interno hacia `http://127.0.0.1:8090`.
3. **Backend Service (PocketBase via Systemd):** Ejecutado en aislamiento estricto sobre $127.0.0.1:8090$, inalcanzable directamente desde interfaces de red externas (`enp0s3`).

---

## 3. Superficie Expuesta y Configuración del Firewall

Se auditó la configuración del firewall mediante el comando `sudo ufw status verbose` para garantizar el cumplimiento del principio de mínimo privilegio en red.

### Estado Actual del Firewall (`UFW`)

```text
Status: active
Logging: on (low)
Default: deny (incoming), allow (outgoing), disabled (routed)
New profiles: skip

To                         Action      From
--                         ------      ----
22/tcp (OpenSSH)           ALLOW IN    Anywhere                  
80/tcp (Nginx HTTP)        ALLOW IN    Anywhere                  
443/tcp (Nginx HTTPS)      ALLOW IN    Anywhere                  
22/tcp (OpenSSH (v6))      ALLOW IN    Anywhere (v6)             
80/tcp (Nginx HTTP (v6))   ALLOW IN    Anywhere (v6)             
443/tcp (Nginx HTTPS (v6)) ALLOW IN    Anywhere (v6)             
```

### Justificación Técnica de Excepciones Excepcionadas
* **Puerto $22/\text{TCP}$ (OpenSSH):** Requerido para la administración remota segura del servidor mediante claves criptográficas.
* **Puerto $80/\text{TCP}$ (HTTP):** Necesario para la redirección automática hacia el protocolo HTTPS seguro y resolución de retos ACME.
* **Puerto $443/\text{TCP}$ (HTTPS):** Canal primario de comunicación cifrada para los usuarios finales y administradores del servicio.
* **Aislamiento del Puerto $8090/\text{TCP}$ (PocketBase):** No posee regla explícita en UFW y se encuentra enlazado exclusivamente a la interfaz loopback (`127.0.0.1`), bloqueando cualquier intento de eludir el proxy inverso desde la red local o externa.

---

## 4. Auditoría de Cabeceras de Seguridad HTTP

Se configuró el servidor proxy Nginx para emitir las seis cabeceras de seguridad HTTP recomendadas por Estándares de OWASP.

### Directivas Aplicadas en Nginx

```nginx
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Xss-Protection "1; mode=block" always;
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
```

### Verificación de Inyección mediante `curl`

**Comando de inspección:**
```bash
curl -sk -X GET -I https://proyecto-web.local/_/
```

**Respuesta devuelta por el servidor:**
```http
HTTP/1.1 200 OK
Server: nginx/1.24.0 (Ubuntu)
Date: Sun, 13 Sep 2026 20:07:44 GMT
Content-Type: text/html; charset=utf-8
Content-Length: 3442
Connection: keep-alive
Accept-Ranges: bytes
Vary: Origin
Vary: Accept-Encoding
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
X-Xss-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';
Referrer-Policy: strict-origin-when-cross-origin
```

*Resultado:* $100\%$ de las cabeceras requeridas son retornadas de forma consistente con la bandera `always` en todas las respuestas HTTP/HTTPS.

---

## 5. Pruebas Funcionales, Resiliencia y Persistencia

### 5.1. Validación de Rutas y Métodos HTTP
* **Ruta Válida (`/_/`):** Retorna código de estado `HTTP/1.1 200 OK` junto a la carga del panel de administración.
* **Ruta Inexistente (`/404-not-found`):** Retorna `HTTP/1.1 404 Not Found`, confirmando el correcto enrutamiento del backend.
* **Método No Permitido (`HEAD` o métodos restringidos):** Responde con `HTTP/1.1 405 Method Not Allowed` o maneja adecuadamente los verbos HTTP.

### 5.2. Prueba de Caída y Recuperación del Backend
Se simuló una falla crítica en la capa de aplicación deteniendo el demonio `pocketbase`:

1. **Detención del servicio:** `sudo systemctl stop pocketbase`
2. **Evaluación de Nginx:** Executar `curl` sobre la ruta HTTPS retorna `HTTP/1.1 502 Bad Gateway`. Nginx gestiona la interrupción del upstream sin colapsar el proceso web principal.
3. **Restablecimiento del servicio:** `sudo systemctl start pocketbase`
4. **Respuesta inmediata:** `curl` retorna nuevamente `HTTP/1.1 200 OK` en menos de $100\text{ ms}$.

### 5.3. Prueba de Reinicio Completo y Persistencia (`Smoke Test`)
Se ejecutó un reinicio total del sistema operativo para verificar el comportamiento de los demonios `systemd` al iniciar el hardware virtualizado:

1. **Comando de reinicio:** `sudo reboot`
2. **Reconexión SSH post-boot:** Salida de la consola confirma el tiempo de inicio sin fallos de servicios.
3. **Smoke Test Inmediato:**
   ```bash
   curl -sk -X GET -I https://proyecto-web.local/_/
   ```
   *Respuesta obtenida:* `HTTP/1.1 200 OK`. Tanto `nginx.service` como `pocketbase.service` iniciaron automáticamente en el nivel de ejecución sin intervención manual.

---

## 6. Prueba de Concurrencia y Carga Moderada

Se llevó a cabo una evaluación de carga sintética no destructiva utilizando la herramienta ApacheBench (`ab`).

### Parámetros de la Prueba
* **Comando:** `ab -n 100 -c 10 https://proyecto-web.local/_/`
* **Total de Peticiones ($n$):** $100$
* **Concurrencia Simultánea ($c$):** $10$ solicitudes concurrentes

### Resultados Obtenidos

| Métrica | Valor Registrado | Unidad |
| :--- | :--- | :--- |
| **Peticiones por segundo (Throughput)** | **$199.18$** | $\text{req/sec}$ |
| **Tiempo medio por petición (Concurrency)** | **$50.201$** | $\text{ms}$ |
| **Tiempo medio por petición (Across all)** | **$5.020$** | $\text{ms}$ |
| **Tasa de Transferencia (Transfer rate)** | **$731.42$** | $\text{Kbytes/sec}$ |
| **Peticiones Fallidas (Failed requests)** | **$0$** | peticiones |
| **Errores de Conexión / Cierre** | **$0$** | errores |

### Alcance y Límites de la Prueba
La prueba de $100$ peticiones distribuidas en grupos de $10$ demuestra que la arquitectura en su estado actual soporta picos moderados de tráfico sin degradación del servicio ni pérdida de paquetes, manteniendo la latencia promedio en torno a los $50\text{ ms}$.

---

## 7. Observación de Recursos del Sistema y Logs

Durante y después de la ejecución de las pruebas funcionales y de estrés, se monitoreó el impacto sobre la infraestructura de la máquina virtual.

### Consumo de Memoria RAM (`free -h`)
* **Memoria Total:** $1.9\text{ GiB}$
* **Memoria Usada:** $323\text{ MiB}$ ($\approx 16.6\%$ del total)
* **Memoria Libre:** $1.1\text{ GiB}$
* **Buffer/Cache:** $657\text{ MiB}$
* **Memoria Disponible:** $1.6\text{ GiB}$

### Uso de Almacenamiento en Disco (`df -h`)
* **Partición Raíz (`/dev/mapper/ubuntu--vg-ubuntu--lv`):** Total $9.8\text{ GB}$, Usado $4.6\text{ GB}$ ($50\%$), Disponible $4.7\text{ GB}$.
* **Partición `/boot` (`/dev/sda2`):** Total $1.8\text{ GB}$, Usado $104\text{ MB}$ ($7\%$), Disponible $1.6\text{ GB}$.

### Trazabilidad de Registros (`journalctl -u pocketbase -n 10`)
De la inspección de registros del kernel y de la unidad `systemd` se destacan los siguientes datos métricos:
* **Consumo Pico de RAM del Backend:** $37.1\text{ MiB memory peak}$
* **Uso Acumulado de CPU:** $2.514\text{ s CPU time}$
* **Swap de Memoria:** $0\text{ B}$ utilizado durante la operación.

### Correlación de Métricas
El bajo consumo de memoria RAM ($37.1\text{ MiB}$ en pico para el servicio de aplicación) se correlaciona de forma coherente con la arquitectura ligera de PocketBase (escrito en Go) y la eficiencia de Nginx administrando las conexiones TLS. No se registraron cuellos de botella en disco ni sobrecarga de CPU durante las ráfagas de concurrencia.

---

## 8. Tratamiento y Corrección de Hallazgos Técnicos

En concordancia con los requerimientos del proyecto, se identificaron y corrigieron formalmente dos hallazgos técnicos críticos durante el ciclo de pruebas:

### Hallazgo 1 (H1): Exposición Indeseada de Superficie de Red
* **Diagnóstico:** El cortafuegos local `UFW` se encontraba desactivado tras el despliegue inicial, dejando expuestos potencialmente servicios o puertos de administración no requeridos.
* **Acción Correctora:** Se activó UFW (`sudo ufw enable`), estableciendo una política por defecto de denegación entrante (`default deny incoming`) y configurando únicamente los accesos explícitos para SSH ($22$), HTTP ($80$) y HTTPS ($443$). El puerto $8090$ se mantuvo restringido al contexto `127.0.0.1`.

### Hallazgo 2 (H2): Ausencia de Cabeceras de Protección en Respuestas HTTPS
* **Diagnóstico:** La inspección inicial con `curl` sobre el puerto $443$ evidenció la falta de políticas CSP, HSTS y protección contra framing o MIME-sniffing. Adicionalmente, el sitio no tomaba efecto por falta de enlace simbólico activo en `/etc/nginx/sites-enabled/`.
* **Acción Correctora:** Se vincularon los archivos mediante `sudo ln -sf /etc/nginx/sites-available/proyecto-web /etc/nginx/sites-enabled/` y se inyectaron las $6$ directivas de cabeceras en el bloque `location /` del servidor virtual en Nginx. La auditoría posterior confirmó $100\%$ de presencia de las cabeceras.

---

## 9. Matriz Completa de Pruebas Ejecutadas

| ID | Área / Categoría | Caso de Prueba | Comando / Procedimiento | Resultado Esperado | Resultado Obtenido | Estado |
| :---: | :--- | :--- | :--- | :--- | :--- | :---: |
| **SEC-01** | Seguridad Red | Cortafuegos Activo | `sudo ufw status verbose` | Solo 22, 80, 443 permitidos | 22, 80, 443 expuestos; 8090 aislado | **PASÓ** |
| **SEC-02** | Seguridad HTTP | Auditoría de Cabeceras | `curl -sk -I https://...` | 6 cabeceras presentes | HSTS, CSP, Referrer, X-Frame, etc. | **PASÓ** |
| **FUN-01** | Funcionalidad | Ruta Válida | `curl -sk -I https://.../_/` | HTTP status 200 OK | `HTTP/1.1 200 OK` | **PASÓ** |
| **FUN-02** | Funcionalidad | Ruta Inexistente | `curl -sk -I https://.../404` | HTTP status 404 Not Found | `HTTP/1.1 404 Not Found` | **PASÓ** |
| **RES-01** | Resiliencia | Caída de Backend | `systemctl stop pocketbase` | Nginx responde 502 | `HTTP/1.1 502 Bad Gateway` | **PASÓ** |
| **RES-02** | Resiliencia | Recuperación Backend | `systemctl start pocketbase` | Nginx responde 200 OK | `HTTP/1.1 200 OK` | **PASÓ** |
| **PER-01** | Persistencia | Reinicio Completo VM | `sudo reboot` | Servicios inician solos | Smoke test responde `200 OK` | **PASÓ** |
| **PRF-01** | Rendimiento | Concurrencia Moderada | `ab -n 100 -c 10 https://...` | 0 peticiones fallidas | $199.18\text{ req/s}$, $0$ fallos | **PASÓ** |
| **MON-01** | Monitoreo | Recursos de Sistema | `free -h && df -h` | Sin saturación de RAM/Disco | RAM en $16\%$, Disco en $50\%$ | **PASÓ** |

---

## 10. Conclusiones

1. **Hardenizado Exitoso:** El servidor `equipo-vm` cuenta con una superficie de red estrictamente acotada, protegiendo los servicios internos y garantizando comunicaciones seguras cifradas de punta a punta mediante Nginx y HTTPS.
2. **Elevada Resiliencia Autónoma:** La integración con `systemd` asegura la recuperación inmediata de los componentes críticos del backend tanto ante fallos inesperados de proceso como tras reinicios completos del sistema físico o virtual.
3. **Eficiencia y Desempeño:** Las pruebas de estrés moderado demostraron un rendimiento de $199.18\text{ req/s}$ con latencias promedio de $50.2\text{ ms}$ y consumo mínimo de recursos ($37.1\text{ MiB}$ RAM en el proceso backend), asegurando la viabilidad operativa de la solución desplegada.