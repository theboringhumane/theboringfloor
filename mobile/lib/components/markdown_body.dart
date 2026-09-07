import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:markdown_widget/markdown_widget.dart';

import '../utils/markdown_theme.dart';
import 'code_block.dart';

enum MarkdownBodyVariant { standard, compact, error }

/// Selectable Markdown for transcript content. Links intentionally copy only.
class MarkdownBody extends StatelessWidget {
  const MarkdownBody({
    super.key,
    required this.markdown,
    this.variant = MarkdownBodyVariant.standard,
  });

  final String markdown;
  final MarkdownBodyVariant variant;

  Future<void> _copyLink(BuildContext context, String url) async {
    await Clipboard.setData(ClipboardData(text: url));
    if (!context.mounted) return;
    ScaffoldMessenger.of(context)
        .showSnackBar(const SnackBar(content: Text('Copied link')));
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final base = markdownTheme(theme);
    final body =
        theme.textTheme.bodyLarge?.copyWith(
          color: variant == MarkdownBodyVariant.error
              ? scheme.error
              : scheme.onSurface,
          fontFamily: variant == MarkdownBodyVariant.compact
              ? 'monospace'
              : null,
        ) ??
        TextStyle(
          color: variant == MarkdownBodyVariant.error
              ? scheme.error
              : scheme.onSurface,
        );
    final config = base.copy(
      configs: [
        PConfig(textStyle: body),
        LinkConfig(
          style: body.copyWith(
            color: scheme.primary,
            decoration: TextDecoration.underline,
          ),
          onTap: (url) => _copyLink(context, url),
        ),
        PreConfig(
          builder: (code, language) =>
              MessageCodeBlock(code: code, language: language),
        ),
      ],
    );

    return MarkdownBlock(data: markdown, config: config, selectable: true);
  }
}
