-- The app schema every table lives in, and the trigger function every mutable table shares.
-- Attach it as a before update trigger and updated_at stamps itself, so no query sets it by hand:
--
--   create trigger set_updated_at before update on app.account
--       for each row execute function app.set_updated_at();
--
-- Append-only tables have no updated_at and skip the trigger.

-- +goose Up
create schema app;

-- +goose StatementBegin
create function app.set_updated_at()
returns trigger
language plpgsql
as $$
begin
    new.updated_at = now();
    return new;
end;
$$;
-- +goose StatementEnd

-- +goose Down
drop function app.set_updated_at();
drop schema app;
