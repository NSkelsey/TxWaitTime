DELETE FROM relations 
    USING included
    WHERE extract(epoch FROM included.conf_time) < 0
    AND included.txid = relations.txid
;

-- delete all txs with conf_time greater less than zero
DELETE FROM txs 
    USING included
    WHERE extract(epoch FROM included.conf_time) < 0
    AND included.txid = txs.txid
;
