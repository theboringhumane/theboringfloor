import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../utils/typography.dart';

/// A horizontally scrollable, selectable fenced-code presentation.
class MessageCodeBlock extends StatelessWidget {
  const MessageCodeBlock({
    super.key,
    required this.code,
    required this.language,
  });

  final String code;
  final String language;

  Future<void> _copy(BuildContext context) async {
    await Clipboard.setData(ClipboardData(text: code));
    if (!context.mounted) return;
    ScaffoldMessenger.of(context)
        .showSnackBar(const SnackBar(content: Text('Copied code')));
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final label = language.trim().isEmpty ? 'text' : language.trim();
    return Container(
      key: const Key('markdown-code-block'),
      margin: const EdgeInsets.symmetric(vertical: 8),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHigh,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: scheme.outlineVariant),
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(left: 12, right: 4, top: 4),
            child: Row(
              children: [
                Text(
                  label,
                  style: AppFonts.mono(
                    context,
                    base: theme.textTheme.labelSmall?.copyWith(
                      color: scheme.onSurfaceVariant,
                    ),
                  ),
                ),
                const Spacer(),
                IconButton(
                  tooltip: 'Copy code',
                  onPressed: () => _copy(context),
                  icon: const Icon(Icons.copy_outlined, size: 18),
                ),
              ],
            ),
          ),
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
            child: SelectionArea(
              child: Text(
                code.trimRight(),
                style: AppFonts.mono(
                  context,
                  base: theme.textTheme.bodyMedium?.copyWith(
                    color: scheme.onSurface,
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
