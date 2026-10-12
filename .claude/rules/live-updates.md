---
paths:
  - "packages/go/poke/**/*.go"
  - "packages/go/handlers/live/**/*.go"
  - "packages/go/repository/**/*.go"
  - "proto/live/**/*.proto"
  - "packages/ts/api/src/live*.ts"
  - "packages/ts/api/src/hooks/**/*.ts"
---

# Live updates: poke, then pull

The server never pushes data. When a write lands it emits a hint (a topic, sometimes an id), each
interested client invalidates the matching queries, and the refetch goes through the ordinary
RPCs. One read path means the owner fence and redaction are enforced in one place, and a lost hint
costs a refetch, never stale data.

This is how every screen stays current: the Combo lobby filling up, the other player's scan
landing, the combined verdict revealed on both phones. No polling loops and no payload pushes.

## Emitting (Go)

- One NOTIFY channel, `app_events`. `poke.Emit(ctx, db, p)` runs `select pg_notify($1, $2)` with a
  JSON payload.
- **The repository emits, beside the statement the hint describes.** Inside a transaction
  Postgres queues the NOTIFY until commit and drops it on rollback, so a hint fires if and only if
  the write landed. Never emit from a handler after the fact, and never for a write that can still
  roll back.

```go
func (r *Repository) EndCombo(ctx context.Context, comboID int64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		update app.combo
		   set state = 'ended'
		 where combo_id = $1
		   and state = 'open'`, comboID)
	if err != nil {
		return false, known(fmt.Sprintf("end combo combo=%d", comboID), err)
	}
	ended := tag.RowsAffected() > 0
	if ended {
		r.notify(ctx, comboPoke(comboID))
	}
	return ended, nil
}
```

- `r.notify` drops the `Emit` error on purpose and its comment says why: a hint is an accelerator,
  the pull is the truth, and the listener resyncs everyone on reconnect.
- Pokes come from small named helpers in `repository.go` (`comboPoke(comboID)`), so a write names
  the family it staled instead of assembling a struct inline. A write that changed nothing emits
  nothing.
- **Every write that changes something a client reads emits.** A missing emit, or one with the
  wrong topic or scope, fails silently: the write succeeds and some screen never updates. No test
  catches it, so review must.
- Scope, most specific first: an account id reaches only that player's streams; a combo id
  reaches both players' streams; neither reaches everyone (that is how a resync travels). A
  per-entity verb carries the entity id to keep the blast radius small.
- `poke.Topic` is a hand-rolled Go type mirroring the `live.v1.Topic` enum, kept in step by hand.
  `handlers/live` is the one place that maps between them, because proto types stay at the wire.
- Add a topic only for a family that moves on a different clock from every existing one.
  Otherwise ride an existing topic and say why in a comment.

## Delivering (Go)

- One `poke.Listener` per process on a dedicated connection. Its wait is bounded and every timeout
  pings, because `WaitForNotification` blocks forever on a silently dead TCP link. It publishes a
  resync after every reconnect.
- `poke.Hub` gives each stream a mailbox of 16; overflow collapses it to one resync. At most 32
  streams per caller (`ErrTooManySubscribers`, mapped to `ResourceExhausted`). `Publish` never
  blocks and takes the write lock, never `RLock`: delivering drains and refills a mailbox, which
  is only safe one goroutine at a time. The hub never closes a subscriber's channel.
- `LiveService.Watch`: authenticate once at open, `Subscribe` BEFORE sending the hello (an empty
  message), heartbeat every 25 s (under the 60 s proxy timeout), retire the stream after 15
  minutes so revoked access converges, return `nil` when `Send` fails (a closed tab is the normal
  end), and return when the server's `done` channel closes so a deploy drains. The session is
  read-only inside the stream.
- Every failure degrades to "refetch more than needed", never to "keep stale data".

## Applying (TypeScript)

- `useLiveInvalidation()` is mounted ONCE, in the layout that wraps the signed-in screens.
- **Each query family has one shared invalidator** (`useInvalidateCombos()` in its hooks file).
  Mutations call it in `onSettled` and the live map calls the same function. Never keep a second
  list of what a topic stales: two lists drift, and the symptom is a screen that is quietly stale.
  A mutation that needs more widens the shared invalidator.
- The topic switch names every value and ends in a `never` check, so a new topic with no arm is a
  compile error, while an unknown topic from a newer server still invalidates everything:

```ts
switch (event.topic) {
  case Topic.UNSPECIFIED:
    return; // heartbeat
  case Topic.RESYNC:
    inv.everything();
    return;
  case Topic.COMBO:
    inv.combos();
    return;
  default: {
    const unhandled: never = event.topic;
    void unhandled;
    inv.everything(); // a topic from a newer server: coarse, never wrong
  }
}
```

- The hello (first message) invalidates everything: the catch-up rule. The server subscribed
  before sending it, so no hint falls between the refetch and the subscription.
- Reconnect with exponential backoff and jitter, capped at 30 s, reset only after a connection
  stayed healthy for 60 s. The ONLY stop signal is `signal.aborted`; never inspect the error code,
  because connect-es maps every AbortError to `Canceled`.
- The stream needs a fetch that can read a response body as it arrives, so the app passes
  `expo/fetch` into the transport (`ts-api.md`). Each app instance streams for itself.
- Prefer plain invalidation over optimistic patches. A mutation that does patch in `onMutate`
  pairs `suppressNextPoke(topic, id)` there with `settleTopic(queryClient, topic, id)` in
  `onSettled`, never a hand-picked subset of the arm.
- An infinite query is invalidated with `cardinality: undefined`, or the infinite key is missed.
