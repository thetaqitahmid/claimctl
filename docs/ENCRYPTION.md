# Encryption of Sensitive Information

claimctl employs encryption at rest for specific sensitive data to ensure
security. This document outlines the encryption mechanisms, the data covered,
and key management procedures.

## Encryption Algorithm

claimctl uses **AES-256-GCM** (Advanced Encryption Standard in
Galois/Counter Mode) for all data encryption. This provides both confidentiality
and authenticity (ensuring the data hasn't been tampered with).

- **Key Size**: 256 bits (32 bytes).
- **Mode**: GCM with a unique nonce for every encryption operation.
- **Encoding**: Encrypted values are stored as Base64-encoded strings in the
  database.

## Key Management

The encryption key is managed via the `APP_ENCRYPTION_KEY` environment variable.

### Key Loading Priority

The backend attempts to load the key in the following order:

1. **Environment Variable**: `APP_ENCRYPTION_KEY` (Base64 encoded 32-byte
   string).
2. **Key File**: `./keys/app.key`
3. **Automatic Generation**: If no key is found, the system generates a new
   random 32-byte key and saves it to `./keys/app.key`.

> [!IMPORTANT] > **Keep your `APP_ENCRYPTION_KEY` safe.** If this key is lost,
> all data encrypted with it will be unrecoverable. For high-availability or
> migrations, ensure the same key is shared across all instances.

## Encrypted Data

Currently, the following data is encrypted at rest:

### Application Settings (`app_settings`)

Settings marked as `is_secret` (such as SMTP passwords, LDAP bind passwords, or
OIDC client secrets) are automatically encrypted before being saved to the
database.

### API Tokens

API tokens are not stored in plain text. Instead, a **SHA-256 hash** of the
token is stored. The token itself is only shown to the user once upon creation.

### Secrets & Webhooks (Internal Note)

While claimctl provides a "Secrets" management interface and "Webhook"
headers, these are currently stored in **plain text** in the database. Use with
caution in environments where database access is not strictly controlled.

## Security Best Practices

1. **Environment Variables**: In production, prefer setting `APP_ENCRYPTION_KEY`
   as an environment variable rather than relying on the file-based key.
2. **Key Rotation**: While the system doesn't currently support automatic key
   rotation, manual rotation would require decrypting all sensitive data with
   the old key and re-encrypting it with the new key.
3. **Database Access**: Restrict access to the PostgreSQL database to the
   claimctl backend service only.
4. **Backup Security**: Ensure that database backups are stored in a secure,
   encrypted location, as they contain the encrypted secrets.
