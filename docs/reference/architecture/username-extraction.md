# Username and Domain Extraction Architecture

*Version: 1.1+*

## Overview

KrakenHashes implements automatic username and domain extraction from password hash formats that contain identity information. This system enables:

- **User tracking**: Identify which accounts have been cracked
- **Domain mapping**: Understand organizational structure
- **Prioritization**: Focus on high-value accounts (administrators, machine accounts)
- **Reporting**: Generate client-facing reports with username context
- **Statistics**: Analyze crack rates by user, domain, or account type

## Architecture Components

### 1. Database Schema

**Migration 000070** adds domain support to the hashes table:

```sql
ALTER TABLE hashes
ADD COLUMN domain TEXT;

CREATE INDEX idx_hashes_domain ON hashes(domain) WHERE domain IS NOT NULL;
CREATE INDEX idx_hashes_username ON hashes(username) WHERE username IS NOT NULL;
```

**Schema Fields:**
- `username` (TEXT, nullable): Extracted username
- `domain` (TEXT, nullable): Extracted domain/realm
- `original_hash` (TEXT): Complete hash line as uploaded
- `hash_value` (TEXT): Canonical hash for hashcat processing

### 2. Extraction Pipeline

```
Upload → Type Detection → Line Parse → Extract Username/Domain → Store → Display
```

**Detailed Flow:**

1. **Upload**: User uploads hashlist via API (`POST /api/hashlists`)
2. **Type Selection**: User specifies hash type (e.g., 1000 for NTLM)
3. **Async Processing**: Background worker processes file line-by-line
4. **Extraction**: Each line processed by appropriate extractor
5. **Storage**: Username, domain, and hash stored in database
6. **Indexing**: Indexed for fast filtering and searching
7. **Display**: Surfaced in UI tables and reports

### 3. Extractor System

**Location**: `backend/pkg/hashutils/processing.go`

**Main Function:**
```go
func ExtractUsernameAndDomain(rawHash string, hashTypeID int) *UsernameAndDomain
```

**Type-Specific Extractors:**

| Hash Type | Function | Pattern |
|-----------|----------|---------|
| 1000 (NTLM) | `extractNTLM()` | `DOMAIN\user:sid:LM:NT:::` |
| 1100 (DCC) | `extractDCC()` | `hash:username` |
| 5500 (NetNTLMv1) | `extractNetNTLMv1()` | `user::domain:chal:resp` |
| 5600 (NetNTLMv2) | `extractNetNTLMv2()` | `user::domain:chal:resp` |
| 18200 (Kerberos) | `extractKerberos()` | `$krb5asrep$23$user@domain:hash` |
| 6800 (LastPass) | `extractLastPass()` | `hash:iterations:email` |

**Priority Order:**
1. Custom type-specific extractor (if available)
2. Heuristic fallback extractor
3. Return `nil` if no match

### 4. Domain Parsing

**Function**: `ParseDomainUsername(input string) (username, domain string)`

Handles two standard formats:

**Windows/NetBIOS Format:**
```
DOMAIN\username    →    ("username", "DOMAIN")
CORP\alice         →    ("alice", "CORP")
```

**Kerberos/UPN Format:**
```
user@domain        →    ("user", "domain")
john@CORP.LOCAL    →    ("john", "CORP.LOCAL")
```

**Edge Cases:**
- Multiple `\` characters: Only first is domain separator
- Multiple `@` characters: Only last is domain separator
- No separator found: Domain returns as empty string

### 5. Machine Account Handling

**Identification**: Usernames ending with `$` character

**Examples:**
```
COMPUTERNAME$
DC01$
WKS-001$
SERVER-WEB$
```

**Special Handling:**
- `$` is NOT treated as a separator character
- Trailing `$` is preserved in username field
- Enables identification of computer accounts vs user accounts

**Security Significance:**
- Computer accounts often have elevated privileges
- Can be used for lateral movement in domain environments
- Important to track in penetration testing engagements

## Extractor Implementation Details

### NTLM Extractor (Mode 1000)

**Format**: `DOMAIN\username:sid:LM_hash:NT_hash:::`

```go
func extractNTLM(rawHash string) *UsernameAndDomain {
    parts := strings.Split(rawHash, ":")
    if len(parts) >= 4 {
        // This is pwdump format
        userPart := parts[0]
        username, domain := ParseDomainUsername(userPart)
        return &UsernameAndDomain{
            Username: &username,
            Domain:   ptrIfNotEmpty(domain),
        }
    }
    return nil
}
```

**Example:**
```
Input:  CONTOSO\Administrator:500:aad3b...:8846f7...:::
Output: username="Administrator", domain="CONTOSO"
```

### Kerberos Extractor (Mode 18200)

**Format**: `$krb5asrep$23$user@domain.com:hash_data`

**Challenge**: Preserving `$` in machine account names while parsing format delimiters

**Original Bug** (Fixed in this branch):
```go
// ❌ BUGGY: Splits on ALL $ characters
parts := strings.Split(rawHash, "$")
// Input: $krb5asrep$23$WKS01$@CORP.COM:...
// Result: ["", "krb5asrep", "23", "WKS01", "@CORP.COM:..."]
//         loses the $ in WKS01$!
```

**Fixed Implementation**:
```go
// ✅ FIXED: Find exactly the first 3 $ delimiters
idx1 := strings.Index(rawHash, "$")
idx2 := strings.Index(rawHash[idx1+1:], "$") + idx1 + 1
idx3 := strings.Index(rawHash[idx2+1:], "$") + idx2 + 1
userDomainPart := rawHash[idx3+1:]  // Everything after 3rd $

// Now extract user@domain
colonIdx := strings.Index(userDomainPart, ":")
if colonIdx != -1 {
    userDomainPart = userDomainPart[:colonIdx]
}

// Input: $krb5asrep$23$WKS01$@CORP.COM:hash...
// userDomainPart: "WKS01$@CORP.COM"
// Result: username="WKS01$", domain="CORP.COM" ✓
```

### NetNTLMv2 Extractor (Mode 5600)

**Format**: `username::domain:server_challenge:HMAC_MD5`

```go
func extractNetNTLMv2(rawHash string) *UsernameAndDomain {
    parts := strings.Split(rawHash, ":")
    if len(parts) >= 3 && parts[1] == "" {
        username := parts[0]
        domain := ptrIfNotEmpty(parts[2])
        return &UsernameAndDomain{
            Username: &username,
            Domain:   domain,
        }
    }
    return nil
}
```

**Examples:**
```
alice::ENTERPRISE:1122...   →  username="alice", domain="ENTERPRISE"
WKS99$::DOMAIN:abcd...      →  username="WKS99$", domain="DOMAIN"
testuser:::fedcba...        →  username="testuser", domain=NULL
```

### LastPass Extractor (Mode 6800)

**Format**: `hash:iterations:email`

```go
func extractLastPass(rawHash string) *UsernameAndDomain {
    parts := strings.Split(rawHash, ":")
    if len(parts) == 3 {
        email := parts[2]
        return &UsernameAndDomain{
            Username: &email,
            Domain:   nil, // LastPass doesn't have a domain concept
        }
    }
    return nil
}
```

**Examples:**
```
a1b2c3...:500:john@example.com      →  username="john@example.com", domain=NULL
b2c3d4...:1000:alice@corp.com       →  username="alice@corp.com", domain=NULL
```

### DCC/MS Cache Extractor (Mode 1100)

**Format**: `hash:username`

```go
func extractDCC(rawHash string) *UsernameAndDomain {
    parts := strings.Split(rawHash, ":")
    if len(parts) == 2 {
        username := parts[1]
        return &UsernameAndDomain{
            Username: &username,
            Domain:   nil,
        }
    }
    return nil
}
```

**Examples:**
```
a1b2c3...:jsmith              →  username="jsmith", domain=NULL
b2c3d4...:administrator       →  username="administrator", domain=NULL
c3d4e5...:WKS01$              →  username="WKS01$", domain=NULL
```

### Heuristic Fallback Extractor

Used when no type-specific extractor matches:

**Strategy:**
1. Look for DOMAIN\username pattern first
2. Look for username@domain pattern second
3. Look for hash:username pattern
4. Check if line starts/ends with username patterns
5. Preserve `$` suffix (don't treat as separator)

**Code Excerpt:**
```go
func extractUsernameHeuristic(rawHash string) *UsernameAndDomain {
    // Try DOMAIN\username
    if idx := strings.Index(rawHash, "\\"); idx > 0 {
        username, domain := ParseDomainUsername(rawHash[:...])
        return &UsernameAndDomain{...}
    }

    // Try username@domain
    if strings.Contains(rawHash, "@") {
        parts := strings.Split(rawHash, ":")
        if parts[0] contains "@" {
            username, domain := ParseDomainUsername(parts[0])
            return &UsernameAndDomain{...}
        }
    }

    // Try hash:username format
    parts := strings.Split(rawHash, ":")
    if len(parts) == 2 && looks like hash format {
        return &UsernameAndDomain{Username: &parts[1]}
    }

    return nil
}
```

## Database Integration

### Storage Operations

**Batch Insert with Username/Domain:**

```sql
INSERT INTO hashes (hash_value, original_hash, username, domain, hash_type_id, is_cracked, password)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (original_hash, hash_type_id) DO UPDATE SET
    username = COALESCE(hashes.username, EXCLUDED.username),
    domain = COALESCE(hashes.domain, EXCLUDED.domain),
    is_cracked = EXCLUDED.is_cracked OR hashes.is_cracked,
    password = COALESCE(EXCLUDED.password, hashes.password)
```

**Query Patterns:**

```sql
-- Filter by domain
SELECT * FROM hashes WHERE domain = 'CONTOSO';

-- Filter by username
SELECT * FROM hashes WHERE username LIKE 'admin%';

-- Machine accounts only
SELECT * FROM hashes WHERE username LIKE '%$';

-- Cracked domain admins
SELECT * FROM hashes
WHERE domain = 'ENTERPRISE'
  AND username = 'Administrator'
  AND is_cracked = true;
```

### Performance Considerations

**Indexes Created:**
```sql
CREATE INDEX idx_hashes_domain ON hashes(domain) WHERE domain IS NOT NULL;
CREATE INDEX idx_hashes_username ON hashes(username) WHERE username IS NOT NULL;
```

**Query Optimization:**
- Domain and username fields are indexed
- Partial indexes only index non-NULL values
- Enables fast filtering in UI and API endpoints

## API Integration

### Hashlist Hash Endpoint

**Endpoint**: `GET /api/hashlists/{id}/hashes`

**Query Parameters:**
- `limit`: Number of results (default: 500, max: 2000, -1 for all)
- `offset`: Pagination offset
- `search`: Filter across all fields

**Response includes domain:**
```json
{
  "hashes": [
    {
      "id": "uuid",
      "hash_value": "8846f7eaee8fb117ad06bdd830b7586c",
      "original_hash": "CONTOSO\\Administrator:500:aad3...:8846...",
      "username": "Administrator",
      "domain": "CONTOSO",
      "is_cracked": true,
      "password": "Password123!"
    }
  ],
  "total": 1000,
  "limit": 500,
  "offset": 0
}
```

**Sorting**: Results sorted by `is_cracked DESC, id` (cracked hashes first)

**Pagination Limits:**
- Default: 500 hashes per page
- Maximum: 2000 hashes per page
- Special: -1 for unlimited (all hashes)

## Frontend Integration

### HashDetail Interface

```typescript
interface HashDetail {
  id: string;
  hash_value: string;
  original_hash: string;
  username?: string;
  domain?: string;          // Added in v1.1+
  hash_type_id: number;
  is_cracked: boolean;
  password?: string;
  last_updated: string;
}
```

### Display Components

**HashlistHashesTable Component:**
- Displays username, domain, and password columns
- Filters by username or domain via search
- Copy button copies password (if cracked) or hash
- Color-coded status chips

**Component Location**: `frontend/src/components/hashlist/HashlistHashesTable.tsx`

**Features:**
- Paginated table with customizable page sizes
- Real-time search across all fields
- Automatic sorting (cracked hashes first)
- Copy-to-clipboard integration
- Dynamic responsive layout

## Testing

### Test Coverage

Created comprehensive test files in `/tmp/kh-test-hashes/`:

**Hash Types Tested:**
- NTLM pwdump format (7 cases)
- NTLM various formats (6 cases)
- NetNTLMv1 (6 cases)
- NetNTLMv2 (6 cases)
- Kerberos AS-REP (6 cases)
- LastPass (6 cases)
- DCC/MS Cache (6 cases)
- Machine accounts across types (6 cases)
- Edge cases (10 cases)

**Total Test Cases**: 59

**Test Results**: 100% passing after Kerberos bug fix

### Test Examples

**NTLM Pwdump Format:**
```
# Test 1: Standard pwdump with domain
CONTOSO\Administrator:500:aad3b435b51404eeaad3b435b51404ee:8846f7eaee8fb117ad06bdd830b7586c:::
Expected: username="Administrator", domain="CONTOSO"

# Test 2: Plain NTLM without username
8846f7eaee8fb117ad06bdd830b7586c
Expected: username=NULL, domain=NULL
```

**Kerberos Machine Account:**
```
# Test with machine account
$krb5asrep$23$WKS01$@ENTERPRISE.LOCAL:3e156a5f5e5e5e5e5e5e5e5e5e5e5e5e$a1b2c3d4e5f6
Expected: username="WKS01$", domain="ENTERPRISE.LOCAL"
```

**NetNTLMv2 with Empty Domain:**
```
# Test with no domain
testuser:::fedcba9876543210:b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b20101000000000000
Expected: username="testuser", domain=NULL
```

### Known Issues and Fixes

#### Issue: Kerberos Machine Account Bug

**Problem**: Machine accounts with `$` in username corrupted during extraction

**Example Failure:**
```
Input:  $krb5asrep$23$WKS01$@ENTERPRISE.LOCAL:hash...
Expect: username="WKS01$", domain="ENTERPRISE.LOCAL"
Actual: username="WKS01", domain=NULL ❌
```

**Root Cause**: `strings.Split(rawHash, "$")` split on ALL `$` characters, including the one in the username

**Fix**: Implemented targeted delimiter finding to locate exactly the first 3 `$` separators

**Status**: ✅ Fixed in feature branch `feature/enhanced-username-domain-extraction`

**Impact**: All 6 Kerberos test cases now passing

## Best Practices

### For Administrators

1. **Verify Hash Type**: Always select correct hash type when uploading
2. **Check Extraction**: Review username/domain extraction in detail view
3. **Filter by Domain**: Use domain field to focus on specific organizations
4. **Prioritize Targets**: Focus on administrative accounts and machine accounts

### For Developers

1. **Test Edge Cases**: Always test with machine accounts and special characters
2. **Preserve Original**: Never lose information from original hash line
3. **Handle NULLs**: Domain may be NULL for many hash types
4. **Index Appropriately**: Use partial indexes on nullable fields
5. **Document Formats**: Clearly document expected input formats

### For Penetration Testers

1. **Identify High-Value Targets**: Look for Administrator, Domain Admin accounts
2. **Track Machine Accounts**: Computer accounts can provide lateral movement
3. **Domain Mapping**: Use domain field to understand organizational structure
4. **Password Reuse**: Check if same passwords used across different domains

## Implementation Details

### File Locations

**Backend:**
- `backend/pkg/hashutils/processing.go` - Extraction functions
- `backend/internal/repository/hash_repository.go` - Database queries
- `backend/internal/processor/hashlist_processor.go` - Upload processing
- `backend/db/migrations/000070_add_domain_to_hashes.up.sql` - Schema migration

**Frontend:**
- `frontend/src/components/hashlist/HashlistHashesTable.tsx` - Table component
- `frontend/src/components/hashlist/HashlistDetailView.tsx` - Detail page
- `frontend/src/types/` - TypeScript interfaces

### Code Changes Summary

**Migration 000070:**
- Added `domain` TEXT column to `hashes` table
- Created partial indexes on `domain` and `username`

**Processing Pipeline:**
- Modified `hashlist_processor.go` to call extraction functions
- Updated batch insert to include username and domain
- Maintained backward compatibility

**Repository Layer:**
- Updated all SQL queries to include domain field
- Added COALESCE logic to preserve existing values
- Added sorting by cracked status

**API Layer:**
- Increased pagination limits (500 default, 2000 max)
- Added support for `-1` (unlimited results)
- Included domain in JSON responses

**Frontend:**
- Created new paginated table component
- Added domain column to table display
- Implemented copy-to-clipboard for passwords

## Future Enhancements

**Potential Improvements:**
- Automatic detection of privileged accounts (admin, root, etc.)
- Domain hierarchy visualization
- Cross-hashlist username tracking
- Machine account identification in UI
- Export reports grouped by domain
- Statistics by user and domain
- LDAP/Active Directory integration for username validation
- Username-based reporting and analytics

**API Enhancements:**
- Filter endpoint by username or domain
- Aggregation queries (count by domain, etc.)
- Export functionality with username/domain fields

**UI Enhancements:**
- Domain tree view for enterprise environments
- User-specific crack statistics
- Highlight privileged accounts in UI
- Filter presets (e.g., "Show only machine accounts")

## References

- Hashcat hash mode reference: https://hashcat.net/wiki/doku.php?id=example_hashes
- Active Directory machine accounts: https://docs.microsoft.com/en-us/windows/security/identity-protection/access-control/active-directory-accounts
- Kerberos authentication: https://datatracker.ietf.org/doc/html/rfc4120
- NTLM authentication: https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-nlmp/

---

*Last Updated: October 2025*
*Feature Branch: `feature/enhanced-username-domain-extraction`*
*Migration: 000070*
*Version: 1.1+*
