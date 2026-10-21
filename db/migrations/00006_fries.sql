-- One row per fry run: a trip a player logged from Fries mode. A fry run earns no points; the Hall
-- of Fries counts them. The spot belongs to the places provider and nothing else in the schema
-- knows it, so the row keeps its id and name as the app sent them. The spot_id and spot_name
-- bounds are restated as validator tags in packages/go/handlers/fries/validate.go: keep the two
-- in step.

-- +goose Up
create table app.fry_run (
    fry_run_id bigint      generated always as identity primary key,
    account_id bigint      not null references app.account (account_id) on delete cascade,
    -- The provider's own id for the spot, opaque to us (the spot_id tag).
    spot_id    text        not null check (char_length(spot_id) between 1 and 256),
    -- The spot's name when the run was logged (the spot_name tag).
    spot_name  text        not null check (char_length(spot_name) between 1 and 120),
    created_at timestamptz not null default now()
);

-- A player's runs, newest last: the Hall of Fries counts them over a week, and deleting an
-- account finds them through it.
create index fry_run_account_id_created_at_idx on app.fry_run (account_id, created_at);

-- +goose Down
drop table app.fry_run;
