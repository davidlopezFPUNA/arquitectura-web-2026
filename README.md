# arquitectura-web-2026
Proyecto Integrador - Arquitectura Web 2026: Despliegue de PocketBase en Ubuntu Server 24.04 LTS con Nginx Reverse Proxy y systemd.

# Proyecto Integrador — Arquitectura Web 2026

**Estudiantes:** David López, Junior Legal

**Institución:** Facultad Politécnica - Universidad Nacional de Asunción (FPUNA)  
**Sistema Operativo:** Ubuntu Server 24.04.4 LTS (x86_64)  
**Infraestructura:** VirtualBox (2 vCPU, 2 GB RAM, 20 GB Disk)  

---
## Entrega 1. Descubrimiento, selección y linea base

## 1. Ficha Técnica de la Aplicación

| Criterio | Detalle |
| :--- | :--- |
| **Nombre** | PocketBase |
| **Repositorio Oficial** | [github.com/pocketbase/pocketbase](https://github.com/pocketbase/pocketbase) |
| **Versión Seleccionada** | `v0.22.21` |
| **Licencia** | MIT License |
| **Lenguaje / Framework** | Go (Golang) |
| **Servidor de Aplicación** | HTTP Server embebido en Go |
| **Base de Datos** | SQLite (embebida en archivo local) |
| **Puertos Previstos** | `8090` (PocketBase Backend) / `80` (Nginx Reverse Proxy) |

---

## 2. Matriz de Direccionamiento y Línea Base de Red

| Parámetro | Valor |
| :--- | :--- |
| **Nombre de Host** | `equipo-vm` |
| **Interfaz de Red** | `enp0s3` |
| **Modo de Red VirtualBox** | Adaptador Puente (*Bridged*) |
| **Dirección IPv4** | `192.168.1.25` |
| **Máscara / Prefijo** | `255.255.255.224` (`/27`) |
| **Puerta de Enlace (Gateway)** | `192.168.1.1` |
| **Servidor DNS** | `192.168.1.1` |
| **Puertos Abiertos Iniciales** | `22/tcp` (OpenSSH) |

---

## 3. Diagrama de Arquitectura Lógica Inicial

```text
 [ Cliente / Host Windows ]
             │
             │ (Petición HTTP - Puerto 80)
             ▼
 [ Adaptador Puente / Red 192.168.1.0/27 ]
             │
             ▼
 [ Servidor Web: Nginx Proxy (Puerto 80) ]
             │
             │ (Redirección local 127.0.0.1:8090)
             ▼
 [ Aplicación: PocketBase Service (Puerto 8090) ]
             │
             ▼
 [ Base de Datos: SQLite (Archivo embebido en disco) ]
```
---

## 4. Registro de Decisiones de Arquitectura (ADR-001)

* **Título:** Selección de PocketBase, Nginx como Proxy Inverso y Red en Modo Puente.
* **Estado:** Aceptado.
* **Autores:** David López, Junior Legal.
* **Contexto:** Se requiere desplegar una aplicación observable sobre Ubuntu Server 24.04 LTS optimizando los recursos asignados en la máquina virtual (2 vCPU, 2 GB RAM). Para ello se evaluaron dos candidatos:
  * **PocketBase v0.22.21 (Seleccionado):** Licencia MIT, binario único en Go con SQLite embebida.
  * **Strapi v4 (Descartado):** Licencia MIT, Node.js + DB externa. Descartado por requerir dependencias complejas y mayor consumo de memoria RAM.
* **Decisión:**
  * i. Se elige **PocketBase** como backend debido a su arquitectura de binario único embebido con SQLite, eliminando dependencias externas pesadas y acelerando el tiempo de despliegue.
  * ii. Se establece el modo de red **Adaptador Puente (Bridged)** para integrar la VM directamente a la subred física (`192.168.1.0/27`) y habilitar administración remota vía SSH.
  * iii. Se define **Nginx** como proxy inverso para exponer la aplicación en el puerto HTTP estándar `80`.

---

## 5. Evidencias de Comandos Ejecutados (Línea Base)

```bash
# Identificación del sistema y kernel
$ hostnamectl
Static hostname: equipo-vm
Operating System: Ubuntu 24.04.4 LTS
Kernel: Linux 6.8.0-139-generic
Architecture: x86-64

# Recursos de Hardware (CPU y RAM)
$ lscpu | grep -E "Model name|CPU\(s\):"
CPU(s):         2
Model name:     AMD FX(tm)-4300 Quad-Core Processor

$ free -h
Mem Total: 1.9Gi | Used: 316Mi | Free: 1.5Gi

# Red, Enrutamiento y DNS
$ ip a
inet 192.168.1.25/27 brd 192.168.1.31 scope global dynamic enp0s3

$ ip route
default via 192.168.1.1 dev enp0s3 proto static
192.168.1.0/27 dev enp0s3 proto kernel scope link src 192.168.1.25

$ resolvectl status
Link 2 (enp0s3): Current DNS Server: 192.168.1.1

# Puertos Escuchando en el Sistema (SSH, Nginx, PocketBase)
$ sudo ss -lntup
Netid  State   Recv-Q  Send-Q   Local Address:Port   Process
tcp    LISTEN  0       511            0.0.0.0:80     ("nginx")
tcp    LISTEN  0       4096         127.0.0.1:8090   ("pocketbase")
tcp    LISTEN  0       128            0.0.0.0:22     ("sshd")

# Verificación del servicio Web y Versión de Git
$ curl -I [http://127.0.0.1](http://127.0.0.1)
HTTP/1.1 405 Method Not Allowed
Server: nginx/1.24.0 (Ubuntu)

$ git rev-parse HEAD
6997003
```
---

## 6. Evidencia de Funcionamiento

<img width="1524" height="766" alt="Captura de pantalla 2026-09-12 185903 png" src="https://github.com/user-attachments/assets/b93043b4-bb3a-49b8-83a4-0c21bfa189ce" />

---
## Entrega 2 — Instalación y Publicación HTTP

### 1. Dominio Local y Proxy Inverso
Se configuró la resolución por nombre en el archivo `hosts` del equipo anfitrión y el proxy inverso en Nginx para publicar el servicio en el puerto HTTP 80.

* **Dominio local mapeado:** `proyecto-web.local` → `192.168.1.25`
* **Acceso HTTP público (Nginx):** `http://proyecto-web.local/` (Puerto 80)
* **Binding interno (PocketBase):** `http://127.0.0.1:8090` (Aislado de la red pública)

---

### 2. Comandos de Diagnóstico y Evidencias HTTP

```text
# Validación de sintaxis en Nginx
$ sudo nginx -t
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful

# Estado del servicio administrado por systemd
$ sudo systemctl status pocketbase
● pocketbase.service - PocketBase service
     Loaded: loaded (/etc/systemd/system/pocketbase.service; enabled)
     Active: active (running)

# Pruebas de cabeceras HTTP desde cliente
$ curl.exe -i [http://proyecto-web.local/_/](http://proyecto-web.local/_/)
HTTP/1.1 200 OK
Server: nginx/1.24.0 (Ubuntu)
Content-Type: text/html; charset=utf-8
X-Frame-Options: SAMEORIGIN
X-Content-Type-Options: nosniff

```
---
### 3. Prueba de Persistencia de Datos

Se validó la capacidad de almacenamiento no volátil del motor SQLite registrando datos en el Admin UI y verificando su integridad tras reiniciar el daemon del servicio:

* **Colección:** `contactos`
* **Campo creado:** `nombre` (Plain text)
* **Registro de prueba:** `David López` (ID: `jjgykjsh7a0tf0d`)
* **Ubicación del archivo de base de datos:** `/home/dlopez/pocketbase/pb_data/data.db`

```bash
# Reinicio del servicio para comprobar no volatilidad
sudo systemctl restart pocketbase
```

<img width="1533" height="766" alt="Captura de pantalla 2026-09-12 214951" src="https://github.com/user-attachments/assets/1143c519-5e58-46e7-ac48-1f8e29b47f9d" />

# Entrega 3: DNS local, TLS, HTTP y medición

Este documento consolida la configuración del dominio local, la caracterización del certificado X.509, el análisis de rendimiento, las estrategias de almacenamiento en caché, la comparativa de arquitectura backend/proxy y la captura de tráfico de red.

---

## 1. Identificación y Resolución DNS Local

Se diferencian los cuatro elementos fundamentales de la arquitectura de red:

* **Nombre de Dominio:** `proyecto-web.local` (Identificador alfanumérico legible resuelto localmente vía `/etc/hosts`).
* **Dirección IP:** `127.0.0.1` / `192.168.1.25` (Dirección lógica de Capa 3 para el enrutamiento del paquete).
* **Puertos de Red:** `80` (HTTP - Redirección), `443` (HTTPS - Nginx Proxy), `8090` (HTTP - Backend PocketBase).
* **URL Completa:** `https://proyecto-web.local/_/` (Localizador global que especifica esquema, host, puerto implícito 443 y ruta).

---

## 2. Caracterización del Certificado Digital (X.509)

El certificado autofirmado alojado en `/etc/ssl/proyecto-web/proyecto-web.crt` presenta los siguientes parámetros técnicos:

| Parámetro | Valor Registrado |
| :--- | :--- |
| **Sujeto (Subject)** | `C = PY, ST = Central, L = Asuncion, O = ArquitecturaWeb, CN = proyecto-web.local` |
| **Emisor (Issuer)** | `C = PY, ST = Central, L = Asuncion, O = ArquitecturaWeb, CN = proyecto-web.local` |
| **Cadena de Confianza** | Un solo nivel (Certificado Raíz Autofirmado = Certificado Hoja) |
| **Período de Validez** | `Sep 13 01:09:23 2026 GMT` a `Sep 13 01:09:23 2027 GMT` |
| **Nombres Alternativos (SAN)** | `DNS:proyecto-web.local`, `IP Address:192.168.1.25` |
| **Algoritmo de Firma y Clave** | RSA 2048 bits / `sha256WithRSAEncryption` |
| **Huella Digital (SHA-256)** | `4C:4B:72:9B:C2:2B:F7:D0:B6:AB:B0:BF:4E:2D:6F:50:93:CB:B7:00:46:D3:6D:76:DD:DD:F2:4E:E1:43:3C:32` |

> **Evaluación de Confianza del Navegador:** El cliente emite una advertencia de seguridad (*No seguro*) debido a que la Entidad Emisora (Issuer) es autofirmada y no reside en el almacén de autoridades de certificación raíz de confianza (Trust Store) del sistema operativo del cliente. A nivel de cifrado de canal, TLS garantiza confidencialidad e integridad.

---

## 3. Redirección HTTP a HTTPS y Negociación TLS

* **Redirección:** `curl -I http://proyecto-web.local` confirma la respuesta `HTTP/1.1 301 Moved Permanently` con la cabecera `Location: https://proyecto-web.local/`.
* **Protocolo Negociado:** `TLSv1.3`
* **Cipher Suite:** `TLS_AES_256_GCM_SHA384`
* **Intercambio de Claves:** `X25519` / `RSASSA-PSS`

---

## 4. Análisis de Rendimiento (Métricas `curl -w`)

Resumen estadístico derivado de 10 ejecuciones secuenciales hacia `https://proyecto-web.local/_/`:

| Fase de Conexión | Tiempos Mínimos | Promedio (10 it.) | Tiempos Máximos |
| :--- | :--- | :--- | :--- |
| **Resolución DNS (`time_namelookup`)** | 1.16 ms | **1.88 ms** | 3.25 ms |
| **Conexión TCP (`time_connect`)** | 1.45 ms | **2.30 ms** | 3.55 ms |
| **Apretón TLS (`time_appconnect`)** | 10.51 ms | **13.95 ms** | 18.45 ms |
| **Primer Byte / TTFB (`time_starttransfer`)** | 13.43 ms | **18.46 ms** | 23.39 ms |
| **Tiempo Total (`time_total`)** | 13.50 ms | **18.53 ms** | 23.46 ms |

**Interpretación:** La fase TLS (`13.95 ms`) representa la mayor fracción del tiempo de conexión inicial por la negociación asimétrica Handshake.

---

## 5. Análisis de Caché, Compresión y Respuestas HTTP 304

* **Estado Actual:** PocketBase entrega los assets embebidos sin cabeceras explícitas de caché persistente (`Cache-Control` o `ETag`), devolviendo código `200 OK` directo.
* **Propuesta de Optimización en Nginx:** Para habilitar validación condicional (`304 Not Modified`) y reducir transferencia de red, se sugiere incorporar en el bloque de Nginx:

```nginx
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
    proxy_pass [http://127.0.0.1:8090](http://127.0.0.1:8090);
    expires 1d;
    add_header Cache-Control "public, no-transform";
    gzip on;
    gzip_types text/plain text/css application/javascript image/svg+xml;
}
```
---
## 6. Comparativa: Acceso Directo Backend vs. Reverse Proxy

| Criterio | Acceso Directo (`http://127.0.0.1:8090`) | Acceso Mediante Proxy (`https://proyecto-web.local`) |
| :--- | :--- | :--- |
| **Seguridad de Transporte** | Tráfico HTTP en texto claro (sin cifrado) | Cifrado SSL/TLS 1.3 de extremo a extremo |
| **Aislamiento y Exposición** | Expone el puerto del servicio Go directamente a la red | Oculta la infraestructura interna tras Capa 7 (Nginx) |
| **Cabeceras de Seguridad** | Limitadas por la configuración por defecto del backend | Inyección centralizada de políticas (`X-Frame-Options`, `nosniff`) |
| **Rendimiento y Latencia** | Menor latencia procesada por omitir la capa SSL | Ligera sobrecarga inicial (~14 ms) por handshake TLS |

---

## 7. Captura e Interpretación de Tráfico (`tcpdump`)

Análisis del flujo de tramas registradas en el archivo `captura_https.pcap` sobre la interfaz de bucle de retorno (`lo`):

* **Resolución de Red (DNS / ARP):** Al operar sobre la interfaz local loopback (`127.0.0.1`), las solicitudes no requirieron tramas ARP en la interfaz física ni consultas DNS externas; el mapeo de `proyecto-web.local` fue resuelto a nivel de sistema por la pila de red vía `/etc/hosts`.
* **Establecimiento de Sesión TCP (3-Way Handshake):**
  * `Cliente -> Servidor [SYN]` (`seq 3123783644`, puerto efímero de origen `46382` hacia `443`).
  * `Servidor -> Cliente [SYN, ACK]` (`seq 3670356629`, `ack 3123783645`).
  * `Cliente -> Servidor [ACK]` (Canal de transporte Capa 4 establecido correctamente).
* **Tráfico Cifrado TLS 1.3:**
  * `Client Hello` (`Flags [P.]`, 517 bytes): El cliente envía los parámetros de cifrado soportados y la extensión SNI con el nombre `proyecto-web.local`.
  * `Server Hello` (`Flags [P.]`, 1563 bytes): Nginx responde entregando el certificado X.509 autofirmado y negociando el cipher suite `TLS_AES_256_GCM_SHA384`.
  * `Application Data` (`length 3791 B`): Transferencia de la carga útil HTTP cifrada sin posibilidad de lectura en texto claro en la traza.
* **Cierre de Conexión:**
  * `Cliente -> Servidor [FIN, ACK]` (`Flags [F.]`): Finalización limpia y ordenada de la sesión TCP.

---

# Entrega 4 — Resiliencia, Seguridad, Cierre y Defensa

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

---

## Entrega 5 — Matriz Mínima de Pruebas Globales (P01 — P12)

Esta sección consolida los 12 puntos de evaluación requeridos para la validación global de la arquitectura, unificando los hallazgos y evidencias obtenidas a lo largo del proyecto integrador.

| ID | Dimensión | Prueba y Evidencia Registrada | Resultado de Aceptación | Estado |
| :---: | :--- | :--- | :--- | :---: |
| **P01** | **Direccionamiento** | `ip a` / `ip route`: IP `192.168.1.25/27`, Gateway `192.168.1.1`, Interfaz `enp0s3` (Adaptador Puente). | Direccionamiento coherente y alineado a la subred del anfitrión. | **Aprobado** |
| **P02** | **Conectividad** | Ping y conexiones TCP a puertos 22, 80 y 443 desde Host Windows. Traza ARP comprobada. | Alcance demostrado desde el cliente anfitrión sin bloqueos. | **Aprobado** |
| **P03** | **Puertos** | `sudo ss -lntup`: SSH (`:22`), Nginx (`:80`, `:443`), PocketBase (`127.0.0.1:8090`). | Solo servicios justificados escuchando; backend aislado localmente. | **Aprobado** |
| **P04** | **Nombre** | Mapeo en `hosts` de `proyecto-web.local` $\rightarrow$ `192.168.1.25`. Resolución DNS probada. | El nombre resuelve a la VM de forma repetible en la LAN. | **Aprobado** |
| **P05** | **HTTP** | Solicitudes `curl -i`: Métodos GET, HEAD, estados `200 OK`, `404 Not Found`, `405 Method Not Allowed`. | Respuestas válidas con semántica HTTP explicada. | **Aprobado** |
| **P06** | **Proxy Inverso** | Flujo `Cliente -> Nginx (443) -> PocketBase (127.0.0.1:8090)`. Registros en `journalctl`. | Proxy funcional; backend protegido sin exposición pública directa. | **Aprobado** |
| **P07** | **TLS** | Certificado X.509 autofirmado (RSA 2048, SAN: `proyecto-web.local`, TLS 1.3). | HTTPS funcional; causa del aviso de confianza en navegador justificada. | **Aprobado** |
| **P08** | **Rendimiento** | Métricas `curl -w` (10 it.): DNS 1.88 ms, TCP 2.30 ms, TLS 13.95 ms, TTFB 18.46 ms, Total 18.53 ms. | Tabla estadística detallada e interpretación prudente del canal TLS. | **Aprobado** |
| **P09** | **Caché** | Análisis de assets en PocketBase (`200 OK`); diseño de reglas `Cache-Control` / `304` en Nginx. | Comportamiento del backend demostrado y optimización documentada. | **Aprobado** |
| **P10** | **Seguridad** | UFW activo (22, 80, 443); 6 cabeceras de seguridad inyectadas (`nosniff`, CSP, HSTS, etc.). | Hallazgos corregidos y superficie acotada al mínimo privilegio. | **Aprobado** |
| **P11** | **Resiliencia** | `systemctl stop/start` (502 Bad Gateway $\rightarrow$ 200 OK); `sudo reboot` con persistencia `systemd`. | Falla observable, manejo controlado y recuperación autónoma post-boot. | **Aprobado** |
| **P12** | **Recursos** | Benchmarking `ab -n 100 -c 10` ($199.18\text{ req/s}$); RAM al $16.6\%$ (Pico PB: $37.1\text{ MiB}$), Disco al $50\%$. | Mediciones de CPU/RAM/Disco correlacionadas con la prueba de carga. | **Aprobado** |

