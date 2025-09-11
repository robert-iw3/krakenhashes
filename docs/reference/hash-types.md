# Hash Types Reference

This document provides comprehensive information about the hash types supported by KrakenHashes for password cracking operations.

## Overview

Hash types in KrakenHashes correspond to hashcat modes and define the algorithm and format used to hash passwords. Each hash type has a unique numerical identifier (mode number) that matches hashcat's mode system. KrakenHashes currently supports **504 different hash types** covering a wide range of algorithms and applications.

## Hash Type Categories

### Fast Hash Types
Fast hash algorithms are computationally inexpensive and can be cracked at high speeds. These include basic cryptographic hashes and simple salted variants.

**Examples:**
- MD5 (mode 0)
- SHA1 (mode 100)
- SHA2-256 (mode 1400)
- SHA2-512 (mode 1700)
- NTLM (mode 1000)

### Slow Hash Types
Slow hash algorithms are intentionally computationally expensive to resist brute-force attacks. These require significantly more time and resources to crack but provide better security. KrakenHashes supports **132 slow hash types**.

**Examples:**
- bcrypt (mode 3200)
- PBKDF2 variants (modes 10900, 12000, 12100)
- scrypt (mode 8900)
- Argon2 variants
- TrueCrypt/VeraCrypt containers

### Application-Specific Hashes
Many hash types are specific to particular applications, operating systems, or protocols.

**Categories include:**
- **Database Systems**: MySQL, PostgreSQL, Oracle, MSSQL
- **Operating Systems**: Windows (NTLM, NetNTLM), Unix/Linux (crypt variants), macOS
- **Web Applications**: WordPress, Joomla, Django, Drupal
- **Network Protocols**: WPA/WPA2, Kerberos, SNMP
- **Archive Formats**: ZIP, RAR, 7-Zip
- **Cryptocurrency**: Bitcoin wallets, Ethereum wallets

## Hash Type Structure

Each hash type in KrakenHashes has the following properties:

- **ID**: Unique numerical identifier (hashcat mode number)
- **Name**: Descriptive name of the hash algorithm
- **Example**: Sample hash format showing the expected input structure
- **Needs Processing**: Flag indicating if special preprocessing is required
- **Is Enabled**: Whether the hash type is currently supported
- **Slow**: Flag indicating if this is a computationally expensive algorithm

## Common Hash Types by Use Case

### Web Application Security Testing
| Mode | Algorithm | Common Applications |
|------|-----------|-------------------|
| 400 | phpass | WordPress, phpBB |
| 500 | md5crypt | Traditional Unix systems |
| 1600 | Apache $apr1$ | Apache htpasswd |
| 7900 | Drupal7 | Drupal CMS |
| 124 | Django (SHA-1) | Django framework |
| 10000 | Django (PBKDF2-SHA256) | Modern Django |

### Enterprise/Active Directory
| Mode | Algorithm | Use Case |
|------|-----------|----------|
| 1000 | NTLM | Windows password hashes |
| 5500 | NetNTLMv1 | Network authentication |
| 5600 | NetNTLMv2 | Network authentication |
| 7500 | Kerberos 5 AS-REQ | Domain authentication |
| 13100 | Kerberos 5 TGS-REP | Ticket Granting Service |

### Database Security
| Mode | Algorithm | Database |
|------|-----------|----------|
| 12 | PostgreSQL | PostgreSQL MD5 |
| 300 | MySQL4.1/MySQL5 | MySQL SHA1 |
| 200 | MySQL323 | Legacy MySQL |
| 131 | MSSQL (2000) | SQL Server |
| 132 | MSSQL (2005) | SQL Server |
| 1731 | MSSQL (2012, 2014) | SQL Server |

### File/Archive Security
| Mode | Algorithm | Application |
|------|-----------|-------------|
| 13000 | RAR5 | WinRAR archives |
| 12500 | RAR3-hp | WinRAR archives |
| 11600 | 7-Zip | 7-Zip archives |
| 17200 | PKZIP (Compressed) | ZIP archives |
| 17210 | PKZIP (Uncompressed) | ZIP archives |

## Performance Considerations

### Speed Classifications

**Ultra-Fast (>1 billion attempts/sec on modern GPUs):**
- MD4 (900), MD5 (0), SHA1 (100)
- Simple salted variants

**Fast (100M-1B attempts/sec):**
- SHA2 variants, NTLM
- Basic application-specific hashes

**Medium (1M-100M attempts/sec):**
- Multiple iteration hashes
- Complex salted variants

**Slow (<1M attempts/sec):**
- PBKDF2, bcrypt, scrypt
- Full disk encryption
- Cryptocurrency wallets

### Resource Requirements

**GPU Memory Considerations:**
- Large wordlists may require significant GPU memory
- Rule-based attacks can multiply memory requirements
- Some hash types have higher per-hash memory overhead

**CPU vs GPU Performance:**
- Most hash types benefit significantly from GPU acceleration
- Some algorithms may perform better on CPU for small datasets
- Hybrid attacks may utilize both CPU and GPU resources

## Hash Format Examples

### Basic Formats
```
MD5:           8743b52063cd84097a65d1633f5c74f5
SHA1:          b89eaac7e61417341b710b727768294d0e6a277b
SHA256:        127e6fbfe24a750e72930c220a8e138275656b8e5d8f48a98c3c92df2caba935
NTLM:          b4b9b02e6f09a9bd760f388b67351e2b
```

### Salted Formats
```
md5($pass.$salt):     01dfae6e5d4d90d9892622325959afbe:7050461
sha1($salt.$pass):    cac35ec206d868b7d7cb0b55f31d9425b075082b:5363620024
sha256($pass.$salt):  c73d08de890479518ed60cf670d17faa26a4a71f995c1dcc978165399401a6c4:53743528
```

### Application-Specific Formats
```
WordPress:      $P$984478476IagS59wHZvyQMArzfx58u.
bcrypt:         $2a$05$LhayLxezLhK1LhWvKxCyLOj0j1u.Kj0jZ0pEmm134uzrQlFvQJLF6
Django:         pbkdf2_sha256$20000$H0dPx8NeajVu$GiC4k5kqbbR9qWBlsRgDywNqC2vd9kqfk7zdorEnNas=
NetNTLMv2:      admin::N46iSNekpT:08ca45b7d7ea58ee:88dcbe4446168966a153a0064958dac6
```

## Hash Type Identification

### Automatic Detection
KrakenHashes can often identify hash types based on:
- **Length**: Different algorithms produce different output lengths
- **Character set**: Hex vs base64 vs custom encoding
- **Delimiters**: Colons, dollar signs, or other separators
- **Prefixes**: Algorithm identifiers like `$2a$`, `{SHA}`, etc.

### Manual Identification Guidelines

**By Length (hex-encoded):**
- 32 characters: MD5, NTLM, MD4
- 40 characters: SHA1, MySQL4.1
- 56 characters: SHA2-224
- 64 characters: SHA2-256, BLAKE2s-256
- 96 characters: SHA2-384
- 128 characters: SHA2-512, BLAKE2b-512

**By Format Patterns:**
- `$algorithm$`: crypt-style formats (bcrypt, sha512crypt, etc.)
- `{ALGORITHM}`: LDAP-style formats
- `hash:salt`: Simple salted hashes
- Complex delimited formats for specific applications

### Common Identification Mistakes
- Confusing MD5 with NTLM (both 32 hex characters)
- Misidentifying base64-encoded vs hex-encoded hashes
- Not recognizing application-specific wrapper formats

## Best Practices

### Hash Type Selection
1. **Verify the source**: Confirm the application/system that generated the hashes
2. **Check format carefully**: Pay attention to delimiters, prefixes, and encoding
3. **Test with known samples**: Use test hashes to verify correct identification
4. **Consider variations**: Many applications have multiple hash format variants

### Performance Optimization
1. **Start with fast hashes**: Test identification with quick attacks first
2. **Use appropriate wordlists**: Match wordlist complexity to hash strength
3. **Consider slow hash implications**: Budget time appropriately for PBKDF2, bcrypt, etc.
4. **Monitor resource usage**: Slow hashes can consume significant GPU memory

### Security Considerations
1. **Handle sensitive data properly**: Ensure secure storage and transmission
2. **Use appropriate attack methods**: Don't waste resources on over-engineered attacks
3. **Respect rate limits**: Some hash types may benefit from attack rate limiting
4. **Document findings**: Keep records of successful techniques for similar engagements

## Advanced Features

### Processing Requirements
Some hash types require special preprocessing before cracking:
- **NTLM (mode 1000)**: Requires UTF-16LE encoding conversion
- Character set normalization for international passwords
- Case conversion requirements for specific applications

### Multi-Hash Support
KrakenHashes supports attacking multiple hashes of the same type simultaneously:
- Efficient memory usage for large hashlist processing
- Optimized GPU kernels for batch operations
- Progress tracking per individual hash

### Custom Hash Types
For specialized requirements:
- Contact development team for custom hash type implementation
- Provide detailed specification and test vectors
- Consider performance implications for custom algorithms

## Troubleshooting

### Common Issues
1. **Hash not cracking**: Verify hash type identification
2. **Slow performance**: Check if hash type is marked as "slow"
3. **GPU errors**: Some hash types require specific GPU capabilities
4. **Memory errors**: Large hashlists may exceed available GPU memory

### Getting Help
- Check the hash format against provided examples
- Verify the source application and version
- Test with known password/hash pairs
- Consult the troubleshooting guide for hardware-specific issues

## Complete Hash Type List

For a complete list of all supported hash types with examples, consult the database directly or use the admin interface. The list includes detailed information about:
- Algorithm specifications
- Example hash formats
- Performance characteristics
- Special requirements or limitations

---

*Last updated: Based on KrakenHashes database with 504 supported hash types*
*For the most current information, always check the application's hash type management interface*