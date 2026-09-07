import 'package:flutter/material.dart';
import 'package:markdown_widget/markdown_widget.dart';

import 'typography.dart';

/// Builds Markdown presentation from the active application theme.
MarkdownConfig markdownTheme(ThemeData theme, {BuildContext? context}) {
  final scheme = theme.colorScheme;
  final text = theme.textTheme;
  final body =
      text.bodyLarge?.copyWith(color: scheme.onSurface) ??
      TextStyle(color: scheme.onSurface);
  TextStyle bodyStyle(TextStyle base) =>
      context == null ? base : AppFonts.body(context, base: base);
  TextStyle headingStyle(TextStyle base) =>
      context == null ? base : AppFonts.heading(context, base: base);
  TextStyle monoStyle(TextStyle base) =>
      context == null ? base : AppFonts.mono(context, base: base);

  return MarkdownConfig(
    configs: [
      PConfig(textStyle: bodyStyle(body)),
      H1Config(
        style: headingStyle(
          (text.headlineSmall ?? body).copyWith(color: scheme.onSurface),
        ),
      ),
      H2Config(
        style: headingStyle(
          (text.titleLarge ?? body).copyWith(color: scheme.onSurface),
        ),
      ),
      H3Config(
        style: headingStyle(
          (text.titleMedium ?? body).copyWith(color: scheme.onSurface),
        ),
      ),
      H4Config(
        style: headingStyle(
          (text.titleSmall ?? body).copyWith(color: scheme.onSurface),
        ),
      ),
      H5Config(
        style: headingStyle(
          (text.titleSmall ?? body).copyWith(color: scheme.onSurface),
        ),
      ),
      H6Config(
        style: headingStyle(
          (text.labelLarge ?? body).copyWith(color: scheme.onSurface),
        ),
      ),
      CodeConfig(
        style: monoStyle(
          body.copyWith(backgroundColor: scheme.surfaceContainerHighest),
        ),
      ),
      BlockquoteConfig(
        sideColor: scheme.outlineVariant,
        textColor: scheme.onSurfaceVariant,
      ),
      TableConfig(
        border: TableBorder.all(color: scheme.outlineVariant),
        headerRowDecoration: BoxDecoration(color: scheme.surfaceContainerHigh),
        bodyRowDecoration: BoxDecoration(color: scheme.surfaceContainerLow),
        headerStyle: bodyStyle(
          (text.labelLarge ?? body).copyWith(color: scheme.onSurface),
        ),
        bodyStyle: bodyStyle(body),
      ),
      HrConfig(color: scheme.outlineVariant),
    ],
  );
}
