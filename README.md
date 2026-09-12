# arquitectura-web-2026
Proyecto Integrador - Arquitectura Web 2026: Despliegue de PocketBase en Ubuntu Server 24.04 LTS con Nginx Reverse Proxy y systemd.

# Proyecto Integrador — Arquitectura Web 2026

**Estudiante:** David López, Junior Legal,
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
