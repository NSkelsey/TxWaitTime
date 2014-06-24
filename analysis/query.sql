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

-- avg confirmation time for the last week
CREATE OR REPLACE VIEW avg_conf_time AS
SELECT included.kind, avg(included.conf_time) 
    FROM blocks INNER JOIN included ON 
    (included.block = blocks.hash)
    GROUP BY included.kind
;

-- transaction confirmation rates for the week
CREATE OR REPLACE VIEW conf_rates AS
WITH included AS (
    SELECT included.kind, count(*) AS confirmed
        FROM included
        GROUP BY included.kind
),  mempool AS (
    SELECT kind, count(*) AS total
        FROM mempool 
        GROUP BY kind
),  forgotten AS (
    SELECT kind, count(*) AS total
        FROM forgotten
        GROUP BY kind
)
SELECT included.kind, t.forgotten, t.mempool, included.confirmed 
    FROM (
        SELECT mempool.kind, 
                 mempool.total AS mempool,
                 forgotten.total AS forgotten
              FROM mempool FULL OUTER JOIN forgotten ON
              mempool.kind = forgotten.kind
    ) AS t FULL OUTER JOIN included ON
    t.kind = included.kind
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


-- histograms of tx confirmation times 
CREATE OR REPLACE VIEW pubkey_histogram AS
SELECT kind, 
       width_bucket(
               extract(epoch FROM included.conf_time), 
               -1,
               -- TODO runs for every row!
               4000,--max_conftime('pubkeyhash'),
               90
        ), 
        count(*)
        FROM included
        WHERE kind = 'pubkeyhash'
        GROUP BY 1, 2
        ORDER BY 2
;


-- Execute external views
SELECT * FROM avg_conf_time;
SELECT * FROM conf_rates;
SELECT * FROM pubkey_histogram;
