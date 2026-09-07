import 'package:flutter/material.dart';

import '../models/activity_group.dart';
import '../utils/typography.dart';
import 'loading_snake.dart';

/// A collapsible, two-level view of one sub-agent's activity run.
class ActivityBubble extends StatelessWidget {
  const ActivityBubble({
    super.key,
    required this.group,
    required this.expanded,
    required this.onToggle,
    this.running = false,
    this.officeWorking = false,
  });

  final ActivityGroup group;
  final bool expanded;
  final VoidCallback onToggle;
  final bool running;
  final bool officeWorking;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final agentType = group.agentType;
    final roleLabel = activityRoleLabel(agentType);
    final parent = group.usesStructuredActivity
        ? _structuredParent(roleLabel, group.taskTitle, group.summary)
        : group.taskTitle ?? group.summary;
    final effectiveRunning = running || (officeWorking && group.hasRunningTool);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 2),
      child: Material(
        color: scheme.surface.withValues(alpha: 0),
        child: InkWell(
          key: Key('activity-${group.messages.first.id}'),
          onTap: onToggle,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 6),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(
                      expanded ? Icons.expand_more : Icons.chevron_right,
                      size: 18,
                      color: scheme.onSurfaceVariant,
                    ),
                    const SizedBox(width: 4),
                    Expanded(
                      child: Text(
                        parent,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: AppFonts.body(
                          context,
                          base: theme.textTheme.bodySmall?.copyWith(
                            color: scheme.onSurfaceVariant,
                          ),
                        ),
                      ),
                    ),
                    if (roleLabel != null && agentType != null) ...[
                      const SizedBox(width: 6),
                      _AgentChip(agentType: agentType),
                    ],
                  ],
                ),
                if (group.taskTitle != null)
                  Padding(
                    padding: const EdgeInsets.only(left: 22, top: 2),
                    child: Text(
                      group.summary,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: AppFonts.body(
                        context,
                        base: theme.textTheme.labelSmall?.copyWith(
                          color: scheme.onSurfaceVariant,
                        ),
                      ),
                    ),
                  ),
                if (effectiveRunning) ...[
                  const SizedBox(height: 6),
                  LoadingSnake(height: 2, color: scheme.primary),
                ],
                if (expanded &&
                    group.tools.isNotEmpty &&
                    (group.taskTitle != null ||
                        group.usesStructuredActivity)) ...[
                  const SizedBox(height: 4),
                  for (final tool in group.tools)
                    Padding(
                      padding: const EdgeInsets.only(left: 24, top: 3),
                      child: Row(
                        children: [
                          if (_isFailure(tool.state)) ...[
                            Icon(
                              Icons.error_outline,
                              key: Key('activity-tool-failure-${tool.name}'),
                              size: 14,
                              color: scheme.error,
                              semanticLabel: 'Tool failed',
                            ),
                            const SizedBox(width: 4),
                          ],
                          Flexible(
                            fit: FlexFit.loose,
                            child: Text(
                              '↳ ${tool.name}',
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: AppFonts.mono(
                                context,
                                base: theme.textTheme.bodySmall?.copyWith(
                                  color: _isFailure(tool.state)
                                      ? scheme.error
                                      : scheme.onSurfaceVariant,
                                ),
                              ),
                            ),
                          ),
                          if (tool.arguments.isNotEmpty) ...[
                            const SizedBox(width: 4),
                            Expanded(
                              child: Text(
                                tool.arguments,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: AppFonts.mono(
                                  context,
                                  base: theme.textTheme.bodySmall?.copyWith(
                                    color: _isFailure(tool.state)
                                        ? scheme.error
                                        : scheme.onSurfaceVariant,
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ],
                      ),
                    ),
                ],
                // Older worker activity has no task envelope on the wire.
                // Keep its existing detail disclosure until the server emits
                // structured task data for every worker update.
                if (expanded && group.taskTitle == null) ...[
                  const SizedBox(height: 4),
                  for (final message in group.messages)
                    Padding(
                      padding: const EdgeInsets.only(left: 24, top: 3),
                      child: Text(
                        message.text.trim().replaceAll(RegExp(r'\s+'), ' '),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: AppFonts.body(
                          context,
                          base: theme.textTheme.bodySmall?.copyWith(
                            color: scheme.onSurfaceVariant,
                          ),
                        ),
                      ),
                    ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

String _structuredParent(String? label, String? task, String summary) {
  if (label != null && task != null) return '$label Task — $task';
  if (label != null) return '$label Task';
  return summary;
}

bool _isFailure(String? state) => state == 'error' || state == 'aborted';

class _AgentChip extends StatelessWidget {
  const _AgentChip({required this.agentType});

  final String agentType;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: ShapeDecoration(
        color: scheme.secondaryContainer,
        shape: StadiumBorder(side: BorderSide(color: scheme.outlineVariant)),
      ),
      child: Text(
        '@$agentType',
        style: AppFonts.mono(
          context,
          base: theme.textTheme.labelSmall?.copyWith(
            color: scheme.onSecondaryContainer,
          ),
        ),
      ),
    );
  }
}
