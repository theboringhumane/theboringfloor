import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/components/activity_bubble.dart';
import 'package:theboringfloor/components/loading_snake.dart';
import 'package:theboringfloor/models/models.dart';
import 'package:theboringfloor/theme.dart';

TranscriptMessage _message({
  required String id,
  required String kind,
  required String text,
  TranscriptActivity? activity,
}) => TranscriptMessage(
  id: id,
  from: 'worker',
  kind: kind,
  text: text,
  at: 1,
  activity: activity,
);

ActivityGroup _group({bool withTools = true}) {
  final messages = [
    _message(
      id: 'task',
      kind: 'wthink',
      text: 'Developer Task — Untrack Gradle build artifact (@developer subagent)',
    ),
    if (withTools)
      _message(
        id: 'tool',
        kind: 'wtool',
        text: 'Bash flutter analyze && flutter test',
      ),
  ];
  return (groupTranscript(messages).single as ActivityEntry)
      .group; // Keep test data on the same parsing path as production.
}

Future<void> _pump(
  WidgetTester tester, {
  required ActivityGroup group,
  bool expanded = false,
  bool running = false,
  bool officeWorking = false,
  VoidCallback? onToggle,
  double width = 400,
}) => tester.pumpWidget(
  MaterialApp(
    theme: buildLightTheme(),
    home: MediaQuery(
      data: MediaQueryData(size: Size(width, 800)),
      child: Scaffold(
        body: SizedBox(
          width: width,
          child: ActivityBubble(
            group: group,
            expanded: expanded,
            running: running,
            officeWorking: officeWorking,
            onToggle: onToggle ?? () {},
          ),
        ),
      ),
    ),
  ),
);

void main() {
  testWidgets('renders a task parent, chip, and no child tools when absent', (
    tester,
  ) async {
    await _pump(tester, group: _group(withTools: false));

    expect(
      find.text('Developer Task — Untrack Gradle build artifact'),
      findsOneWidget,
    );
    expect(find.text('@developer'), findsOneWidget);
    expect(find.textContaining('Bash'), findsNothing);
  });

  testWidgets('shows the loading snake only while running', (tester) async {
    await _pump(tester, group: _group(), running: true);
    expect(find.byType(LoadingSnake), findsOneWidget);

    await _pump(tester, group: _group(), running: false);
    expect(find.byType(LoadingSnake), findsNothing);
  });

  testWidgets('expansion reveals and collapses child tools', (tester) async {
    var expanded = false;
    await _pump(tester, group: _group(), onToggle: () => expanded = !expanded);
    expect(find.text('flutter analyze && flutter test'), findsNothing);

    await _pump(tester, group: _group(), expanded: expanded);
    expect(find.text('flutter analyze && flutter test'), findsNothing);

    expanded = true;
    await _pump(tester, group: _group(), expanded: expanded);
    expect(find.text('flutter analyze && flutter test'), findsOneWidget);

    expanded = false;
    await _pump(tester, group: _group(), expanded: expanded);
    expect(find.text('flutter analyze && flutter test'), findsNothing);
  });

  testWidgets('does not overflow at 320px with long task and tool rows', (
    tester,
  ) async {
    final messages = [
      _message(
        id: 'task',
        kind: 'wthink',
        text: 'Developer Task — A deliberately very long task title that must ellipsize safely (@developer subagent)',
      ),
      for (var index = 0; index < 4; index++)
        _message(
          id: 'tool-$index',
          kind: 'wtool',
          text:
              'Bash flutter analyze --a-very-long-argument-that-must-never-overflow-$index',
        ),
    ];
    final group = (groupTranscript(messages).single as ActivityEntry).group;

    await _pump(tester, group: group, expanded: true, width: 320);

    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'structured running activity shows its snake despite false override',
    (tester) async {
      final group =
          (groupTranscript([
                    _message(
                      id: 'thought',
                      kind: 'wthink',
                      text: 'misleading legacy prose',
                      activity: const TranscriptActivity(
                        role: 'developer',
                        task: 'Structured task',
                      ),
                    ),
                    _message(
                      id: 'tool',
                      kind: 'wtool',
                      text: 'Bash · flutter test',
                      activity: const TranscriptActivity(state: 'running'),
                    ),
                  ]).single
                  as ActivityEntry)
              .group;

      await _pump(tester, group: group, running: false, officeWorking: true);

      expect(find.text('Developer Task — Structured task'), findsOneWidget);
      expect(find.text('@developer'), findsOneWidget);
      expect(find.byType(LoadingSnake), findsOneWidget);
    },
  );

  testWidgets('idle office hides a stale structured running snake', (
    tester,
  ) async {
    final group =
        (groupTranscript([
                  _message(
                    id: 'tool',
                    kind: 'wtool',
                    text: 'Bash · stale command',
                    activity: const TranscriptActivity(state: 'running'),
                  ),
                ]).single
                as ActivityEntry)
            .group;

    await _pump(tester, group: group, running: false, officeWorking: false);

    expect(find.byType(LoadingSnake), findsNothing);
  });

  testWidgets('working office shows a structured running snake', (
    tester,
  ) async {
    final group =
        (groupTranscript([
                  _message(
                    id: 'tool',
                    kind: 'wtool',
                    text: 'Bash · current command',
                    activity: const TranscriptActivity(state: 'running'),
                  ),
                ]).single
                as ActivityEntry)
            .group;

    await _pump(tester, group: group, running: false, officeWorking: true);

    expect(find.byType(LoadingSnake), findsOneWidget);
  });

  testWidgets('unknown structured roles render no label or agent chip', (
    tester,
  ) async {
    final group =
        (groupTranscript([
                  _message(
                    id: 'thought',
                    kind: 'wthink',
                    text: 'internal task',
                    activity: const TranscriptActivity(
                      role: 'quartermaster',
                      task: 'Unknown role task',
                    ),
                  ),
                ]).single
                as ActivityEntry)
            .group;

    await _pump(tester, group: group);

    expect(find.text('@quartermaster'), findsNothing);
    expect(find.text('Quartermaster Task — Unknown role task'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('known structured roles retain their lowercase agent chip', (
    tester,
  ) async {
    final group =
        (groupTranscript([
                  _message(
                    id: 'thought',
                    kind: 'wthink',
                    text: 'internal task',
                    activity: const TranscriptActivity(
                      role: 'developer',
                      task: 'Known role task',
                    ),
                  ),
                ]).single
                as ActivityEntry)
            .group;

    await _pump(tester, group: group);

    expect(find.text('@developer'), findsOneWidget);
  });

  testWidgets('structured failed tool uses the theme error colour and marker', (
    tester,
  ) async {
    final group =
        (groupTranscript([
                  _message(
                    id: 'tool',
                    kind: 'wtool',
                    text: 'Bash · fail loudly',
                    activity: const TranscriptActivity(state: 'error'),
                  ),
                ]).single
                as ActivityEntry)
            .group;

    await _pump(tester, group: group, expanded: true);

    expect(find.byKey(const Key('activity-tool-failure-Bash')), findsOneWidget);
    expect(
      tester.widget<Text>(find.text('↳ Bash')).style?.color,
      buildLightTheme().colorScheme.error,
    );
  });

  testWidgets('legacy activity retains parsed title, chip, and no snake', (
    tester,
  ) async {
    await _pump(tester, group: _group(), running: false);

    expect(
      find.text('Developer Task — Untrack Gradle build artifact'),
      findsOneWidget,
    );
    expect(find.text('@developer'), findsOneWidget);
    expect(find.byType(LoadingSnake), findsNothing);
  });
}
