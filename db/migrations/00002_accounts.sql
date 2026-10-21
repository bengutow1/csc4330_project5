-- One row per device account, and the sessions that carry it. The handle is the player's public
-- name on the board, and the kitchen settings shape every roast and recipe. Each closed set is
-- spelled as docs/contracts.md spells it. The handle and avatar checks are restated as validator
-- tags in packages/go/handlers/validate.go: keep the two in step.
--
-- sessions is pgxstore's table exactly as its README defines it, in the default schema, because
-- that is where pgxstore looks. Only scs reads and writes it.

-- +goose Up
create table app.account (
    account_id  bigint      generated always as identity primary key,
    -- 3 to 20 lowercase letters, digits and underscores (the handle tag).
    handle      text        not null check (handle ~ '^[a-z0-9_]{3,20}$'),
    -- One emoji, so 1 to 10 code points: the longest, a kiss with two skin tones, is 10 (the
    -- emoji tag).
    avatar      text        not null check (char_length(avatar) between 1 and 10),
    roast_level text        not null default 'medium'
                            check (roast_level in ('mild', 'medium', 'extra_crispy')),
    equipment   text[]      not null default '{}'
                            check (equipment <@ array['microwave', 'stovetop', 'oven', 'air_fryer']),
    diet        text        not null default 'none'
                            check (diet in ('none', 'vegetarian', 'vegan', 'pescatarian')),
    allergies   text[]      not null default '{}'
                            check (allergies <@ array['peanuts', 'tree_nuts', 'milk', 'eggs',
                                                      'wheat', 'soy', 'fish', 'shellfish',
                                                      'sesame']),
    -- Every staple is on until the player turns one off.
    staples     text[]      not null default array['salt', 'pepper', 'oil', 'butter', 'rice',
                                                   'pasta', 'flour', 'sugar', 'spices']
                            check (staples <@ array['salt', 'pepper', 'oil', 'butter', 'rice',
                                                    'pasta', 'flour', 'sugar', 'spices']),
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now(),
    -- A taken handle is this constraint firing, never a check before the insert.
    constraint account_handle_key unique (handle)
);

create trigger set_updated_at before update on app.account
    for each row execute function app.set_updated_at();

create table sessions (
    token  text        primary key,
    data   bytea       not null,
    expiry timestamptz not null
);

create index sessions_expiry_idx on sessions (expiry);

-- +goose Down
drop table sessions;
drop table app.account;
