import 'package:flutter/material.dart';

import '../utils/typography.dart';

class EmptyState extends StatelessWidget {
  const EmptyState(this.message, {super.key});
  final String message;
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Text(
          message,
          textAlign: TextAlign.center,
          style: AppFonts.heading(
            context,
            base: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ),
      ),
    );
  }
}
