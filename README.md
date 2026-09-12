# arquitectura-web-2026
Proyecto Integrador - Arquitectura Web 2026: Despliegue de PocketBase en Ubuntu Server 24.04 LTS con Nginx Reverse Proxy y systemd.

# Proyecto Integrador — Arquitectura Web 2026

**Estudiantes:** David López, Junior Legal

**Institución:** Facultad Politécnica - Universidad Nacional de Asunción (FPUNA)  
**Sistema Operativo:** Ubuntu Server 24.04.4 LTS (x86_64)  
**Infraestructura:** VirtualBox (2 vCPU, 2 GB RAM, 20 GB Disk)  

---

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
* **Contexto:** Se requiere desplegar una aplicación observable sobre Ubuntu Server 24.04 LTS optimizando los recursos asignados en la máquina virtual (2 vCPU, 2 GB RAM).
* **Decisión:** 
  1. Se elige **PocketBase** como backend debido a su arquitectura de binario único embebido con SQLite, eliminando dependencias externas pesadas.
  2. Se establece el modo de red **Adaptador Puente (*Bridged*)** para integrar la VM directamente a la subred física (`192.168.1.0/27`) y habilitar administración remota vía SSH.
  3. Se define **Nginx** como proxy inverso para exponer la aplicación en el puerto HTTP estándar `80`.
* **Consecuencias:** Despliegue altamente eficiente, facilidad de gestión como servicio de `systemd` y bajo consumo de recursos de hardware.

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
CPU(s):              2
Model name:          AMD FX(tm)-4300 Quad-Core Processor

$ free -h
Mem Total: 1.9Gi | Used: 316Mi | Free: 1.5Gi

# Red e Interfaces
$ ip a
inet 192.168.1.25/27 brd 192.168.1.31 scope global dynamic enp0s3

# Puertos Escuchando en el Sistema
$ sudo ss -tulpn
Netid   State    Local Address:Port
tcp     LISTEN   0.0.0.0:22 (sshd)
```
---

## 6. Evidencia de Funcionamiento

<img width="1524" height="766" alt="Captura de pantalla 2026-09-12 185903 png" src="https://github.com/user-attachments/assets/b93043b4-bb3a-49b8-83a4-0c21bfa189ce" />
