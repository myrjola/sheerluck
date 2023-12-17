create table if not exists sessions
(
    token  text primary key,
    data   blob not null,
    expiry real not null
) strict;

create index if not exists sessions_expiry_idx on sessions (expiry);

create table if not exists users
(
    id           blob primary key,
    display_name text not null,

    created      text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
    updated      text not null default (strftime('%Y-%m-%dT%H:%M:%fZ'))
) strict;

create trigger if not exists users_updated_timestamp
    after update
    on users
begin
    update users set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
end;

create table if not exists credentials
(
    id                          blob primary key,
    public_key                  blob    not null,
    attestation_type            text    not null,
    transport                   text    not null,
    flag_user_present           integer not null,
    flag_user_verified          integer not null,
    flag_backup_eligible        integer not null,
    flag_backup_state           integer not null,
    authenticator_aaguid        blob    not null,
    authenticator_sign_count    integer not null,
    authenticator_clone_warning integer not null,
    authenticator_attachment    text    not null,

    created                     text    not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
    updated                     text    not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),

    user_id                     blob    not null references users (id) on delete cascade
) strict;

create trigger if not exists credentials_updated_timestamp
    after update
    on credentials
begin
    update credentials set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
end;
