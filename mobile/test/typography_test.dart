import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/utils/typography.dart';

void main() {
  setUpAll(disableRuntimeFontFetching);

  testWidgets('AppFonts helpers apply their families over the supplied base', (
    tester,
  ) async {
    const base = TextStyle(
      color: Colors.deepPurple,
      fontSize: 27,
      fontWeight: FontWeight.w700,
    );
    late BuildContext context;

    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (value) {
            context = value;
            return const SizedBox();
          },
        ),
      ),
    );

    final styles = <TextStyle>[
      AppFonts.body(context, base: base),
      AppFonts.heading(context, base: base),
      AppFonts.serif(context, base: base),
      AppFonts.mono(context, base: base),
    ];
    const expectedFamilies = <String>[
      'Inter',
      'Space Grotesk',
      'Playfair Display',
      'JetBrains Mono',
    ];

    for (var index = 0; index < styles.length; index++) {
      final style = styles[index];
      expect(style.fontFamily, expectedFamilies[index]);
      expect(style.fontSize, base.fontSize);
      expect(style.color, base.color);
      expect(style.fontWeight, base.fontWeight);
    }
  });

  test('buildAppTextTheme maps slots without changing size or color', () {
    final source = ThemeData.light().textTheme.copyWith(
      displayLarge: const TextStyle(fontSize: 31, color: Colors.red),
      headlineLarge: const TextStyle(fontSize: 29, color: Colors.orange),
      titleLarge: const TextStyle(fontSize: 25, color: Colors.yellow),
      bodyLarge: const TextStyle(fontSize: 21, color: Colors.green),
      labelLarge: const TextStyle(fontSize: 17, color: Colors.blue),
    );
    final result = buildAppTextTheme(source);

    final headingSlots = <TextStyle?>[
      result.displayLarge,
      result.displayMedium,
      result.displaySmall,
      result.headlineLarge,
      result.headlineMedium,
      result.headlineSmall,
      result.titleLarge,
      result.titleMedium,
      result.titleSmall,
    ];
    final bodySlots = <TextStyle?>[
      result.bodyLarge,
      result.bodyMedium,
      result.bodySmall,
      result.labelLarge,
      result.labelMedium,
      result.labelSmall,
    ];

    for (final style in headingSlots) {
      expect(style!.fontFamily, 'Space Grotesk');
    }
    for (final style in bodySlots) {
      expect(style!.fontFamily, 'Inter');
    }
    expect(result.displayLarge!.fontSize, source.displayLarge!.fontSize);
    expect(result.displayLarge!.color, source.displayLarge!.color);
    expect(result.headlineLarge!.fontSize, source.headlineLarge!.fontSize);
    expect(result.headlineLarge!.color, source.headlineLarge!.color);
    expect(result.titleLarge!.fontSize, source.titleLarge!.fontSize);
    expect(result.titleLarge!.color, source.titleLarge!.color);
    expect(result.bodyLarge!.fontSize, source.bodyLarge!.fontSize);
    expect(result.bodyLarge!.color, source.bodyLarge!.color);
    expect(result.labelLarge!.fontSize, source.labelLarge!.fontSize);
    expect(result.labelLarge!.color, source.labelLarge!.color);
  });

  test('disableRuntimeFontFetching remains a callable compatibility no-op', () {
    expect(disableRuntimeFontFetching, returnsNormally);
  });
}
