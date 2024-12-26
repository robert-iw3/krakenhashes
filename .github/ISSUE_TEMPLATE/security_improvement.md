---
name: Security Improvement
about: Template for security-related improvements
title: 'Security: Improve CA Certificate Directory Permissions Handling'
labels: security, enhancement
assignees: ''
---

## Security Improvement: CA Certificate Directory Permissions

### Current Behavior
- Backend attempts to create and access `/etc/hashdom/ca` directory directly
- Requires elevated privileges to create/write to `/etc` directory
- Application needs to run with higher privileges than necessary
- No separation between setup and runtime permissions

### Proposed Solution
Implement a secure permission model that separates initial setup from runtime:

1. **Setup Phase (Elevated Privileges)**
   - Create a separate `hashdom-setup` binary for initialization
   - Create required directories
   - Generate initial CA certificates if needed
   - Set proper ownership and permissions
   - Can be run with sudo/root during installation

2. **Runtime Phase (Restricted Privileges)**
   - Main application runs as non-privileged user
   - Uses pre-configured directories and permissions
   - Follows principle of least privilege
   - Clear documentation of required permissions

3. **Implementation Details**
   - Create dedicated service user/group (`hashdom`)
   - Directory structure:
     ```
     /etc/hashdom/
     ├── ca/
     │   ├── ca.crt (0644)
     │   └── ca.key (0600)
     └── ... (other configs)
     ```
   - Ownership: `hashdom:hashdom`
   - Directory permissions: `0700`

4. **Deployment Configuration**
   - SystemD service definition
   - Docker container configuration
   - Installation/upgrade scripts
   - Clear documentation

### Security Benefits
1. Minimal runtime privileges
2. Clear separation of concerns
3. Proper file/directory permissions
4. Follows security best practices
5. Easier security auditing

### Implementation Steps
1. [ ] Create `hashdom-setup` command
2. [ ] Implement permission checks in main application
3. [ ] Update installation documentation
4. [ ] Create SystemD service definition
5. [ ] Update Docker configuration
6. [ ] Add security documentation
7. [ ] Create upgrade path for existing installations

### Additional Considerations
- Backup/restore procedures
- Certificate rotation
- Monitoring/alerting for permission changes
- Audit logging for setup operations

### Documentation Updates Needed
- Installation guide
- Security model documentation
- Operations manual
- Upgrade guide

/label ~security ~enhancement ~documentation 