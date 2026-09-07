import 'transcript.dart';

/// A run of consecutive agent-activity messages folded into one row.
class ActivityGroup {
  const ActivityGroup({
    required this.messages,
    required this.summary,
    this.taskTitle,
    this.agentType,
    this.tools = const [],
  });

  final List<TranscriptMessage> messages;
  final String summary;
  final String? taskTitle;
  final String? agentType;
  final List<ActivityTool> tools;
}

/// A single worker tool invocation shown beneath an activity task.
class ActivityTool {
  const ActivityTool({required this.name, required this.arguments});

  final String name;
  final String arguments;
}

sealed class TranscriptEntry {
  const TranscriptEntry();
}

/// A standalone message rendered on its own (member or assistant).
final class MessageEntry extends TranscriptEntry {
  const MessageEntry(this.message);

  final TranscriptMessage message;
}

/// A folded run of agent activity.
final class ActivityEntry extends TranscriptEntry {
  const ActivityEntry(this.group);

  final ActivityGroup group;
}

/// Folds a flat, ascending-by-time transcript into display entries.
List<TranscriptEntry> groupTranscript(List<TranscriptMessage> messages) {
  final entries = <TranscriptEntry>[];
  final activityRun = <TranscriptMessage>[];

  void addActivityRun() {
    if (activityRun.isEmpty) return;
    final groupedMessages = List<TranscriptMessage>.unmodifiable(activityRun);
    entries.add(
      ActivityEntry(
        ActivityGroup(
          messages: groupedMessages,
          summary: _summarizeActivity(groupedMessages),
          taskTitle: _taskFor(groupedMessages).title,
          agentType: _taskFor(groupedMessages).agentType,
          tools: List<ActivityTool>.unmodifiable(
            groupedMessages
                .where((message) => message.kind == 'wtool')
                .map(_toolFor),
          ),
        ),
      ),
    );
    activityRun.clear();
  }

  for (final message in messages) {
    if (_isActivity(message)) {
      activityRun.add(message);
    } else {
      addActivityRun();
      entries.add(MessageEntry(message));
    }
  }
  addActivityRun();

  return entries;
}

bool _isActivity(TranscriptMessage message) =>
    message.kind == 'wthink' || message.kind == 'wtool';

_ActivityTask _taskFor(List<TranscriptMessage> messages) {
  final thought = messages
      .where((message) => message.kind == 'wthink')
      .firstOrNull;
  if (thought == null) return const _ActivityTask();
  final text = thought.text.trim();
  if (text.isEmpty) return const _ActivityTask();

  final match = RegExp(
    r'^(.+?\s+Task)\s*[—-]\s*(.+?)\s*\(@([A-Za-z][A-Za-z0-9_-]*)\s+subagent\)\s*$',
  ).firstMatch(text);
  if (match != null) {
    return _ActivityTask(
      title: '${match.group(1)} — ${match.group(2)}',
      agentType: match.group(3),
    );
  }

  // A task-shaped but incomplete worker row still has useful text to show.
  // Ordinary worker thoughts retain the compact summary-only presentation.
  if (text.contains(RegExp(r'\bTask\b'))) return _ActivityTask(title: text);
  return const _ActivityTask();
}

ActivityTool _toolFor(TranscriptMessage message) {
  final text = message.text.trim().replaceAll(RegExp(r'\s+'), ' ');
  final separator = text.indexOf('·');
  if (separator >= 0) {
    final name = text.substring(0, separator).trim();
    final arguments = text.substring(separator + 1).trim();
    if (name.isNotEmpty) return ActivityTool(name: name, arguments: arguments);
  }
  final match = RegExp(r'^([A-Za-z][A-Za-z0-9_-]*)(?:\s+(.*))?$')
      .firstMatch(text);
  if (match != null) {
    return ActivityTool(
      name: match.group(1)!,
      arguments: match.group(2)?.trim() ?? '',
    );
  }
  return ActivityTool(name: text, arguments: '');
}

class _ActivityTask {
  const _ActivityTask({this.title, this.agentType});

  final String? title;
  final String? agentType;
}

/// Produces the compact, count-only label for one folded activity run.
///
/// Keep wording decisions here so they can be adjusted independently of the
/// grouping algorithm.
String _summarizeActivity(List<TranscriptMessage> messages) {
  final counts = <String, int>{};
  for (final message in messages) {
    final category = message.kind == 'wthink'
        ? 'thought'
        : _toolCategory(message.text);
    counts.update(category, (count) => count + 1, ifAbsent: () => 1);
  }

  return counts.entries
      .map((entry) => '${entry.value} ${_pluralize(entry.key, entry.value)}')
      .join(' · ');
}

String _toolCategory(String text) {
  final separatorIndex = text.indexOf('·');
  if (separatorIndex < 0) return 'tool';

  final toolName = text.substring(0, separatorIndex).trim();
  if (toolName.isEmpty ||
      toolName.contains(RegExp(r'\s')) ||
      toolName.startsWith('{') ||
      toolName.startsWith('[') ||
      toolName.length > 64) {
    return 'tool';
  }
  return toolName;
}

String _pluralize(String word, int count) {
  if (count == 1) return word;

  final lowerCase = word.toLowerCase();
  if (lowerCase.endsWith('y') &&
      lowerCase.length > 1 &&
      !_isVowel(lowerCase[lowerCase.length - 2])) {
    return '${word.substring(0, word.length - 1)}ies';
  }
  if (lowerCase.endsWith('s') ||
      lowerCase.endsWith('x') ||
      lowerCase.endsWith('z') ||
      lowerCase.endsWith('ch') ||
      lowerCase.endsWith('sh')) {
    return '${word}es';
  }
  return '${word}s';
}

bool _isVowel(String character) => 'aeiou'.contains(character);
