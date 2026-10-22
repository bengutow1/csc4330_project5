-- A scan is one judging of a player's kitchen: the score our code worked out from the model's
-- list, the model's roast, and the list itself in app.scan_item. The score is set once, when the
-- scan is made, and editing the list never changes it. The tier comes from the score on every
-- read and is never a column. items_version counts the list's versions, so the recipes (#21) know
-- when to regenerate. The photos are never stored.
--
-- The item checks are restated as validator tags in packages/go/handlers/scan/validate.go: keep
-- the two in step. Each category is spelled as docs/contracts.md spells it.

-- +goose Up
create table app.scan (
    scan_id       bigint      generated always as identity primary key,
    account_id    bigint      not null references app.account (account_id) on delete cascade,
    score         int         not null check (score between 0 and 100),
    -- The model's line about the fridge, never the person.
    roast         text        not null,
    -- 1 for the model's list, and one more for each edit.
    items_version int         not null default 1 check (items_version >= 1),
    created_at    timestamptz not null default now(),
    updated_at    timestamptz not null default now()
);

-- A player's newest scan first, for GetLastScan.
create index scan_account_id_created_at_idx on app.scan (account_id, created_at desc, scan_id desc);

create trigger set_updated_at before update on app.scan
    for each row execute function app.set_updated_at();

-- An edit replaces a scan's items rather than updating them, so this table has no updated_at.
create table app.scan_item (
    scan_item_id bigint      generated always as identity primary key,
    scan_id      bigint      not null references app.scan (scan_id) on delete cascade,
    -- 1 to 40 characters, trimmed before it is stored.
    name         text        not null check (char_length(name) between 1 and 40),
    category     text        not null
                             check (category in ('protein', 'vegetable', 'fruit', 'grain', 'dairy',
                                                 'condiment', 'drink', 'snack', 'leftover',
                                                 'other')),
    -- Free text such as 'x4' or '1 bag', and '' when unknown. It never counts toward the score.
    quantity     text        not null check (char_length(quantity) <= 20),
    -- The display order, from 1.
    position     int         not null check (position >= 1),
    created_at   timestamptz not null default now(),
    constraint scan_item_scan_id_position_key unique (scan_id, position)
);

-- +goose Down
drop table app.scan_item;
drop table app.scan;
