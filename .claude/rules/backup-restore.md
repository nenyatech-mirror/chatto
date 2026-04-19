# Backup & Restore

## Architecture

- `chatto backup` connects to a running NATS server via client config, snapshots all JetStream streams using `jsm.go`'s `SnapshotToDirectory`, and creates a `.tar.gz` archive with a `manifest.json`. Use `--encrypt` for age-encrypted archives (`.tar.gz.age`).
- `chatto restore` extracts the archive and restores each stream using `jsm.go`'s `RestoreSnapshotFromDirectory`. Auto-detects age-encrypted archives. For embedded NATS setups, it starts a temporary in-process NATS server. For external NATS, it connects via the client config.

## Key Files

- `cli/cmd/backup.go` — Backup command, tar/gzip utilities, encrypted archive support, skip list, manifest types
- `cli/cmd/restore.go` — Restore command with conflict handling, age detection, and embedded NATS support
- `cli/cmd/keys.go` — Encryption key export/import with age encryption

## Encryption

All encryption (backup archives and key exports) uses `filippo.io/age` with passphrase-based scrypt recipients. This is the same format as the `age` CLI tool — files are interoperable.

Key functions:
- `createEncryptedTarGz()` / `extractEncryptedTarGz()` — streaming backup encryption
- `encryptKeysToFile()` / `decryptKeysFromFile()` — key export encryption
- `isAgeEncrypted()` — detects age header for auto-detection in restore
- `getPassphrase(flagValue, prompt, confirm)` — shared passphrase input (flag or interactive)

The tar functions are split into streaming versions (`writeTarGz`/`readTarGz` accepting `io.Writer`/`io.Reader`) and file wrappers (`createTarGz`/`extractTarGz`). This enables chaining: file → age → gzip → tar.

## Stream Skip List

The `skipReason()` function in `backup.go` determines which streams are excluded. When adding new KV buckets or streams to core, consider whether they should be backed up:

| Should backup | Should skip |
|---------------|-------------|
| User data, messages, config | Caches (regeneratable) |
| Roles, permissions, memberships | Ephemeral/memory streams |
| Assets (avatars, attachments) | Security-sensitive (encryption keys, auth tokens) |

If you add a new memory-only or cache bucket, add it to `skipReason()`.

## Encryption Keys

Encryption keys (`KV_ENCRYPTION_KEYS`) are intentionally excluded from data backups. This is a security design choice — backups contain only encrypted data, not the keys to decrypt it.

Separate key management commands exist:
- `chatto keys export -o keys.backup` — Exports all per-user encryption keys, encrypted with age
- `chatto keys import keys.backup` — Imports keys; skips users that already have a key (safe to re-run)

Key files: `cli/cmd/keys.go`, `cli/cmd/keys_test.go`

The export file format is version 2: an age-encrypted JSON payload containing a `KeyExport` struct with version, timestamp, and key array.

## Manifest Format (v1)

```json
{
  "version": 1,
  "created_at": "2024-01-01T00:00:00Z",
  "streams": [
    {"name": "KV_INSTANCE", "type": "kv", "messages": 42, "bytes": 1024},
    {"name": "KV_USER_PRESENCE", "type": "skipped", "messages": 0, "bytes": 0}
  ],
  "stats": {
    "total_streams": 10,
    "total_bytes": 102400,
    "duration_ms": 500,
    "skipped": 3,
    "failed": 0
  }
}
```

## Restore Conflict Modes

- `--conflict=error` (default): Fail if any stream exists. Safe for fresh restores.
- `--conflict=skip`: Skip existing streams. Useful for partial restore.
- `--conflict=overwrite`: Delete and recreate. Destructive but complete.
