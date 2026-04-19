# Admin Interface

## Instance Admin vs Space Admin

Two separate authorization concepts:

- **Instance admin**: Configured via `admin.emails` in `chatto.toml`. Can access `/admin` routes to view system-wide data.
- **Space admin**: Per-space role (`RoleAdmin` in permissions.go). Can manage a specific space's settings, rooms, and members.

These are independent - a space admin is not automatically an instance admin and vice versa.

## Privacy Boundary

Instance admins can see operational metadata but NOT user content:

| Can See                            | Cannot See       |
| ---------------------------------- | ---------------- |
| User list (login, email, avatar)   | Message content  |
| Space/room names and member counts | Private messages |
| NATS/JetStream metrics             | File contents    |
| System configuration               | User passwords   |

This boundary is intentional. If message visibility is needed for moderation, it should be a separate, auditable feature with explicit consent.

## Backend Authorization

Admin queries use a nested `admin` type pattern. The `Query.admin` resolver checks authorization once and returns `nil` for non-admins:

```go
func (r *queryResolver) Admin(ctx context.Context) (*model.AdminQueries, error) {
    user := auth.ForContext(ctx)
    if user == nil {
        return nil, nil // Not authenticated
    }
    if !isConfigAdmin(ctx, r.core, r.adminConfig, user.Id) {
        return nil, nil // Not an admin
    }
    // Return populated AdminQueries...
}
```

The `isConfigAdmin` helper checks if any of the user's _verified_ emails match the `admin.emails` list. Unverified/pending emails are never matched.

All fields under `admin` (users, spaces, systemInfo) don't need individual auth checks - the parent resolver handles it.

## Configuration

```toml
[admin]
emails = ["admin@example.com", "ops@example.com"]
```

Users are granted instance admin access if any of their verified email addresses matches an entry in this list. The `isConfigAdmin` helper performs the matching - only verified emails are considered, never pending/unverified ones.
