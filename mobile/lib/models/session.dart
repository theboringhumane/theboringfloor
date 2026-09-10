import 'transcript.dart';

class Status {
  const Status({
    required this.dir,
    required this.backend,
    required this.primaryId,
    required this.planDraftLen,
    required this.planApprovedLen,
    required this.chatCount,
    this.planPending = false,
    this.planRevision = "",
    this.execution,
  });
  final String dir, backend, primaryId;
  final bool planPending;
  final String planRevision;
  final ExecutionStatus? execution;
  final int planDraftLen, planApprovedLen, chatCount;
  factory Status.fromJson(Map<String, dynamic> j) => Status(
    planPending: j['planPending'] == true,
    planRevision: j['planRevision'] as String? ?? '',
    execution: j['execution'] is Map<String, dynamic>
        ? ExecutionStatus.fromJson(j['execution'] as Map<String, dynamic>)
        : null,
    dir: j['dir'] as String,
    backend: j['backend'] as String,
    primaryId: j['primaryId'] as String? ?? '',
    planDraftLen: (j['planDraftLen'] as num?)?.toInt() ?? 0,
    planApprovedLen: (j['planApprovedLen'] as num?)?.toInt() ?? 0,
    chatCount: (j['chatCount'] as num?)?.toInt() ?? 0,
  );
}

class ExecutionStatus {
  const ExecutionStatus({
    required this.state,
    required this.summary,
    this.action = '',
    this.planSafety = '',
  });
  final String state, summary, action, planSafety;
  bool get needsAttention =>
      {'plan', 'permission', 'question', 'offline'}.contains(state);
  factory ExecutionStatus.fromJson(Map<String, dynamic> json) =>
      ExecutionStatus(
        state: json['state'] as String? ?? 'unknown',
        summary: json['summary'] as String? ?? 'Status unavailable',
        action: json['action'] as String? ?? '',
        planSafety: json['planSafety'] as String? ?? '',
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
  final bool busy, pendingBoss, thinking, delegating, questionParked;
  factory Busy.fromJson(Map<String, dynamic> j) => Busy(
    busy: j['busy'] as bool,
    pendingBoss: j['pendingBoss'] as bool,
    thinking: j['thinking'] as bool,
    delegating: j['delegating'] as bool,
    questionParked: j['questionParked'] as bool,
  );
}

class SessionData {
  const SessionData({
    required this.status,
    required this.messages,
    this.hasMore = false,
    this.busy,
  });
  final Status status;
  final List<TranscriptMessage> messages;

  /// Whether the office holds transcript history older than [messages].
  final bool hasMore;
  final Busy? busy;
}

enum StartOfficeOutcome {
  accepted,
  alreadyRunning,
  projectNotFound,
  launchFailed,
}

class StartOfficeResult {
  const StartOfficeResult(this.outcome, {this.message});
  final StartOfficeOutcome outcome;
  final String? message;
}
