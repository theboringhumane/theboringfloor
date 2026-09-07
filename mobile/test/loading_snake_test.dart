import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/components/loading_snake.dart';
import 'package:theboringfloor/theme.dart';

void main() {
  Widget host(Widget child) => MaterialApp(
    theme: buildLightTheme(),
    home: Scaffold(body: child),
  );

  testWidgets('LoadingSnake animates at narrow widths and disposes cleanly', (
    tester,
  ) async {
    await tester.pumpWidget(
      host(const SizedBox(width: 320, child: LoadingSnake())),
    );

    expect(find.byType(LoadingSnake), findsOneWidget);
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pump(const Duration(milliseconds: 400));
    expect(tester.takeException(), isNull);

    await tester.pumpWidget(host(const SizedBox.shrink()));
    await tester.pump();
    expect(tester.binding.transientCallbackCount, 0);
  });

  testWidgets('LoadingSnake has a static reduced-motion frame', (tester) async {
    await tester.pumpWidget(
      host(
        const MediaQuery(
          data: MediaQueryData(disableAnimations: true),
          child: SizedBox(width: 320, child: LoadingSnake()),
        ),
      ),
    );

    expect(find.bySemanticsLabel('Working'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
