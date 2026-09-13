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

# Entregable 3: Habilitación de HTTPS, Análisis TLS y Rendimiento

Este documento recopila la configuración, la caracterización técnica del certificado X.509, las pruebas de redirección HTTP/HTTPS, las métricas de latencia y el análisis de tráfico de red para la infraestructura del dominio `proyecto-web.local`.

---

## 1. Caracterización del Certificado Digital (X.509)

El certificado autofirmado generado con OpenSSL y desplegado en `/etc/ssl/proyecto-web/proyecto-web.crt` presenta las siguientes especificaciones técnicas:

| Parámetro | Valor Registrado |
| :--- | :--- |
| **Sujeto (Subject)** | `C = PY, ST = Central, L = Asuncion, O = ArquitecturaWeb, CN = proyecto-web.local` |
| **Emisor (Issuer)** | `C = PY, ST = Central, L = Asuncion, O = ArquitecturaWeb, CN = proyecto-web.local` |
| **Período de Validez** | `Sep 13 01:09:23 2026 GMT` a `Sep 13 01:09:23 2027 GMT` |
| **Nombres Alternativos (SAN)** | `DNS:proyecto-web.local`, `IP Address:192.168.1.25` |
| **Algoritmo de Firma y Clave** | RSA 2048 bits / `sha256WithRSAEncryption` |
| **Huella Digital (SHA-256)** | `4C:4B:72:9B:C2:2B:F7:D0:B6:AB:B0:BF:4E:2D:6F:50:93:CB:B7:00:46:D3:6D:76:DD:DD:F2:4E:E1:43:3C:32` |

---

## 2. Verificación de Redirección y Protocolo HTTPS

### Redirección HTTP a HTTPS
La consulta al puerto 80 mediante `curl -I http://proyecto-web.local` ratifica la redirección forzada a nivel de servidor Web:

* **Código de Estado:** `HTTP/1.1 301 Moved Permanently`
* **Cabecera de Ubicación:** `Location: https://proyecto-web.local/`

### Negociación de Canal Cifrado
* **Versión del Protocolo:** `TLSv1.3`
* **Algoritmo de Cifrado (Cipher Suite):** `TLS_AES_256_GCM_SHA384`
* **Intercambio de Claves:** `X25519` / `RSASSA-PSS`

> **Justificación Técnica de Advertencia en Navegador:** El navegador señala la conexión como *No segura* debido a que el certificado es autofirmado y su Entidad Emisora (CA) no está agregada en el almacén de certificados de confianza del sistema operativo cliente. El cifrado del canal TLS funciona adecuadamente a nivel de transporte.

---

## 3. Análisis de Rendimiento (curl -w)

Resumen estadístico derivado de las 10 ejecuciones secuenciales sobre el recurso `https://proyecto-web.local/_/`:

| Fase de Conexión | Tiempos Mínimos | Promedio (10 it.) | Tiempos Máximos |
| :--- | :--- | :--- | :--- |
| **Resolución DNS (`time_namelookup`)** | 1.16 ms | **1.88 ms** | 3.25 ms |
| **Conexión TCP (`time_connect`)** | 1.45 ms | **2.30 ms** | 3.55 ms |
| **Apretón TLS (`time_appconnect`)** | 10.51 ms | **13.95 ms** | 18.45 ms |
| **Primer Byte / TTFB (`time_starttransfer`)** | 13.43 ms | **18.46 ms** | 23.39 ms |
| **Tiempo Total (`time_total`)** | 13.50 ms | **18.53 ms** | 23.46 ms |

**Interpretación de Resultados:** El intercambio de claves TLS promedió **13.95 ms**, constituyendo el **75.2%** de la latencia total del primer acceso. Las conexiones subsecuentes reducen este valor mediante la reutilización de la sesión TLS.

---

## 4. Captura e Interpretación de Tráfico (tcpdump)

Inspección de las tramas capturadas en el archivo `captura_https.pcap` a través de la interfaz de bucle de retorno (`lo`):

* **Establecimiento de Sesión TCP (3-Way Handshake):**
  * `Cliente -> Servidor [SYN]` (`seq 3123783644`, puerto de origen `46382` hacia `443`).
  * `Servidor -> Cliente [SYN, ACK]` (`seq 3670356629`, `ack 3123783645`).
  * `Cliente -> Servidor [ACK]` (`ack 1`, canal de capa 4 establecido).
* **Inicio del Canal TLS 1.3:**
  * `Cliente -> Servidor [PUSH, ACK]` (`Client Hello`, 517 bytes con indicación SNI).
  * `Servidor -> Cliente [PUSH, ACK]` (`Server Hello` + entrega de certificado, 1563 bytes).
* **Intercambio y Cierre:**
  * Envío de datos HTTP cifrados (`Application Data`, 3791 bytes).
  * Desconexión ordenada a nivel TCP mediante la bandera `FIN` (`Flags [F.]`).
