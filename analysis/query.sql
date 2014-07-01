-- All Statistics are computed based on 'lastweek'
CREATE OR REPLACE VIEW lastweek AS
SELECT txs.*
    FROM txs WHERE txs.firstseen > current_timestamp - interval '7 days'
;


CREATE OR REPLACE VIEW mempool AS
SELECT txs.* 
    FROM txs LEFT JOIN relations ON
    relations.txid = txs.txid
    WHERE txs.firstseen > current_timestamp - interval '72 hours'
    AND relations.txid is null
;

-- Transactions forgotten within the current week
-- Anything that pops up in here was censored by the network
-- for some reason
CREATE OR REPLACE VIEW forgotten AS
SELECT txs.*
    FROM txs LEFT JOIN relations ON
    relations.txid = txs.txid
    WHERE txs.firstseen < current_timestamp - interval '72 hours'
    AND txs.firstseen > current_timestamp - interval '7 days'
    AND relations.txid is null
;

-- All transactions found in blocks this week
CREATE OR REPLACE VIEW included AS
SELECT t.*, 
       blocks.firstseen - t.firstseen AS conf_time
    FROM (
        SELECT lastweek.*, relations.block
        FROM lastweek INNER JOIN relations ON
        relations.txid = lastweek.txid
    ) AS t INNER JOIN blocks ON
    t.block = blocks.hash
;

CREATE OR REPLACE VIEW unseen AS
SELECT * FROM included 
    WHERE conf_time < INTERVAL '5 microseconds';
    
-- All transactions that are confirmed
CREATE OR REPLACE VIEW confirmed AS
SELECT * FROM included
    WHERE conf_time > INTERVAL '5 microseconds';

-- avg confirmation time for the last week
CREATE OR REPLACE VIEW avg_conf_time AS
SELECT included.kind, 
    EXTRACT(EPOCH FROM avg(included.conf_time)) as avg,
    count(*) as confirmed
    FROM blocks INNER JOIN included ON 
    (included.block = blocks.hash)
    GROUP BY included.kind
;

-- Computes the average fee from the 'extra' field
CREATE OR REPLACE VIEW avg_fee AS
SELECT kind, avg(included.fee) 
    FROM included
    WHERE included.extra
    GROUP BY kind
;

-- Summary statistics
CREATE OR REPLACE VIEW summary AS
SELECT COALESCE(avg_conf_time.kind, avg_fee.kind) as kind,
    avg_conf_time.avg as conf_time,
    avg_fee.avg as fee
    FROM avg_conf_time FULL JOIN avg_fee ON
    (avg_fee.kind = avg_conf_time.kind)
;


-- transaction confirmation rates for the week
CREATE OR REPLACE VIEW conf_rates AS
WITH confirmed AS (
    SELECT confirmed.kind, count(*) AS num
        FROM confirmed
        GROUP BY kind
),  unseen AS (
    SELECT unseen.kind, count(*) AS num
        FROM unseen
        GROUP BY kind
),  mempool AS (
    SELECT kind, count(*) AS num
        FROM mempool 
        GROUP BY kind
),  forgot AS (
    SELECT kind, count(*) AS num
        FROM forgotten
        GROUP BY kind
)
SELECT COALESCE(confirmed.kind, unseen.kind, mempool.kind, forgot.kind) as kind,
       forgot.num as forgotten, 
       mempool.num as mempool, 
       unseen.num as unseen, 
       confirmed.num as confirmed 
    FROM (
        confirmed FULL JOIN unseen ON (confirmed.kind = unseen.kind)
        FULL JOIN mempool ON (confirmed.kind = mempool.kind)
        FULL JOIN forgot ON (confirmed.kind = forgot.kind)
    )
;


-- computes the max confirmation time for a given kind
CREATE OR REPLACE FUNCTION max_conftime(kinds) RETURNS float AS $$
DECLARE
max interval;
BEGIN
    SELECT max(included.conf_time) into max
        FROM included
        WHERE included.kind = $1
    ;
    RETURN extract(epoch FROM max);
END;
$$ LANGUAGE plpgsql;

DROP TYPE col_rows CASCADE;

CREATE TYPE col_rows AS (kind kinds, col int, count bigint);

-- Returns a histogram of transaction types
CREATE OR REPLACE FUNCTION conf_time_histogram(targ kinds) RETURNS setof col_rows AS $$
DECLARE
max float;
BEGIN
    max = max_conftime(targ);
    RETURN QUERY SELECT 
               confirmed.kind,
               width_bucket(
                  extract(epoch FROM confirmed.conf_time),
                  0,
                  max,
                  50
               ) AS col,
               count(*)
        FROM confirmed
        WHERE confirmed.kind = targ
        GROUP BY kind, col
        ORDER BY kind, col;
END;
$$ language 'plpgsql'
;

DROP TYPE ct_sum CASCADE;

CREATE TYPE ct_sum AS (kind kinds, max float, leftout bigint, intail bigint, rightout bigint);

-- Returns a set of rows summarizing needed stats for our histograms
CREATE OR REPLACE FUNCTION conf_time_stats(targ kinds) RETURNS ct_sum AS $$
DECLARE
max float; biggest interval; 
intail bigint; rightout bigint; leftout bigint;
BEGIN
    max = max_conftime(targ);

    SELECT count(*) INTO leftout FROM unseen WHERE kind = targ;

    SELECT count(*) INTO rightout FROM mempool 
        FULL JOIN forgotten ON (mempool.txid = forgotten.txid) 
        WHERE mempool.kind = targ OR forgotten.kind = targ;

    biggest = INTERVAL '1 second' *  max;

    SELECT count(*) INTO intail FROM confirmed WHERE 
        (kind = targ AND conf_time > biggest);

    RETURN (targ, max, leftout, intail, rightout);
END;
$$ language 'plpgsql'
;

-- Execute external views
--SELECT * FROM avg_conf_time;
--SELECT * FROM conf_rates;
--SELECT * FROM conf_time_histogram('nonstandard');
SELECT * FROM conf_time_stats('pubkeyhash');
