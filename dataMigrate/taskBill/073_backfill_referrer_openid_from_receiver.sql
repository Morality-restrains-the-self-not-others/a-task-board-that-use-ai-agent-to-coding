-- 073: backfill billing_profit_sharing.referrer_openid from the registered
-- billing_profit_sharing_receiver pair (OPT-20260826-015). Read/scan paths that
-- still snapshot referrer_openid must not see a website-app openid (or empty)
-- while the receiver table already holds the registered WeChat Pay openid.
-- Idempotent: only touches rows whose referrer_openid is empty or diverges from
-- the receiver openid; re-running finds nothing to change.
-- Scale: bounded by unmatched paid referred orders; no partition.

UPDATE billing_profit_sharing ps
JOIN billing_profit_sharing_receiver rcv
  ON ps.referrer_user_id = rcv.referrer_user_id
SET ps.referrer_openid = rcv.openid
WHERE rcv.status = 'registered'
  AND rcv.openid <> ''
  AND (ps.referrer_openid = '' OR ps.referrer_openid <> rcv.openid);
