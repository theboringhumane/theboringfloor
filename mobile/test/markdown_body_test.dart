import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:theboringfloor/components/markdown_body.dart';
import 'package:theboringfloor/utils/markdown_theme.dart';

void main() {
  Widget subject(String markdown) => MaterialApp(
    home: Scaffold(
      body: SizedBox(width: 320, child: MarkdownBody(markdown: markdown)),
    ),
  );

  testWidgets('fenced code has its language label without overflow', (
    tester,
  ) async {
    await tester.pumpWidget(
      subject(
        '```bash\necho this-is-a-very-long-command-that-scrolls-horizontally\n```',
      ),
    );

    expect(find.text('bash'), findsOneWidget);
    expect(find.byTooltip('Copy code'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('heading and bullet list render', (tester) async {
    await tester.pumpWidget(subject('# Heading\n\n- First\n- Second'));

    expect(find.text('Heading'), findsOneWidget);
    expect(find.text('First'), findsOneWidget);
    expect(find.text('Second'), findsOneWidget);
  });

  testWidgets('table renders', (tester) async {
    await tester.pumpWidget(
      subject('| Name | Value |\n| --- | --- |\n| One | Two |'),
    );

    expect(find.text('Name'), findsOneWidget);
    expect(find.text('Two'), findsOneWidget);
  });

  testWidgets('unclosed fence and plain text render without throwing', (
    tester,
  ) async {
    await tester.pumpWidget(subject('```dart\nfinal incomplete = true;'));
    expect(tester.takeException(), isNull);

    await tester.pumpWidget(subject('ordinary paragraph text'));
    expect(find.text('ordinary paragraph text'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  test('markdown theme follows light and dark colors', () {
    final light = ThemeData(
      colorScheme: ColorScheme.fromSeed(seedColor: Colors.teal),
    );
    final dark = ThemeData(
      colorScheme: ColorScheme.fromSeed(
        seedColor: Colors.teal,
        brightness: Brightness.dark,
      ),
    );

    expect(
      markdownTheme(light).p.textStyle.color,
      isNot(markdownTheme(dark).p.textStyle.color),
    );
  });
}
