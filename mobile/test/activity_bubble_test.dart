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
}) => TranscriptMessage(id: id, from: 'worker', kind: kind, text: text, at: 1);

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
}
