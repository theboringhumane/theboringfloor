String formatRelativeTime(DateTime value, {DateTime? now}) {
  final difference = (now ?? DateTime.now()).difference(value);
  if (difference.isNegative || difference.inSeconds < 10) return 'just now';
  if (difference.inMinutes < 1) return '${difference.inSeconds}s ago';
  if (difference.inHours < 1) return '${difference.inMinutes}m ago';
  if (difference.inDays < 1) return '${difference.inHours}h ago';
  if (difference.inDays < 7) return '${difference.inDays}d ago';
  return '${difference.inDays ~/ 7}w ago';
}
