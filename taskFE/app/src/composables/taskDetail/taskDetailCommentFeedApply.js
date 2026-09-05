/**
 * Attach parallel comment-feed results onto a task-detail payload.
 */
export function applyCommentFeedToTask(data, field, feed, cursorKey, hasMoreKey) {
  if (feed.error) return
  data[field] = feed.data
  data[cursorKey] = feed.nextCursor ?? null
  data[hasMoreKey] = Boolean(feed.hasMore)
}

export function attachCommentFeeds(data, humanFeed, aiFeed, agentFeed) {
  applyCommentFeedToTask(data, 'comments', humanFeed, 'comments_next_cursor', 'comments_has_more')
  applyCommentFeedToTask(data, 'ai_comments', aiFeed, 'ai_comments_next_cursor', 'ai_comments_has_more')
  applyCommentFeedToTask(
    data,
    'container_agent_comments',
    agentFeed,
    'container_agent_comments_next_cursor',
    'container_agent_comments_has_more',
  )
  const feedErrors = [
    ...(Array.isArray(data.comments_feed_errors) ? data.comments_feed_errors : []),
    ...(humanFeed.error ? [humanFeed.error] : []),
    ...(aiFeed.error ? [aiFeed.error] : []),
    ...(agentFeed.error ? [agentFeed.error] : []),
  ]
  if (feedErrors.length > 0) {
    data.comments_feed_errors = feedErrors
  } else {
    delete data.comments_feed_errors
  }
  data.comments_feeds_loaded = true
}
