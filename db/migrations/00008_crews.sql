-- Crews: private boards for friend groups, joined by a six-digit code. A crew is its name, its code
-- and the player who made it; app.crew_member says who is in it. Deleting an account takes its
-- memberships and the crews it made with it, as every table on app.account cascades
-- (docs/contracts.md). A crew also ends when its last member goes, however they go. The name bound
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

-- The last member out ends the crew, whether they left or their account was deleted (once the
-- owner has left, nothing else would), so no crew is left with no one to see it and its code goes
-- free. It lives here, beside the rows it guards, so leaving and deleting an account cannot follow
-- different rules.
-- +goose StatementBegin
create function app.end_empty_crew() returns trigger
language plpgsql as $$
begin
    delete from app.crew c
     where c.crew_id = old.crew_id
       and not exists (select 1 from app.crew_member m where m.crew_id = old.crew_id);
    return null;
end;
$$;
-- +goose StatementEnd

create trigger crew_member_end_empty_crew
    after delete on app.crew_member
    for each row execute function app.end_empty_crew();

-- +goose Down
drop table app.crew_member;
drop function app.end_empty_crew();
drop table app.crew;
