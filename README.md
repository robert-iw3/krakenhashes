# KrakenHashes

KrakenHashes is a distributed password cracking system designed for security professionals and red teams. The platform coordinates GPU/CPU resources across multiple agents to perform high-speed hash cracking using tools like Hashcat through a secure web interface.

## Disclaimer

**⚠️ Active Development Warning**  
This project is currently in **early alpha development** as a personal research project. Key considerations:

-   **Not production-ready**: Core functionality is incomplete/unstable
-   **No external contributions**: We are not accepting pull requests until version 1 is released. This is a passion project that I would like to take to ready before getting outside help.
-   **Breaking changes guaranteed**: Database schema and API contracts will change as I want to prevent every minor schema change to add a new migration file.
-   **Zero stability guarantees**: Features may disappear or break without warning
-   **Not for testing**: Core security/auth systems remain unimplemented

**Until v1.0 release:**

-   No compatibility between versions
-   No migration path for existing data
-   No documented upgrade process
-   No stability in configuration formats

**Post v1.0:**  
We plan to implement proper database migration tooling and versioned API contracts once core feature development stabilizes. Until then, consider this codebase a moving target.

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

-   **[Documentation Index](docs/README.md)** - Complete documentation overview
-   **[User Guide](docs/user/understanding_jobs_and_workflows.md)** - Understanding jobs and workflows
-   **[Admin Guide](docs/admin/preset_jobs_and_workflows.md)** - Creating and managing attack strategies
-   **[Docker Setup](docs/docker/initialization.md)** - Getting started with Docker

## Development

Instructions for setting up and running each component can be found in their respective directories.

## Version 1.0 Roadmap

### Core Infrastructure

-   [x] Implement TLS support
    -   [x] Self-signed certificate support
    -   [x] User-provided certificate support (Not Tested - Should work)
    -   [x] Certbot integration (Written but not tested - Please open an issue if you have issues)
-   [x] Docker containerization
    -   [x] Environment variable configuration
    -   [x] Database initialization handling
    -   [x] Deployment to DockerHub with documentation
-   [x] Email notification system
    -   [x] Security event notifications (template)
    -   [x] Job completion notifications (template)
    -   [x] Admin error notifications (template)
    -   [x] Email-based MFA (template)

### Authentication & Authorization

-   [x] Enhanced user management
    -   [x] User groups (Admin/User roles)
    -   [x] MFA implementation
        -   [x] Email-based authentication
        -   [x] Backup codes (based on admin auth settings)
        -   [x] Admin MFA override capability
    -   [x] Password change functionality
    -   [x] Account management features

### Job Processing System

-   [ ] Hashlist management
    -   [x] Comprehensive hashcat hash type support
    -   [ ] Hash configuration database
        -   [ ] Salt status tracking
        -   [ ] Performance characteristics (slow/fast)
    -   [ ] Agent-side validation with error parsing
-   [ ] Task management
    -   [x] Multi-level priority system
        -   [x] Priority-based execution (0-1000)
        -   [x] Pre-defined task templates (Preset Jobs & Workflows)
    -   [ ] Intelligent job distribution
        -   [x] Agent availability tracking
        -   [ ] Owner priority handling
    -   [x] Progress tracking
    -   [x] Result storage
-   [x] Resource management
    -   [x] Wordlist upload/management
    -   [x] Rules file management
    -   [x] Tool version management

### Agent Enhancements

-   [ ] Job processing
    -   [ ] Hashcat integration
        -   [x] Command generation based on hash type
        -   [x] Error handling and reporting
    -   [ ] Benchmark system
        -   [ ] Device-specific benchmark metrics storage
        -   [x] Per-device speed tracking by hash type
    -   [ ] Dynamic workload calculation
-   [ ] Advanced monitoring
    -   [ ] GPU/CPU temperature tracking
    -   [ ] Resource usage history
    -   [ ] Performance metrics
-   [ ] Scheduling system
    -   [ ] On/Off toggle
    -   [ ] Daily schedule configuration
    -   [ ] Resource usage limits

### Frontend Features

-   [ ] Dashboard
    -   [ ] User-specific job status
    -   [ ] Performance statistics
    -   [ ] System health indicators
-   [x] Job management interface
    -   [x] Job creation/modification
    -   [x] Priority level assignment
    -   [x] Job template management
    -   [x] Progress monitoring
    -   [x] Result viewing
-   [x] Resource management pages
    -   [x] Wordlist management
    -   [x] Rules management
    -   [x] Tool configuration
-   [ ] Admin panel
    -   [ ] System configuration
    -   [ ] User management
        -   [ ] MFA management
    -   [x] Security settings
-   [x] Account management
    -   [x] Profile settings
    -   [x] Security settings
        -   [x] MFA setup/recovery

### Documentation

-   [ ] API documentation
-   [ ] Deployment guides
-   [ ] User manual
    -   [ ] Priority system guidelines
    -   [x] Hash type reference
    -   [ ] Best practices
-   [ ] Administrator guide
    -   [ ] Security recommendations

### Version 2.0 Considerations

-   [ ] Passkey support for MFA
-   [ ] Additional authentication methods
-   [ ] Team resource quotas
-   [ ] Advanced job dependencies
-   [ ] Team system implementation
    -   [ ] Team management infrastructure
        -   [ ] Team manager roles
        -   [ ] User-team assignments
        -   [ ] Team-based agent access control
    -   [ ] Frontend team interfaces
        -   [ ] Team management UI
        -   [ ] Team assignment system
        -   [ ] Team management guidelines
