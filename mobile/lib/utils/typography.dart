import 'package:flutter/material.dart';

/// Named font families used throughout the app.
class AppFonts {
  /// Inter — default body text.
  static TextStyle body(BuildContext context, {TextStyle? base}) =>
      _withFamily(base ?? DefaultTextStyle.of(context).style, 'Inter')!;

  /// Space Grotesk — headings, titles, section labels.
  static TextStyle heading(BuildContext context, {TextStyle? base}) =>
      _withFamily(base ?? DefaultTextStyle.of(context).style, 'Space Grotesk')!;

  /// Playfair Display — reserved for hero/elegant moments only.
  static TextStyle serif(BuildContext context, {TextStyle? base}) =>
      _withFamily(
        base ?? DefaultTextStyle.of(context).style,
        'Playfair Display',
      )!;

  /// JetBrains Mono — code blocks, terminal output, inline code.
  static TextStyle mono(BuildContext context, {TextStyle? base}) => _withFamily(
    base ?? DefaultTextStyle.of(context).style,
    'JetBrains Mono',
  )!;
}

/// Applies Inter to body styles and Space Grotesk to display/headline/title styles.
TextTheme buildAppTextTheme(TextTheme base) => base.copyWith(
  displayLarge: _withFamily(base.displayLarge, 'Space Grotesk'),
  displayMedium: _withFamily(base.displayMedium, 'Space Grotesk'),
  displaySmall: _withFamily(base.displaySmall, 'Space Grotesk'),
  headlineLarge: _withFamily(base.headlineLarge, 'Space Grotesk'),
  headlineMedium: _withFamily(base.headlineMedium, 'Space Grotesk'),
  headlineSmall: _withFamily(base.headlineSmall, 'Space Grotesk'),
  titleLarge: _withFamily(base.titleLarge, 'Space Grotesk'),
  titleMedium: _withFamily(base.titleMedium, 'Space Grotesk'),
  titleSmall: _withFamily(base.titleSmall, 'Space Grotesk'),
  bodyLarge: _withFamily(base.bodyLarge, 'Inter'),
  bodyMedium: _withFamily(base.bodyMedium, 'Inter'),
  bodySmall: _withFamily(base.bodySmall, 'Inter'),
  labelLarge: _withFamily(base.labelLarge, 'Inter'),
  labelMedium: _withFamily(base.labelMedium, 'Inter'),
  labelSmall: _withFamily(base.labelSmall, 'Inter'),
);

/// Retained for test compatibility; asset fonts never fetch at runtime.
void disableRuntimeFontFetching() {}

TextStyle? _withFamily(TextStyle? base, String family) {
  if (base == null) {
    return null;
  }
  return base.copyWith(fontFamily: family);
}
