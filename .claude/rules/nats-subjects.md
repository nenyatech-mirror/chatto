# NATS Subject Patterns

## Design Principles

When designing NATS subject patterns, follow these principles:

### 1. Unified Namespaces for Related Events

Group related events under a common prefix so a single wildcard subscription captures all of them:

```
# Good: All messages (root + thread) under msg.>
space.{s}.room.{r}.msg.{eventId}                    # Root message
space.{s}.room.{r}.msg.{rootId}.replies.{eventId}   # Thread reply

# Bad: Separate namespaces require multiple subscriptions
space.{s}.room.{r}.msg.{eventId}                    # Root message
space.{s}.room.{r}.thread.{rootId}.{eventId}        # Thread reply
```

### 2. Semantic Markers for Disambiguation

Use explicit semantic tokens (like `.replies.`) to distinguish subject types, rather than relying on part counts alone:

```
# Good: Clear semantic marker
msg.{rootId}.replies.{eventId}   # "replies" explicitly marks thread messages

# Less clear: Only part count differs
msg.{eventId}                    # Root (6 parts)
thread.{rootId}.{eventId}        # Thread (7 parts)
```

### 3. Hierarchical Nesting

Structure subjects so children nest under parents in the namespace:

```
# Good: Threads nest under their root message
msg.{rootId}.replies.{eventId}

# Less intuitive: Separate top-level namespace
thread.{rootId}.{eventId}
```

## Filtering Patterns Reference

For room messages, these wildcard patterns enable efficient filtering:

| Pattern | Matches |
|---------|---------|
| `msg.>` | All messages (root + threads) |
| `msg.*` | Root messages only |
| `msg.*.replies.>` | All thread replies (any thread) |
| `msg.{rootId}.replies.>` | Replies in a specific thread |
| `msg.*.replies.{eventId}` | Lookup thread reply by event ID |

## Subject Refactoring Checklist

When changing subject patterns:

1. **Update construction functions** in `subjects.go` (e.g., `SpaceRoomThread`)
2. **Update parsing functions** in `subjects.go` (e.g., `IsThreadSubject`, `ParseEventIDFromSubject`)
3. **Update all test expectations** in `subjects_test.go`
4. **Update comments** in files that reference the patterns (e.g., `rooms.go`)
5. **Update `docs/ARCHITECTURE.md`** subject tables and filtering examples
6. **Run full test suite** including e2e tests - subject changes cascade through the entire system

Subject changes are high-risk because they affect:
- JetStream stream configs and filters
- Consumer subscriptions
- `GetLastMsgForSubject` lookups
- Event routing and delivery
