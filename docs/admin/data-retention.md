# Retention Settings

This section details various administrative settings available in KrakenHashes.

## Data Retention

KrakenHashes includes a data retention feature to automatically purge old hashlists and associated data. Administrators can configure the default retention policy that applies system-wide unless overridden by client-specific settings.

### Default Retention Policy

-   **Purpose:** Sets the default number of months after which a hashlist is considered "old" and eligible for automatic deletion.
-   **Scope:** This setting applies to all hashlists *not* associated with a specific client, or to hashlists associated with clients that do not have their own retention policy configured.
-   **Mechanism:** A background job runs periodically (typically daily) to identify and delete hashlists older than the configured retention period.

### Configuration via API

Administrators can view and update the default retention policy using the following API endpoints:

-   **`GET /api/admin/settings/retention`**
    -   **Description:** Retrieves the current default retention settings.
    -   **Response Body (Example):**
        ```json
        {
          "default_retention_days": 90,
          "retention_enabled": true
        }
        ```
    -   `default_retention_days`: Number of days to retain hashlists by default. A value of 0 or less usually indicates retention is disabled or indefinite (check `retention_enabled`).
    -   `retention_enabled`: Boolean indicating if the default policy is active.

-   **`PUT /api/admin/settings/retention`**
    -   **Description:** Updates the default retention settings.
    -   **Request Body (Example):**
        ```json
        {
          "default_retention_days": 120,
          "retention_enabled": true
        }
        ```
    -   Requires administrator privileges.
    -   Updates the system-wide default values.

### Important Notes

-   The retention job deletes the hashlist metadata, its association with hashes (`hashlist_hashes` entries), and the original uploaded file from storage.
-   It does **not** delete individual hashes from the central `hashes` table, as they might belong to other, non-expired hashlists.
-   Client-specific retention settings take precedence over these default settings. See [Client Management](./client-management.md) for details. 