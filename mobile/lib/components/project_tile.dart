import 'package:flutter/material.dart';

import '../models/project.dart';
import '../utils/typography.dart';

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

    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
        child: Row(
          children: [
            Semantics(
              label: '$status status',
              child: Container(
                key: Key('status-dot-${project.id}'),
                width: 8,
                height: 8,
                decoration: BoxDecoration(
                  color: project.live ? colors.primary : colors.outline,
                  shape: BoxShape.circle,
                ),
              ),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    project.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: AppFonts.heading(
                      context,
                      base: text.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '$status · ${project.dir}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: text.bodyMedium?.copyWith(
                      color: colors.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
            if (trailing != null) ...[const SizedBox(width: 12), trailing!],
          ],
        ),
      ),
    );
  }
}
