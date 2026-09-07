String formatRelativeTime(DateTime value, {DateTime? now}) {
  final d = (now ?? DateTime.now()).difference(value);
  if (d.isNegative || d.inSeconds < 10) return 'just now';
  if (d.inMinutes < 1) return '${d.inSeconds}s ago';
  if (d.inHours < 1) return '${d.inMinutes}m ago';
  if (d.inDays < 1) return '${d.inHours}h ago';
  if (d.inDays < 7) return '${d.inDays}d ago';
  return '${d.inDays ~/ 7}w ago';
}
