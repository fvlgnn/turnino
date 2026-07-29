# Queue numbering

Turnino supports numeric queues, queues with a fixed service prefix, and progressive alphanumeric ticket rolls. Every queue has its own numbering policy and current position.

## Numeric queues

A numeric queue has no prefix. The administration interface provides two common presets:

```text
00, 01, 02, ..., 99, 00
000, 001, 002, ..., 999, 000
```

These policies are suitable for simple paper dispensers and installations where one numeric sequence is sufficient.

## Fixed-prefix queues

A fixed prefix identifies a service or queue but does not advance:

```text
policy: prefix A, range 00-99
A00, A01, ..., A99, A00
```

For example, one installation could use `A00-A99` for general services and `B00-B99` for another service. The prefix belongs to the Turnino queue configuration; it is not a progressive section of a physical ticket roll.

The prefix field is therefore shown for numeric presets and custom fixed-prefix policies. It is hidden for predefined progressive-roll policies to avoid confusing the two concepts.

## Progressive alphanumeric queues

Progressive policies advance the alphabetic part after the numeric range reaches its end:

```text
A00, A01, ..., A99, B00, ..., E99, A00
```

The built-in presets cover:

- `A00-E99`;
- `A00-J99`.

The custom policy supports:

- one or two ASCII letters from `A` to `Z`;
- independently selected alphabetic start and end values;
- one to six numeric digits;
- a custom numeric minimum and maximum.

Examples include:

```text
A00-Z99
C01-H50
AA00-AZ99
BA000-BZ999
```

The alphabetic endpoints must have the same width and the start must precede or equal the end. Numeric values must fit within the configured digit count.

## Queue movement

All policies use wrap:

- **Call next** advances one position and calls it;
- **Call previous** moves back one position and calls it;
- **Recall last** repeats the last call made by that counter without changing the queue position;
- **Reset queue** returns the queue to the "not started" state without deleting call history or producing a call event.

After a reset, the next call selects the first configured number. Moving forwards past the last number wraps to the first; moving backwards from the first wraps to the last.

## Custom-number calls

Custom calls are intended for queues backed by a manual paper dispenser. They must be enabled explicitly for each queue.

The operator enters the complete visible number, such as `A37` or `BC42`. Turnino validates it against the queue policy, records the call and moves the queue to that position. The next automatic call then continues from the following position.

Keep custom calls disabled when ticket numbers are issued automatically by another system.

## Editing an existing policy

Queue policies may be corrected without recreating the queue:

- if the current formatted number is valid under the new policy, Turnino preserves it;
- if it is incompatible, the administration interface requires explicit reset confirmation;
- existing call history remains unchanged because each call stores the number that was actually displayed.

This makes it possible to adapt a queue when a supplier provides a different ticket-roll range while keeping past calls readable.
