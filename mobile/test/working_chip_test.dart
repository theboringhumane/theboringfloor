import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/components/working_chip.dart';
import 'package:theboringfloor/theme.dart';
import 'package:theboringfloor/utils/working_words.dart';

void main() {
  Widget host(Widget child) => MaterialApp(
    theme: buildLightTheme(),
    home: Scaffold(body: Center(child: child)),
  );

  test('workingWordAt cycles and wraps', () {
    expect(workingWordAt(0), kWorkingWords.first);
    expect(workingWordAt(kWorkingWords.length), kWorkingWords.first);
    expect(workingWordAt(kWorkingWords.length + 1), kWorkingWords[1]);
  });

  testWidgets('WorkingChip renders nothing while stopped', (tester) async {
    await tester.pumpWidget(host(const WorkingChip(working: false)));

    expect(find.byType(WorkingChip), findsOneWidget);
    expect(find.textContaining('team is working'), findsNothing);
    expect(find.byType(SizedBox), findsWidgets);
  });

  testWidgets('WorkingChip rotates words and disposes its animation', (
    tester,
  ) async {
    await tester.pumpWidget(host(const WorkingChip(working: true)));

    expect(find.textContaining('team is working'), findsOneWidget);
    expect(find.text('${kWorkingWords.first}…'), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is Semantics &&
            widget.properties.label ==
                'Team is working — ${kWorkingWords.first}',
      ),
      findsOneWidget,
    );

    await tester.pump(const Duration(milliseconds: 1));
    await tester.pump(const Duration(seconds: 4));
    await tester.pump();
    expect(find.text('${kWorkingWords[1]}…'), findsOneWidget);

    await tester.pumpWidget(host(const SizedBox.shrink()));
    await tester.pumpAndSettle();
    expect(tester.binding.transientCallbackCount, 0);
  });
}
