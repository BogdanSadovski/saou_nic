# WebSocket Events v1

Envelope:

```json
{
  "version": "v1",
  "type": "session.started",
  "timestamp": "2026-04-08T12:00:00.000Z",
  "payload": {}
}
```

Event types:
- `session.started`
- `message.user`
- `ai.typing.started`
- `ai.message.chunk`
- `ai.typing.stopped`
- `message.ai`
- `session.warning`
- `session.finished`
- `report.ready`

Compatibility rules:
- New optional fields can be added to `payload`.
- Event `type` names are stable in minor versions.
- Breaking changes require `version` bump.
