const ticketStatuses = ['backlog', 'in-progress', 'blocked', 'review', 'done'];
const backends = ['claudecode', 'opencode', 'codex'];
String backendLabel(String value) => switch (value) {
  'claudecode' => 'Claude Code',
  'opencode' => 'OpenCode',
  'codex' => 'Codex',
  _ => value,
};
String statusLabel(String value) => switch (value) {
  'in-progress' => 'In progress',
  'backlog' => 'Backlog',
  'blocked' => 'Blocked',
  'review' => 'Review',
  'done' => 'Done',
  _ => value,
};

class FloorTeam {
  const FloorTeam(this.id, this.name);
  final String id, name;
  factory FloorTeam.fromJson(Map<String, dynamic> j) =>
      FloorTeam(j['id'] as String, j['name'] as String);
}

class FloorConversation {
  const FloorConversation({
    required this.id,
    required this.backend,
    required this.title,
    this.team = '',
    this.messages = 0,
  });
  final String id, backend, title, team;
  final int messages;
  factory FloorConversation.fromJson(Map<String, dynamic> j) =>
      FloorConversation(
        id: j['id'] as String,
        backend: j['backend'] as String,
        title: j['title'] as String? ?? 'Untitled conversation',
        team: j['team'] as String? ?? '',
        messages: (j['messages'] as num?)?.toInt() ?? 0,
      );
}

class FloorTicket {
  const FloorTicket(this.json);
  final Map<String, dynamic> json;
  String get id => json['id'] as String? ?? '';
  String get title => json['title'] as String? ?? '';
  String get description => json['description'] as String? ?? '';
  String get status => json['status'] as String? ?? 'backlog';
  String get priority => json['priority'] as String? ?? 'P2';
  String get team => json['team'] as String? ?? '';
  String get owner => json['owner'] as String? ?? '';
  List<Map<String, dynamic>> get checklist => (json['checklist'] as List? ?? [])
      .map((c) => Map<String, dynamic>.from(c as Map))
      .toList();
}

class FloorData {
  const FloorData({
    required this.name,
    required this.teams,
    required this.tickets,
    required this.conversations,
  });
  final String name;
  final List<FloorTeam> teams;
  final List<FloorTicket> tickets;
  final List<FloorConversation> conversations;
  factory FloorData.fromJson(Map<String, dynamic> j) {
    final floor = j['floor'] as Map<String, dynamic>;
    return FloorData(
      name: floor['name'] as String,
      teams: (floor['teams'] as List? ?? [])
          .map((t) => FloorTeam.fromJson(t as Map<String, dynamic>))
          .toList(),
      tickets: (floor['tickets'] as List? ?? [])
          .map((t) => FloorTicket(t as Map<String, dynamic>))
          .toList(),
      conversations: (j['conversations'] as List? ?? [])
          .map((c) => FloorConversation.fromJson(c as Map<String, dynamic>))
          .toList(),
    );
  }
}

class ProjectFile {
  const ProjectFile({
    required this.path,
    required this.name,
    required this.directory,
  });
  final String path, name;
  final bool directory;
  factory ProjectFile.fromJson(Map<String, dynamic> j) => ProjectFile(
    path: j['path'] as String,
    name: j['name'] as String,
    directory: j['directory'] == true,
  );
}

class ProjectFilePage {
  const ProjectFilePage({
    required this.path,
    required this.directory,
    required this.entries,
    required this.content,
    this.truncated = false,
    this.binary = false,
  });
  final String path, content;
  final bool directory, truncated, binary;
  final List<ProjectFile> entries;
  factory ProjectFilePage.fromJson(Map<String, dynamic> j) => ProjectFilePage(
    path: j['path'] as String? ?? '',
    directory: j['directory'] == true,
    entries: (j['entries'] as List? ?? [])
        .map((e) => ProjectFile.fromJson(e as Map<String, dynamic>))
        .toList(),
    content: j['content'] as String? ?? '',
    truncated: j['truncated'] == true,
    binary: j['binary'] == true,
  );
}

class FloorPlan {
  const FloorPlan({
    this.draft = '',
    this.approved = '',
    this.pending = false,
    this.error = '',
  });
  final String draft, approved, error;
  final bool pending;
  factory FloorPlan.fromJson(Map<String, dynamic> j) => FloorPlan(
    draft: j['draft'] as String? ?? '',
    approved: j['approved'] as String? ?? '',
    pending: j['pending'] == true,
    error: j['error'] as String? ?? '',
  );
}
