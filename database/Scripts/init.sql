create table links
(
    id         integer primary key autoincrement,
    short_url  varchar(16) not null unique,
    long_url   text        not null,
    created_at integer     not null
);
create unique index links_short_url ON links (short_url);

create table link_accesses
(
    id          integer primary key autoincrement,
    link_id     int     not null,
    access_time integer not null default current_timestamp,
    constraint fk_link_accesses_link_id foreign key (link_id) references links (id)
);