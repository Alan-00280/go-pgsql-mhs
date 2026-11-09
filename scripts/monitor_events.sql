SELECT o.id, o.event_type, o.published_at IS NOT NULL AS di-publish,
       n.subject, n.sent_at
FROM outbox o
LEFT JOIN notifications n ON n.outbox_id = o.id
ORDER BY o.id DESC
LIMIT 10;
