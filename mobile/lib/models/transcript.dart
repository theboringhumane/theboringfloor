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
    this.activity,
    this.pending = false,
    this.phase,
    this.attachments = const [],
  });
  final String id, from, kind, text;
  final int at;
  final String? meta;
  final TranscriptActivity? activity;
  final bool pending;
  final String? phase;
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
      activity: TranscriptActivity.fromJson(json['activity']),
      pending: json['pending'] == true,
      phase: (json['phase'] ?? json['channel']) as String?,
      attachments: attachments,
    );
  }
}

/// The phone's conversation is a reading surface, not an execution log.
/// Keep raw messages in the store for opaque pagination cursors. Never strip
/// commands/code out of a real answer: classify rows using their wire metadata.
List<TranscriptMessage> conversationMessages(
  List<TranscriptMessage> messages, {
  bool working = false,
}) {
  final result = <TranscriptMessage>[];
  TranscriptMessage? reply;
  var hasUser = false;
  for (final message in messages) {
    if (message.pending ||
        (message.text.trim().isEmpty && message.attachments.isEmpty)) {
      continue;
    }
    if (message.from == 'user' &&
        {'', 'user', 'message', 'text', 'input_text'}.contains(message.kind)) {
      if (reply != null) result.add(reply);
      reply = null;
      hasUser = true;
      result.add(message);
      continue;
    }
    final primary =
        {'boss', 'assistant'}.contains(message.from) ||
        (message.from == 'office' &&
            message.kind == 'office' &&
            message.id.startsWith('office-'));
    if (!primary ||
        !{
          '',
          'boss',
          'assistant',
          'message',
          'text',
          'final',
          'answer',
          'office',
          'chat',
        }.contains(message.kind) ||
        {'analysis', 'commentary', 'reasoning'}.contains(message.phase)) {
      continue;
    }
    if (hasUser) {
      // Older servers have no phase field. The last completed primary-agent
      // message in a user turn is its answer; earlier prose is progress.
      reply = message;
    } else {
      // A paged history may begin after its user row; don't discard answers
      // from that incomplete prefix just because the cursor split the turn.
      result.add(message);
    }
  }
  if (reply != null && (!working || reply.phase == 'final')) result.add(reply);
  return result;
}

/// Optional structured worker activity supplied by newer office servers.
///
/// Older offices omit this object entirely, so every field remains nullable.
class TranscriptActivity {
  const TranscriptActivity({this.role, this.task, this.state});

  final String? role;
  final String? task;
  final String? state;

  static TranscriptActivity? fromJson(Object? json) {
    if (json is! Map) return null;
    final role = json['role'];
    final task = json['task'];
    final state = json['state'];
    return TranscriptActivity(
      role: role is String ? role : null,
      task: task is String ? task : null,
      state: state is String ? state : null,
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
