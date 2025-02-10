# KrakenHashes

KrakenHashes is a distributed password cracking system designed for security professionals and red teams. The platform coordinates GPU/CPU resources across multiple agents to perform high-speed hash cracking using tools like Hashcat through a secure web interface.

## Component Details

### Backend Service (Go)
- Job scheduler with adaptive load balancing
- REST/gRPC API endpoints with JWT authentication
- PostgreSQL interface for job storage/results
- Redis-based task queue with priority levels
- Prometheus metrics exporter

### Agent System (Go)
- Hardware resource manager (GPU/CPU/RAM allocation)
- Hashcat wrapper with automatic checkpointing
- Safety mechanisms for temperature/usage limits
- Distributed work unit management
- Healthcheck system with self-healing capabilities

### Web Interface (React)
- Real-time job progress visualization
- Hash type detection and configuration wizard
- Team management dashboard for admins
- MFA configuration and recovery flow
- Interactive reporting and analytics

## Security Highlights
- Automatic session invalidation on IP change
- Role-based access control (RBAC) system
- Encrypted job payloads (AES-256-GCM)
- Certificate-pinned agent communications
- Audit-quality logging with chain-of-custody

## Use Cases
- Penetration testing teams coordinating attacks
- Forensic investigators recovering protected evidence
- Red teams executing credential stuffing attacks
- Research analyzing hash vulnerabilities
- Security training environments

> **License**: AGPLv3 (See LICENSE.md)  
> **Status**: Actively in development, there will be bugs and major braking changes

## Development

Instructions for setting up and running each component can be found in their respective directories.

## Version 1.0 Roadmap

### Core Infrastructure
- [x] Implement TLS support
  - [x] Self-signed certificate support
  - [x] User-provided certificate support (Not Tested - Should work)
  - [x] Certbot integration (Written but not tested - Please open an issue if you have issues)
- [x] Docker containerization
  - [x] Environment variable configuration
  - [x] Database initialization handling
  - [x] Deployment to DockerHub with documentation
- [x] Email notification system
  - [x] Security event notifications (template)
  - [x] Job completion notifications (template)
  - [x] Admin error notifications (template)
  - [x] Email-based MFA (template)

### Authentication & Authorization
- [x] Enhanced user management
  - [x] User groups (Admin/User roles)
  - [x] MFA implementation
    - [x] Email-based authentication
    - [x] Backup codes (based on admin auth settings)
    - [x] Admin MFA override capability
  - [x] Password change functionality
  - [x] Account management features

### Job Processing System
- [ ] Hashlist management
  - [ ] Comprehensive hashcat hash type support
  - [ ] Hash configuration database
    - [ ] Salt status tracking
    - [ ] Performance characteristics (slow/fast)
  - [ ] Agent-side validation with error parsing
- [ ] Task management
  - [ ] Multi-level priority system
    - [ ] FIFO within priority levels
    - [ ] Pre-defined task templates
  - [ ] Intelligent job distribution
    - [ ] Team-based routing
    - [ ] Agent availability tracking
    - [ ] Owner priority handling
  - [ ] Progress tracking
  - [ ] Result storage
- [ ] Resource management
  - [ ] Wordlist upload/management
  - [ ] Rules file management
  - [ ] Tool version management

### Agent Enhancements
- [ ] Job processing
  - [ ] Hashcat integration
    - [ ] Command generation based on hash type
    - [ ] Error handling and reporting
  - [ ] Benchmark system
  - [ ] Dynamic workload calculation
- [ ] Advanced monitoring
  - [ ] GPU/CPU temperature tracking
  - [ ] Resource usage history
  - [ ] Performance metrics
- [ ] Scheduling system
  - [ ] On/Off toggle
  - [ ] Daily schedule configuration
  - [ ] Resource usage limits

### Frontend Features
- [ ] Dashboard
  - [ ] User-specific job status
  - [ ] Performance statistics
  - [ ] System health indicators
- [ ] Task management interface
  - [ ] Job creation/modification
  - [ ] Priority level assignment
  - [ ] Task template management
  - [ ] Progress monitoring
  - [ ] Result viewing
- [ ] Resource management pages
  - [ ] Wordlist management
  - [ ] Rules management
  - [ ] Tool configuration
- [ ] Admin panel
  - [ ] System configuration
  - [ ] User management
    - [ ] MFA management
  - [x] Security settings
- [x] Account management
  - [x] Profile settings
  - [x] Security settings
    - [x] MFA setup/recovery

### Documentation
- [ ] API documentation
- [ ] Deployment guides
- [ ] User manual
  - [ ] Priority system guidelines
  - [ ] Hash type reference
  - [ ] Best practices
- [ ] Administrator guide
  - [ ] Team management guidelines
  - [ ] Security recommendations

### Version 2.0 Considerations
- [ ] Passkey support for MFA
- [ ] Additional authentication methods
- [ ] Team resource quotas
- [ ] Advanced job dependencies
- [ ] Team system implementation
  - [ ] Team management infrastructure
    - [ ] Team manager roles
    - [ ] User-team assignments
    - [ ] Team-based agent access control
  - [ ] Frontend team interfaces
    - [ ] Team management UI
    - [ ] Team assignment system
