import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/models/models.dart';

TranscriptMessage _message({
  required String id,
  required String from,
  required String kind,
  required String text,
  required int at,
  TranscriptActivity? activity,
}) => TranscriptMessage(
  id: id,
  from: from,
  kind: kind,
  text: text,
  at: at,
  activity: activity,
);

void main() {
  test('folds five consecutive activity messages into one entry', () {
    final messages = [
      _message(id: '1', from: 'agent', kind: 'wthink', text: 'plan', at: 1),
      _message(
        id: '2',
        from: 'agent',
        kind: 'wtool',
        text: 'search · auth',
        at: 2,
      ),
      _message(
        id: '3',
        from: 'agent',
        kind: 'wtool',
        text: 'search · tests',
        at: 3,
      ),
      _message(id: '4', from: 'agent', kind: 'wthink', text: 'check', at: 4),
      _message(
        id: '5',
        from: 'agent',
        kind: 'wtool',
        text: 'read · app.dart',
        at: 5,
      ),
    ];

    final entries = groupTranscript(messages);

    expect(entries, hasLength(1));
    final entry = entries.single as ActivityEntry;
    expect(entry.group.messages, orderedEquals(messages));
    expect(entry.group.summary, '2 thoughts · 2 searches · 1 read');
  });

  test('a non-activity message splits activity runs', () {
    final first = _message(
      id: '1',
      from: 'agent',
      kind: 'wthink',
      text: 'plan',
      at: 1,
    );
    final member = _message(
      id: '2',
      from: 'user',
      kind: 'member',
      text: 'Continue',
      at: 2,
    );
    final second = _message(
      id: '3',
      from: 'agent',
      kind: 'wtool',
      text: 'read · lib/main.dart',
      at: 3,
    );

    final entries = groupTranscript([first, member, second]);

    expect(entries.map((entry) => entry.runtimeType), [
      ActivityEntry,
      MessageEntry,
      ActivityEntry,
    ]);
    expect((entries[0] as ActivityEntry).group.messages, [first]);
    expect((entries[1] as MessageEntry).message, member);
    expect((entries[2] as ActivityEntry).group.messages, [second]);
  });

  test('member and assistant messages remain standalone in original order', () {
    final member = _message(
      id: 'member',
      from: 'user',
      kind: 'member',
      text: 'Please investigate',
      at: 1,
    );
    final thought = _message(
      id: 'thought',
      from: 'agent',
      kind: 'wthink',
      text: 'investigating',
      at: 2,
    );
    final assistant = _message(
      id: 'assistant',
      from: 'agent',
      kind: 'assistant',
      text: 'Done',
      at: 3,
    );

    final entries = groupTranscript([member, thought, assistant]);

    expect(entries.map((entry) => entry.runtimeType), [
      MessageEntry,
      ActivityEntry,
      MessageEntry,
    ]);
    expect((entries[0] as MessageEntry).message, member);
    expect((entries[2] as MessageEntry).message, assistant);
  });

  test('a single activity message becomes a one-message activity entry', () {
    final activity = _message(
      id: '1',
      from: 'agent',
      kind: 'wtool',
      text: 'bash · gofmt -l lib',
      at: 1,
    );

    final entries = groupTranscript([activity]);

    expect(entries.single, isA<ActivityEntry>());
    final entry = entries.single as ActivityEntry;
    expect(entry.group.messages, [activity]);
    expect(entry.group.summary, '1 bash');
  });

  test('summaries pluralize thought and tool categories correctly', () {
    final singular =
        groupTranscript([
              _message(id: '1', from: 'agent', kind: 'wthink', text: '', at: 1),
              _message(
                id: '2',
                from: 'agent',
                kind: 'wtool',
                text: 'search · one',
                at: 2,
              ),
            ]).single
            as ActivityEntry;
    final plural =
        groupTranscript([
              _message(id: '3', from: 'agent', kind: 'wthink', text: '', at: 3),
              _message(id: '4', from: 'agent', kind: 'wthink', text: '', at: 4),
              _message(
                id: '5',
                from: 'agent',
                kind: 'wtool',
                text: 'search · one',
                at: 5,
              ),
              _message(
                id: '6',
                from: 'agent',
                kind: 'wtool',
                text: 'search · two',
                at: 6,
              ),
            ]).single
            as ActivityEntry;

    expect(singular.group.summary, '1 thought · 1 search');
    expect(plural.group.summary, '2 thoughts · 2 searches');
  });

  test(
    'empty, unknown, and empty kinds remain standalone without throwing',
    () {
      final unknown = _message(
        id: 'unknown',
        from: 'agent',
        kind: 'status',
        text: 'working',
        at: 1,
      );
      final emptyKind = _message(
        id: 'empty',
        from: 'agent',
        kind: '',
        text: 'working',
        at: 2,
      );

      expect(groupTranscript([]), isEmpty);
      expect(groupTranscript([unknown, emptyKind]), [
        isA<MessageEntry>().having(
          (entry) => entry.message,
          'message',
          unknown,
        ),
        isA<MessageEntry>().having(
          (entry) => entry.message,
          'message',
          emptyKind,
        ),
      ]);
    },
  );

  test('summary has no newline and excludes tool invocation detail', () {
    final entry =
        groupTranscript([
              _message(
                id: '1',
                from: 'agent',
                kind: 'wtool',
                text: 'bash · printf "first line\\nsecond line"',
                at: 1,
              ),
            ]).single
            as ActivityEntry;

    expect(entry.group.summary, '1 bash');
    expect(entry.group.summary, isNot(contains('\n')));
    expect(entry.group.summary, isNot(contains('second line')));
  });

  test('activity grouping model does not import Flutter material widgets', () {
    final source = File('lib/models/activity_group.dart').readAsStringSync();

    expect(source, isNot(contains('package:flutter/material.dart')));
  });

  test('extracts a task title, agent type, and ordered child tools', () {
    final entry =
        groupTranscript([
              _message(
                id: 'task',
                from: 'worker',
                kind: 'wthink',
                text: 'Developer Task — Untrack Gradle build artifact (@developer subagent)',
                at: 1,
              ),
              _message(
                id: 'tool',
                from: 'worker',
                kind: 'wtool',
                text: 'Bash flutter analyze && flutter test',
                at: 2,
              ),
              _message(
                id: 'tool-2',
                from: 'worker',
                kind: 'wtool',
                text: 'Grep · Polling\\(|Busy\\(',
                at: 3,
              ),
            ]).single
            as ActivityEntry;

    expect(
      entry.group.taskTitle,
      'Developer Task — Untrack Gradle build artifact',
    );
    expect(entry.group.agentType, 'developer');
    expect(entry.group.tools.map((tool) => tool.name), ['Bash', 'Grep']);
    expect(entry.group.tools.map((tool) => tool.arguments), [
      'flutter analyze && flutter test',
      'Polling\\(|Busy\\(',
    ]);
  });

  test('malformed task-shaped text degrades to the raw parent line', () {
    const raw = 'Developer Task — missing subagent marker';
    final entry =
        groupTranscript([
              _message(
                id: 'malformed',
                from: 'worker',
                kind: 'wthink',
                text: raw,
                at: 1,
              ),
            ]).single
            as ActivityEntry;

    expect(entry.group.taskTitle, raw);
    expect(entry.group.agentType, isNull);
    expect(entry.group.tools, isEmpty);
  });

  test(
    'structured activity beats parsed task metadata and preserves tool state',
    () {
      final group =
          (groupTranscript([
                    _message(
                      id: 'thought',
                      from: 'worker',
                      kind: 'wthink',
                      text:
                          'Developer Task — Parsed title (@developer subagent)',
                      at: 1,
                      activity: const TranscriptActivity(
                        role: 'runner',
                        task: 'Structured title',
                      ),
                    ),
                    _message(
                      id: 'tool',
                      from: 'worker',
                      kind: 'wtool',
                      text: 'Bash · flutter test',
                      at: 2,
                      activity: const TranscriptActivity(state: 'running'),
                    ),
                  ]).single
                  as ActivityEntry)
              .group;

      expect(group.usesStructuredActivity, isTrue);
      expect(group.agentType, 'runner');
      expect(group.taskTitle, 'Structured title');
      expect(group.tools.single.state, 'running');
      expect(group.hasRunningTool, isTrue);
    },
  );

  test('structured role-only and state-only activity do not parse prose', () {
    final roleOnly =
        (groupTranscript([
                  _message(
                    id: 'role',
                    from: 'worker',
                    kind: 'wthink',
                    text: 'Developer Task — Parsed title (@developer subagent)',
                    at: 1,
                    activity: const TranscriptActivity(role: 'scout'),
                  ),
                ]).single
                as ActivityEntry)
            .group;
    final stateOnly =
        (groupTranscript([
                  _message(
                    id: 'state',
                    from: 'worker',
                    kind: 'wtool',
                    text: 'Bash · failed command',
                    at: 1,
                    activity: const TranscriptActivity(state: 'error'),
                  ),
                ]).single
                as ActivityEntry)
            .group;

    expect(roleOnly.agentType, 'scout');
    expect(roleOnly.taskTitle, isNull);
    expect(stateOnly.agentType, isNull);
    expect(stateOnly.taskTitle, isNull);
    expect(stateOnly.tools.single.state, 'error');
    expect(stateOnly.hasRunningTool, isFalse);
  });

  test('maps terminal office roles to display labels', () {
    expect(activityRoleLabel('developer'), 'Developer');
    expect(activityRoleLabel('scout'), 'Explore');
    expect(activityRoleLabel('reviewer'), 'Reviewer');
    expect(activityRoleLabel('runner'), 'Runner');
    expect(activityRoleLabel('cto'), 'CTO');
    expect(activityRoleLabel('hr'), 'HR');
    expect(activityRoleLabel('manager'), 'Manager');
    expect(activityRoleLabel('unknown'), isNull);
    expect(activityRoleLabel(null), isNull);
  });
}
