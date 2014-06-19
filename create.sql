DROP TYPE IF EXISTS kinds CASCADE;
DROP TABLE IF EXISTS relations CASCADE;
DROP TABLE IF EXISTS tx_features;
DROP TABLE IF EXISTS txs;
DROP TABLE IF EXISTS blocks;


CREATE TYPE kinds AS ENUM (
    'pubkeyhash', 'multisig', 'nulldata', 'scripthash', 'pubkey', 'bulletin', 'nonstandard'
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

CREATE TABLE tx_features (
    txid        bytea primary key,
    nonstandard int,
    pubkey      int,
    pubkeyhash  int,
    scripthash  int,
    multisig    int,
    nulldata    int
);

CREATE TABLE blocks (
    hash        bytea primary key,
    firstseen   timestamp
);

CREATE TABLE relations (
    txid        bytea references txs(txid),
    block       bytea references blocks(hash)
);



CREATE OR REPLACE FUNCTION kind(bytea) RETURNS kinds AS $$
DECLARE
ns int; nd int; ms int; sh int; pk int;
BEGIN
    SELECT nonstandard, nulldata, multisig, scripthash, pubkey INTO ns, nd, ms, sh, pk 
    FROM tx_features WHERE txid = $1;
    IF ns > 0 THEN
        RETURN "nonstandard";
    ELSEIF nd > 0 THEN
        RETURN "nulldata";
    ELSEIF ms > 0 THEN
        RETURN "multisig";
    ELSEIF sh > 0 THEN
        RETURN "scripthash";
    ELSEIF pk > 0 THEN
        RETURN "pubkey";
    ELSE 
        RETURN "pubkeyhash";
    END IF;
END;
$$ LANGUAGE plpgsql;


