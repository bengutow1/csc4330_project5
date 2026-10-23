-- A recipe is one dish the model suggested for one version of a scan's list, stored only after
-- packages/go/recipes held it to that list, the player's staples and their allergies. Each version
-- gets its set once, recorded in app.recipe_set even when every recipe was dropped, so an empty
-- set is never asked for again: editing the list bumps the scan's items_version, and the next
-- ListRecipes makes a new set. Older sets stay, because a plate (#23) names the recipe it cooked.
-- Rows are written once and never updated, so neither table has an updated_at.
--
-- These checks are restated as validator tags in packages/go/handlers/cook/validate.go, which
-- also bounds each name and step: keep the two in step. Each closed set is spelled as
-- docs/contracts.md spells it.

-- +goose Up
-- One row per version of a scan's list whose set was made, empty or not.
create table app.recipe_set (
    recipe_set_id bigint      generated always as identity primary key,
    scan_id       bigint      not null references app.scan (scan_id) on delete cascade,
    items_version int         not null check (items_version >= 1),
    -- Snapshot identity, not a live relationship: old dishes survive edits or removal.
    -- Both zero for solo sets. A combo set is cached against both list versions.
    partner_scan_id bigint not null default 0,
    partner_items_version int not null default 0,
    combo_score int check (combo_score between 0 and 100),
    check ((partner_scan_id = 0 and partner_items_version = 0) or
           (partner_scan_id > 0 and partner_items_version > 0)),
    created_at    timestamptz not null default now(),
    -- One set per version: a second ListRecipes that raced the first is this constraint firing,
    -- never a check before the insert.
    constraint recipe_set_scan_id_items_version_key unique
        (scan_id, items_version, partner_scan_id, partner_items_version)
);

create table app.recipe (
    recipe_id     bigint      generated always as identity primary key,
    recipe_set_id bigint not null references app.recipe_set (recipe_set_id) on delete cascade,
    scan_id       bigint      not null references app.scan (scan_id) on delete cascade,
    -- The version of the scan's list the recipe was made from.
    items_version int         not null check (items_version >= 1),
    -- The model's order within its set, from 1.
    position      int         not null check (position >= 1),
    title         text        not null check (char_length(title) between 1 and 80),
    -- The whole time to the plate.
    minutes       int         not null check (minutes between 1 and 240),
    difficulty    text        not null check (difficulty in ('easy', 'medium', 'hard')),
    -- What it cooks with; empty needs none.
    equipment     text[]      not null
                              check (equipment <@ array['microwave', 'stovetop', 'oven', 'air_fryer']),
    -- The food it takes that the player has: names from the list and the staples.
    uses          text[]      not null check (cardinality(uses) between 1 and 20),
    -- The food it takes that the player lacks: none, or the one a One item away recipe needs.
    missing       text[]      not null check (cardinality(missing) <= 1),
    -- Every allergen it contains: what the model declared and what its food gives away.
    allergens     text[]      not null
                              check (allergens <@ array['peanuts', 'tree_nuts', 'milk', 'eggs',
                                                        'wheat', 'soy', 'fish', 'shellfish',
                                                        'sesame']),
    -- The diets besides none that it fits, as packages/go/recipes judged from its food. The row
    -- never changes, so the judgment cannot drift from what it uses.
    diets         text[]      not null
                              check (diets <@ array['vegetarian', 'vegan', 'pescatarian']),
    steps         text[]      not null check (cardinality(steps) between 1 and 20),
    -- 0 is untimed. Empty is a legacy recipe with no timers; otherwise one per step.
    -- Keep these bounds in step with cook/validate.go.
    step_timers   int[]       not null default '{}'
                              check ((cardinality(step_timers) = 0 or
                                      cardinality(step_timers) = cardinality(steps)) and
                                     array_position(step_timers, null) is null and
                                     0 <= all(step_timers) and 14400 >= all(step_timers)),
    -- Provenance is kept with the dish, so later list edits cannot change the bring list.
    ingredient_owners jsonb not null default '[]' check (jsonb_typeof(ingredient_owners) = 'array'),
    only_together boolean not null default false,
    created_at    timestamptz not null default now(),
    -- Every recipe belongs to a set that was made.
    constraint recipe_scan_id_items_version_position_key unique (recipe_set_id, position)
);

create index recipe_scan_id_items_version_idx on app.recipe (scan_id, items_version);

-- +goose Down
drop table app.recipe;
drop table app.recipe_set;
