# KrakenHashes Documentation

Welcome to the KrakenHashes documentation. This guide covers installation, configuration, and usage of the distributed password cracking system.

## Documentation Structure

### 🚀 Getting Started
- [Quick Start Guide](quick-start.md) - Get up and running in 5 minutes
- [Installation Guide](installation.md) - Complete installation guide for production and development
- [Docker Initialization](docker/initialization.md) - Legacy Docker setup documentation

### 👤 User Documentation
- [Understanding Jobs and Workflows](user/understanding_jobs_and_workflows.md) - How preset jobs and workflows optimize password cracking
- [Wordlists and Rules](user/wordlists_and_rules.md) - Working with wordlists and transformation rules

### 🔧 Administrator Documentation

#### Core Features
- [Agent Management](agents/) - Complete guide to agent configuration and management
  - [Agent Scheduling](agents/scheduling.md) - Configure working hours for agents
- [Preset Jobs and Workflows](admin/preset_jobs_and_workflows.md) - Creating and managing attack strategies
- [Client Management](admin/client-management.md) - Managing clients and engagements
- [Wordlists and Rules](admin/wordlists_and_rules.md) - Managing wordlists and rule files
- [Agent File Synchronization](admin/agent_file_sync.md) - How file sync works between backend and agents

#### Security & Authentication
- [Authentication Settings](admin/authentication_settings.md) - Configuring authentication and MFA
- [Data Retention](admin/data-retention.md) - Managing data lifecycle and compliance

#### Email System
- [Email Settings](admin/email/email_settings.md) - Configuring email providers
- [Email Templates](admin/email/email_templates.md) - Customizing email notifications

### 📋 Feature Documentation
- [Hashlists](features/hashlists.md) - Understanding hashlist management

## Quick Links

### 🆕 New to KrakenHashes?
1. Start with the [Quick Start Guide](quick-start.md) - Running in 5 minutes
2. Review [Installation Guide](installation.md) for production deployment options
3. Understand [Jobs and Workflows](user/understanding_jobs_and_workflows.md)

### For Users
1. Start with [Understanding Jobs and Workflows](user/understanding_jobs_and_workflows.md) to learn how KrakenHashes automates password auditing
2. Review [Wordlists and Rules](user/wordlists_and_rules.md) to understand available attack resources

### For Administrators
1. Begin with [Installation Guide](installation.md) for production deployment
2. Configure [Authentication Settings](admin/authentication_settings.md) for security
3. Set up [Agent Management](agents/) including [Scheduling](agents/scheduling.md)
4. Create [Preset Jobs and Workflows](admin/preset_jobs_and_workflows.md) for your organization
5. Manage [Clients](admin/client-management.md) and [Data Retention](admin/data-retention.md)

### For Developers
See the main [README.md](../README.md) for development setup and the [CLAUDE.md](../CLAUDE.md) for codebase guidance.

## Documentation Status

This documentation is under active development. Some features described may not be fully implemented as KrakenHashes is currently in alpha (v0.1.0).

## Contributing

Documentation improvements are welcome after v1.0 release. Until then, please report issues via GitHub.