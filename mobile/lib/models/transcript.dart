/// Messages fetched per transcript page. Keeps the first paint cheap and
/// bounds every backward page the office has to project.
const int transcriptPageSize = 50;

/// One page of transcript history, plus whether older history remains.
///
/// [hasMore] comes from the office and is the ONLY safe signal to stop paging:
/// a short page does not imply the end of history.
class TranscriptPage {
  const TranscriptPage({required this.messages, required this.hasMore});

  final List<TranscriptMessage> messages;
  final bool hasMore;
}

class TranscriptMessage {
  const TranscriptMessage({
    required this.id,
    required this.from,
    required this.kind,
    required this.text,
    required this.at,
    this.meta,
  });
  final String id, from, kind, text;
  final int at;
  final String? meta;
  factory TranscriptMessage.fromJson(Map<String, dynamic> json) =>
      TranscriptMessage(
        id: json['id'] as String,
        from: json['from'] as String,
        kind: json['kind'] as String? ?? '',
        text: json['text'] as String,
        at: (json['at'] as num).toInt(),
        meta: json['meta'] as String?,
      );
}
