-- Two fixed member slots make the two-player limit hold even when joins race.
-- Codes use the same six-digit rule as handlers/combo's validator.
-- +goose Up
create table app.combo (
    combo_id bigint generated always as identity primary key,
    code text not null check (code ~ '^[0-9]{6}$'),
    host_id bigint not null references app.account (account_id) on delete cascade,
    state text not null default 'open' check (state in ('open', 'ended')),
    expires_at timestamptz not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    check (expires_at = created_at + interval '2 hours')
);
create unique index combo_code_key on app.combo (code) where state = 'open';
create index combo_code_created_at_idx on app.combo (code, created_at desc, combo_id desc);
create trigger set_updated_at before update on app.combo
    for each row execute function app.set_updated_at();

create table app.combo_member (
    combo_member_id bigint generated always as identity primary key,
    combo_id bigint not null references app.combo (combo_id) on delete cascade,
    account_id bigint not null references app.account (account_id) on delete cascade,
    slot int not null check (slot in (1, 2)),
    created_at timestamptz not null default now(),
    constraint combo_member_slot_key unique (combo_id, slot),
    constraint combo_member_account_key unique (combo_id, account_id)
);
create index combo_member_account_id_idx on app.combo_member (account_id, combo_id);

alter table app.scan add column combo_id bigint references app.combo (combo_id) on delete set null;
create index scan_combo_id_account_id_idx on app.scan
    (combo_id, account_id, created_at desc, scan_id desc) where combo_id is not null;

-- +goose Down
drop index app.scan_combo_id_account_id_idx;
alter table app.scan drop column combo_id;
drop table app.combo_member;
drop table app.combo;
