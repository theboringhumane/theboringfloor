/// Messages fetched per transcript page. Keeps the first paint cheap and
/// bounds every backward page the office has to project.
const int transcriptPageSize = 500;

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
    this.attachments = const [],
  });
  final String id, from, kind, text;
  final int at;
  final String? meta;
  final List<TranscriptAttachment> attachments;

  factory TranscriptMessage.fromJson(Map<String, dynamic> json) {
    final attachmentJson = json['attachments'];
    final attachments = attachmentJson is List
        ? attachmentJson
              .whereType<Map>()
              .map(TranscriptAttachment.fromJson)
              .whereType<TranscriptAttachment>()
              .toList(growable: false)
        : const <TranscriptAttachment>[];
    return TranscriptMessage(
      id: json['id'] as String,
      from: json['from'] as String,
      kind: json['kind'] as String? ?? '',
      text: json['text'] as String,
      at: (json['at'] as num).toInt(),
      meta: json['meta'] as String?,
      attachments: attachments,
    );
  }
}

/// Metadata for an image attached to a past transcript message.
///
/// Transcript responses intentionally contain no image bytes or fetchable URL.
class TranscriptAttachment {
  const TranscriptAttachment({required this.name, this.mime});

  final String name;
  final String? mime;

  static TranscriptAttachment? fromJson(Map<dynamic, dynamic> json) {
    final name = json['name'];
    if (name is! String) return null;
    final mime = json['mime'];
    return TranscriptAttachment(name: name, mime: mime is String ? mime : null);
  }
}
