-- A plate is a photo of a cooked recipe that the model checked, and the points ledger is what it
-- earned: one row per award, so a player's points are a sum and never a column. Plate photos are
-- kept, because they fill the profile. The same photo never counts twice, for anyone, and each
-- player earns one cook credit per meal window, which the partial unique index below decides.
--
-- The checks are restated in packages/go/handlers/cook (the photo's size) and packages/go/points
-- (the reasons and the windows): keep them in step. Rows are written once and never updated.

-- +goose Up
create table app.plate (
    plate_id     bigint      generated always as identity primary key,
    account_id   bigint      not null references app.account (account_id) on delete cascade,
    -- Checked at the end of the statement, so deleting an account takes its plates and its
    -- recipes together, while a plated recipe never goes on its own.
    recipe_id    bigint      not null references app.recipe (recipe_id) on delete no action,
    -- A JPEG of at most 2 MB (the photo tag).
    photo        bytea       not null check (octet_length(photo) between 1 and 2097152),
    photo_sha256 bytea       not null check (octet_length(photo_sha256) = 32),
    -- When it was plated, by the server's clock: the late-night badge reads its hour.
    created_at   timestamptz not null default now(),
    -- A photo seen before is this constraint firing, never a check before the insert.
    constraint plate_photo_sha256_key unique (photo_sha256)
);

create index plate_account_id_created_at_idx on app.plate (account_id, created_at desc);

create table app.point_event (
    point_event_id bigint      generated always as identity primary key,
    account_id     bigint      not null references app.account (account_id) on delete cascade,
    plate_id       bigint      not null references app.plate (plate_id) on delete cascade,
    reason         text        not null
                               check (reason in ('cook', 'scrappy_bonus', 'against_all_odds',
                                                 'streak', 'combo_cook')),
    points         int         not null check (points > 0),
    -- The plate's meal window: its campus day and window, as '2026-10-21 late_night'.
    meal_window    text        not null
                               check (meal_window ~ '^\d{4}-\d{2}-\d{2} (morning|midday|evening|late_night)$'),
    created_at     timestamptz not null default now()
);

-- One cook credit per player per meal window. A second plate in the window is this index firing.
create unique index point_event_account_id_meal_window_key
    on app.point_event (account_id, meal_window)
    where reason = 'cook';

-- +goose Down
drop table app.point_event;
drop table app.plate;
