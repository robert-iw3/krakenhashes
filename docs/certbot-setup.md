# Certbot Setup Guide for KrakenHashes

This guide explains how to configure KrakenHashes to use Let's Encrypt certificates via Certbot with Cloudflare DNS-01 challenge. This is ideal for internal applications that aren't publicly accessible.

## Prerequisites

1. **Domain Name**: You need to own a domain (e.g., `zerkersec.io`)
2. **Cloudflare Account**: Your domain must be managed by Cloudflare
3. **API Token**: Create a Cloudflare API token with proper permissions

## Step 1: Create Cloudflare API Token

1. Log in to your Cloudflare dashboard
2. Go to **My Profile** → **API Tokens**
3. Click **Create Token**
4. Use the **Custom token** template with these permissions:
    - **Zone** → **DNS** → **Edit**
    - **Zone** → **Zone** → **Read** (optional but recommended)
5. Under **Zone Resources**, select:
    - **Include** → **Specific zone** → Your domain (e.g., `zerkersec.io`)
6. Click **Continue to summary** and **Create Token**
7. **Save the token securely** - you won't be able to see it again!

## Step 2: DNS Configuration

Create an A record for your subdomain:

1. In Cloudflare dashboard, go to your domain
2. Navigate to **DNS** → **Records**
3. Add a new A record:
    - **Type**: A
    - **Name**: `kraken` (or your chosen subdomain)
    - **IPv4 address**: Your internal IP (e.g., `10.0.0.100`)
    - **Proxy status**: DNS only (gray cloud)
    - **TTL**: Auto

**Note**: The IP address doesn't need to be publicly accessible. Certbot only needs to verify domain ownership via DNS TXT records.

## Step 3: Configure KrakenHashes

Edit your `.env` file:

```bash
# Change TLS mode to certbot
KH_TLS_MODE=certbot

# Certbot Configuration
KH_CERTBOT_DOMAIN=kraken.zerkersec.io    # Your full domain
KH_CERTBOT_EMAIL=admin@zerkersec.io      # Your email for Let's Encrypt
KH_CERTBOT_STAGING=true                  # Start with staging for testing!
KH_CERTBOT_AUTO_RENEW=true               # Enable automatic renewal

# Cloudflare Configuration
CLOUDFLARE_API_TOKEN=your-token-here     # The token from Step 1

# Update frontend URLs to use your domain
REACT_APP_API_URL=https://kraken.zerkersec.io:31337
REACT_APP_WS_URL=wss://kraken.zerkersec.io:31337

# Update CORS to allow your domain
CORS_ALLOWED_ORIGIN=https://kraken.zerkersec.io
```

## Step 4: Initial Setup with Staging

**Important**: Always test with Let's Encrypt staging environment first to avoid rate limits!

1. Stop existing containers:

    ```bash
    docker-compose down
    ```

2. Start with certbot mode:

    ```bash
    docker-compose up -d
    ```

3. Monitor the logs:

    ```bash
    docker-compose logs -f krakenhashes
    ```

4. Look for messages indicating successful certificate generation:
    ```
    Obtaining certificates for domain: kraken.zerkersec.io
    Successfully obtained certificates
    ```

## Step 5: Verify Staging Certificates

1. Check certificate files:

    ```bash
    ls -la kh-backend/config/certs/live/kraken.zerkersec.io/
    ```

2. You should see:

    - `fullchain.pem` - Certificate chain
    - `privkey.pem` - Private key
    - `chain.pem` - Intermediate certificates
    - `cert.pem` - Domain certificate

3. Test the application (you'll get a browser warning for staging certificates)

## Step 6: Switch to Production

Once staging certificates work:

1. Update `.env`:

    ```bash
    KH_CERTBOT_STAGING=false
    ```

2. Remove staging certificates:

    ```bash
    rm -rf kh-backend/config/certs/live/*
    rm -rf kh-backend/config/certs/archive/*
    rm -rf kh-backend/config/certs/renewal/*
    ```

3. Restart to get production certificates:
    ```bash
    docker-compose down
    docker-compose up -d
    ```

## Certificate Renewal

Certificates are automatically renewed:

-   Renewal checks run twice daily (3 AM and 3 PM)
-   Certificates renew when less than 30 days remain
-   Services reload automatically after renewal

### Manual Renewal

To manually check/renew certificates:

```bash
docker exec krakenhashes /usr/local/bin/certbot-renew.sh
```

### Monitor Renewal

Check renewal logs:

```bash
docker exec krakenhashes tail -f /var/log/krakenhashes/certbot-renew.log
```

## Troubleshooting

### Common Issues

1. **"CLOUDFLARE_API_TOKEN environment variable is required"**

    - Ensure the token is set in your `.env` file
    - Token must have DNS:Edit permissions

2. **"Failed to obtain certificates"**

    - Check Cloudflare API token permissions
    - Verify domain ownership
    - Check certbot logs: `docker exec krakenhashes cat /etc/krakenhashes/certs/logs/letsencrypt.log`

3. **Browser still shows certificate warnings**

    - Ensure you switched from staging to production
    - Clear browser cache
    - Verify certificates: `docker exec krakenhashes openssl x509 -in /etc/krakenhashes/certs/live/kraken.zerkersec.io/cert.pem -text -noout`

4. **Rate Limits**
    - Let's Encrypt has rate limits (50 certificates per domain per week)
    - Always test with staging first
    - See: https://letsencrypt.org/docs/rate-limits/

### Debug Mode

Enable debug logging:

```bash
DEBUG=true
LOG_LEVEL=DEBUG
```

### Certificate Information

View certificate details:

```bash
# Inside container
docker exec krakenhashes certbot certificates --config-dir /etc/krakenhashes/certs

# Certificate expiry
docker exec krakenhashes openssl x509 -enddate -noout -in /etc/krakenhashes/certs/live/kraken.zerkersec.io/cert.pem
```

## Security Notes

1. **API Token**: Never commit your Cloudflare API token to version control
2. **Permissions**: The token only needs DNS edit access for your specific zone
3. **Internal Use**: This setup works for internal applications not exposed to the internet
4. **Certificate Storage**: Certificates are stored in the persistent `kh-backend/config/certs` directory

## Additional Domains

To add more domains/subdomains:

1. Add them to `KH_ADDITIONAL_DNS_NAMES` in `.env`:

    ```bash
    KH_ADDITIONAL_DNS_NAMES=kraken.zerkersec.io,api.zerkersec.io,*.kraken.zerkersec.io
    ```

2. Ensure all domains are in Cloudflare and accessible by your API token

3. Restart the container to obtain certificates for all domains

## Migration from Self-Signed

If migrating from self-signed certificates:

1. Back up existing certificates (optional)
2. Update `.env` as shown above
3. Restart containers
4. Update any clients/browsers that have the old CA certificate cached
