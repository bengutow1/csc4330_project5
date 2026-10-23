-- Crews: private boards for friend groups, joined by a six-digit code. A crew is its name, its code
-- and the player who made it; app.crew_member says who is in it. Deleting an account takes its
-- memberships with it, and the crews it made, as every table on app.account does. The name bound
-- is restated as a validator tag in packages/go/handlers/crew/validate.go: keep the two in step.

-- +goose Up
create table app.crew (
    crew_id    bigint      generated always as identity primary key,
    -- What the members call it (the name tag).
    name       text        not null check (char_length(name) between 1 and 30),
    -- Six digits, never expiring, and one crew's alone. A collision is this constraint firing, and
    -- the handler draws a fresh code.
    code       text        not null check (code ~ '^[0-9]{6}$'),
    owner_id   bigint      not null references app.account (account_id) on delete cascade,
    created_at timestamptz not null default now(),
    constraint crew_code_key unique (code)
);

-- The crews an owner made, so deleting their account finds them.
create index crew_owner_id_idx on app.crew (owner_id);

create table app.crew_member (
    crew_id    bigint      not null references app.crew (crew_id) on delete cascade,
    account_id bigint      not null references app.account (account_id) on delete cascade,
    -- When they joined: a player's crews list the one joined first first.
    created_at timestamptz not null default now(),
    primary key (crew_id, account_id)
);

-- A player's crews, for their list and the 10-crew cap.
create index crew_member_account_id_idx on app.crew_member (account_id, created_at);

-- +goose Down
drop table app.crew_member;
drop table app.crew;
