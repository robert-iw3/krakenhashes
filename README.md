# KrakenHashes

KrakenHashes is a distributed password cracking system designed for security professionals and red teams. The platform coordinates GPU/CPU resources across multiple agents to perform high-speed hash cracking using tools like Hashcat through a secure web interface. Think of KrakenHashes as a full management system for hashes during, after and before (if a repeat client). Ideally, while also checking hashes for known cracks, we update a potfile with every hash and that can be used as a first run against other types of hashes for a potential quick win.

![KrakenHashes Dashboard](docs/assets/images/screenshots/dashboard_overview.png)

## Disclaimer

**⚠️ Active Development Warning**  
This project is currently in **beta development**. Key considerations:

> **Use at your own risk** - This software may eat your data, catch fire, or summon a digital Kraken. You've been warned.

## Component Details

### Backend Service (Go)

-   Job scheduler with adaptive load balancing
-   REST/gRPC API endpoints with JWT authentication
-   PostgreSQL interface for job storage/results
-   Redis-based task queue with priority levels
-   Prometheus metrics exporter

### Agent System (Go)

-   Hardware resource manager (GPU/CPU/RAM allocation)
-   Hashcat wrapper with automatic checkpointing
-   Safety mechanisms for temperature/usage limits
-   Distributed work unit management
-   Healthcheck system with self-healing capabilities

### Web Interface (React)

-   Real-time job progress visualization
-   Hash type detection and configuration wizard
-   Team management dashboard for admins
-   MFA configuration and recovery flow
-   Interactive reporting and analytics

## Security Highlights

-   Automatic session invalidation on IP change
-   Role-based access control (RBAC) system
-   Encrypted job payloads (AES-256-GCM)
-   Certificate-pinned agent communications
-   Audit-quality logging with chain-of-custody

## Use Cases

-   Penetration testing teams coordinating attacks
-   Forensic investigators recovering protected evidence
-   Red teams executing credential stuffing attacks
-   Research analyzing hash vulnerabilities
-   Security training environments

> **License**: AGPLv3 (See LICENSE.md)  
> **Status**: Actively in development, there will be bugs and major braking changes

## Documentation

Comprehensive documentation is available in the [docs/](docs/) directory:

-   **[Quick Start](dhttps://zerkereod.github.io/krakenhashes/getting-started/quick-start/)** - Quick start guide for installation
-   **[Documentation Index](https://zerkereod.github.io/krakenhashes/)** - Complete documentation overview
-   **[User Guide](https://zerkereod.github.io/krakenhashes/user-guide/)** - Understanding jobs and workflows
-   **[Admin Guide](https://zerkereod.github.io/krakenhashes/admin-guide/)** - Creating and managing attack strategies
-   **[Docker Setup](dhttps://zerkereod.github.io/krakenhashes/deployment/docker/)** - Getting started with Docker

## Development

Instructions for setting up and running each component can be found in their respective directories.

### Version 2.0 Considerations

-   [ ] Passkey support for MFA
-   [ ] Additional authentication methods
-   [ ] Advanced job dependencies
-   [ ] Enhanced benchmarking with historical tracking
-   [ ] Job queuing and scheduling improvements
-   [ ] POT statistics and analytics
-   [ ] Team system implementation
    -   [ ] Team management infrastructure
        -   [ ] Team manager roles
        -   [ ] User-team assignments
        -   [ ] Team-based agent access control
    -   [ ] Frontend team interfaces
        -   [ ] Team management UI
        -   [ ] Team assignment system
        -   [ ] Team management guidelines
-   [ ] Statistics and analytics (move to v2.0)
