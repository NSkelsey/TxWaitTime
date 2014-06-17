DROP TYPE IF EXISTS kinds CASCADE;
DROP TABLE IF EXISTS relations CASCADE;
DROP TABLE IF EXISTS txs;
DROP TABLE IF EXISTS blocks;

CREATE TYPE kinds AS ENUM (
    'p2pkh', 'multisig', 'nulldata', 'p2sh', 'pubkey', 'bulletin', 'unknown'
);

CREATE TABLE txs (
    txid        bytea primary key,
    kind        kinds,
    firstseen   timestamp,
    size        integer,
    extra       boolean,
    priority    float8,
    fee         integer
);

CREATE TABLE blocks (
    hash        bytea primary key,
    firstseen   timestamp
);

CREATE TABLE relations (
    txid        bytea references txs(txid),
    block       bytea references blocks(hash)
);

