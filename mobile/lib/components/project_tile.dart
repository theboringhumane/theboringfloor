import 'package:flutter/material.dart';

import '../models/project.dart';
import '../utils/time.dart';

class ProjectTile extends StatelessWidget {
  const ProjectTile({
    super.key,
    required this.project,
    required this.onTap,
    this.trailing,
  });
  final Project project;
  final VoidCallback onTap;
  final Widget? trailing;
  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    final status = project.live ? 'LIVE' : 'STOPPED';
    final statusBadge =
        trailing ??
        Chip(
          label: Text(status),
          labelStyle: text.labelSmall?.copyWith(
            color: project.live
                ? colors.onPrimaryContainer
                : colors.onSurfaceVariant,
          ),
          backgroundColor: project.live
              ? colors.primaryContainer
              : colors.surfaceContainerHighest,
          side: BorderSide.none,
          visualDensity: VisualDensity.compact,
          padding: EdgeInsets.zero,
        );

    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            CircleAvatar(
              backgroundColor: colors.secondaryContainer,
              foregroundColor: colors.onSecondaryContainer,
              child: Text(
                project.name.isEmpty ? '?' : project.name[0].toUpperCase(),
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.only(top: 2),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      project.name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: text.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      project.dir,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: text.bodyMedium?.copyWith(
                        color: colors.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(width: 12),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  formatRelativeTime(
                    DateTime.fromMillisecondsSinceEpoch(project.savedAt),
                  ),
                  style: text.labelMedium?.copyWith(
                    color: colors.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 8),
                statusBadge,
              ],
            ),
          ],
        ),
      ),
    );
  }
}
