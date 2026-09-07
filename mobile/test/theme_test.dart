import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/theme.dart';
import 'package:theboringfloor/utils/typography.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUpAll(disableRuntimeFontFetching);

  test('seeded fallback builds valid Material 3 light and dark themes', () {
    final lightTheme = buildLightTheme();
    final darkTheme = buildDarkTheme();

    expect(lightTheme.useMaterial3, isTrue);
    expect(darkTheme.useMaterial3, isTrue);
    expect(lightTheme.colorScheme.brightness, Brightness.light);
    expect(darkTheme.colorScheme.brightness, Brightness.dark);
    expect(lightTheme.colorScheme.primary, isNot(equals(Colors.transparent)));
    expect(darkTheme.colorScheme.primary, isNot(equals(Colors.transparent)));
    expect(lightTheme.textTheme.bodyLarge!.fontFamily, 'Inter');
    expect(darkTheme.textTheme.bodyLarge!.fontFamily, 'Inter');
    expect(lightTheme.textTheme.headlineLarge!.fontFamily, 'Space Grotesk');
    expect(darkTheme.textTheme.headlineLarge!.fontFamily, 'Space Grotesk');
  });

  test('uses a supplied dynamic color scheme', () {
    final dynamicScheme = ColorScheme.fromSeed(
      seedColor: Colors.deepPurple,
      brightness: Brightness.light,
    );

    expect(buildLightTheme(dynamicScheme).colorScheme, same(dynamicScheme));
  });
}
