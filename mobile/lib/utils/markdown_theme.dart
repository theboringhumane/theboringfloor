import 'package:flutter/material.dart';
import 'package:markdown_widget/markdown_widget.dart';

/// Builds Markdown presentation from the active application theme.
MarkdownConfig markdownTheme(ThemeData theme) {
  final scheme = theme.colorScheme;
  final text = theme.textTheme;
  final body =
      text.bodyLarge?.copyWith(color: scheme.onSurface) ??
      TextStyle(color: scheme.onSurface);

  return MarkdownConfig(
    configs: [
      PConfig(textStyle: body),
      H1Config(
        style: (text.headlineSmall ?? body).copyWith(color: scheme.onSurface),
      ),
      H2Config(
        style: (text.titleLarge ?? body).copyWith(color: scheme.onSurface),
      ),
      H3Config(
        style: (text.titleMedium ?? body).copyWith(color: scheme.onSurface),
      ),
      H4Config(
        style: (text.titleSmall ?? body).copyWith(color: scheme.onSurface),
      ),
      H5Config(
        style: (text.titleSmall ?? body).copyWith(color: scheme.onSurface),
      ),
      H6Config(
        style: (text.labelLarge ?? body).copyWith(color: scheme.onSurface),
      ),
      CodeConfig(
        style: body.copyWith(
          fontFamily: 'monospace',
          backgroundColor: scheme.surfaceContainerHighest,
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
        headerStyle: (text.labelLarge ?? body).copyWith(
          color: scheme.onSurface,
        ),
        bodyStyle: body,
      ),
      HrConfig(color: scheme.outlineVariant),
    ],
  );
}
