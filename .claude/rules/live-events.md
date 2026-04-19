# Live Events

## Adding New Live Event Types

When adding a new live event type (published via `publishLiveSpaceEvent`), you must update **all** of these locations or the event will be silently dropped:

### Backend Checklist

1. **Proto definition** — Add the event message to `proto/chatto/core/v1/live_event.proto` and the oneof case to `space_event.proto`
2. **Proto interface method** — Add `IsSpaceEventType()` to `cli/internal/pb/chatto/core/v1/graphql.go` so gqlgen recognizes the proto type
3. **Event unwrapper** — Add a case in `unwrapSpaceEvent()` in `cli/internal/graph/event_helpers.go`
4. **Subscription room ID extraction** — Add a case to the `liveRoomMsgChan` switch in `StreamMySpaceEvents` (`cli/internal/core/core.go` ~line 1758) that extracts the `roomID` from the event. **Without this, the event is silently dropped** because `roomID` stays empty and the `if roomID == ""` guard skips it.
5. **GraphQL schema** — Add the type definition and include it in the `SpaceEventType` union in `events.graphqls`

### Frontend Checklist

6. **GraphQL fragment** — Add the event fields to the subscription fragment in `RoomEvent.svelte`
7. **Event handler** — Handle the event in `RoomEventsPane.svelte` (and `ThreadPane.svelte` if applicable)

### Common Pitfall: Room ID Extraction

The subscription handler in `core.go` routes live room events by extracting the `roomID` from each event type via a type switch. If your new event type isn't in that switch, the room ID will be empty and the event will be silently dropped with `continue`. There is no error log — it just disappears. Always check this switch when adding live events.

### Common Pitfall: Race Conditions with KV State

If your event relies on KV state that's set alongside the event (e.g., setting a processing status in RUNTIME KV and then publishing an event), make sure the KV write happens **before** the action that triggers the subscription event. The subscription delivers events immediately, and field resolvers that read KV will see stale/missing data if the write hasn't happened yet.

Example: Video processing sets PENDING state in KV *before* `PostMessage` publishes to JetStream, so that when the subscription resolves `Attachment.videoProcessing`, the KV entry already exists.
