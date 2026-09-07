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
  final String id, dir, name, backend, primaryId, version;
  final bool live;
  final int port, savedAt, chatCount;
  factory Project.fromJson(Map<String, dynamic> json) => Project(
    id: json['id'] as String,
    dir: json['dir'] as String,
    name: json['name'] as String,
    live: json['live'] as bool,
    backend: json['backend'] as String,
    primaryId: json['primaryId'] as String? ?? '',
    port: (json['port'] as num?)?.toInt() ?? 0,
    version: json['version'] as String? ?? '',
    savedAt: (json['savedAt'] as num?)?.toInt() ?? 0,
    chatCount: (json['chatCount'] as num?)?.toInt() ?? 0,
  );
}
