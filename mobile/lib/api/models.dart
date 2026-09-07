class Project {
  const Project({
    required this.id,
    required this.dir,
    required this.name,
    required this.live,
    required this.backend,
    required this.primaryId,
    required this.port,
    required this.version,
    required this.savedAt,
    required this.chatCount,
  });

  final String id;
  final String dir;
  final String name;
  final bool live;
  final String backend;
  final String primaryId;
  final int port;
  final String version;
  final int savedAt;
  final int chatCount;

  factory Project.fromJson(Map<String, dynamic> json) => Project(
    id: json['id'] as String,
    dir: json['dir'] as String,
    name: json['name'] as String,
    live: json['live'] as bool,
    backend: json['backend'] as String,
    primaryId: (json['primaryId'] as String?) ?? '',
    port: (json['port'] as num?)?.toInt() ?? 0,
    version: (json['version'] as String?) ?? '',
    savedAt: (json['savedAt'] as num?)?.toInt() ?? 0,
    chatCount: (json['chatCount'] as num?)?.toInt() ?? 0,
  );
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

  final String id;
  final String from;
  final String kind;
  final String text;
  final int at;
  final String? meta;

  factory TranscriptMessage.fromJson(Map<String, dynamic> json) =>
      TranscriptMessage(
        id: json['id'] as String,
        from: json['from'] as String,
        kind: (json['kind'] as String?) ?? '',
        text: json['text'] as String,
        at: (json['at'] as num).toInt(),
        meta: json['meta'] as String?,
      );
}

class Status {
  const Status({
    required this.dir,
    required this.backend,
    required this.primaryId,
    required this.planDraftLen,
    required this.planApprovedLen,
    required this.chatCount,
  });

  final String dir;
  final String backend;
  final String primaryId;
  final int planDraftLen;
  final int planApprovedLen;
  final int chatCount;

  factory Status.fromJson(Map<String, dynamic> json) => Status(
    dir: json['dir'] as String,
    backend: json['backend'] as String,
    primaryId: (json['primaryId'] as String?) ?? '',
    planDraftLen: (json['planDraftLen'] as num?)?.toInt() ?? 0,
    planApprovedLen: (json['planApprovedLen'] as num?)?.toInt() ?? 0,
    chatCount: (json['chatCount'] as num?)?.toInt() ?? 0,
  );
}

class Busy {
  const Busy({
    required this.busy,
    required this.pendingBoss,
    required this.thinking,
    required this.delegating,
    required this.questionParked,
  });

  final bool busy;
  final bool pendingBoss;
  final bool thinking;
  final bool delegating;
  final bool questionParked;

  factory Busy.fromJson(Map<String, dynamic> json) => Busy(
    busy: json['busy'] as bool,
    pendingBoss: json['pendingBoss'] as bool,
    thinking: json['thinking'] as bool,
    delegating: json['delegating'] as bool,
    questionParked: json['questionParked'] as bool,
  );
}
