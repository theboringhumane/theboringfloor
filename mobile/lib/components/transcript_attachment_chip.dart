import 'package:flutter/material.dart';

import '../models/transcript.dart';
import '../utils/typography.dart';

/// A metadata-only attachment marker for transcript history.
///
/// Past transcript attachments do not include bytes or a URL, so this never
/// attempts to render a thumbnail or fetch an image.
class TranscriptAttachmentChip extends StatelessWidget {
  const TranscriptAttachmentChip({super.key, required this.attachment});

  final TranscriptAttachment attachment;

  String? get _typeLabel {
    final mime = attachment.mime;
    if (mime == null || mime.isEmpty) return null;
    final type = mime.split('/').last;
    return type.isEmpty ? null : type.toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final typeLabel = _typeLabel;
    return Semantics(
      label: typeLabel == null
          ? 'Attachment ${attachment.name}'
          : 'Attachment ${attachment.name}, $typeLabel',
      child: Container(
        key: Key('transcript-attachment-chip-${attachment.name}'),
        constraints: const BoxConstraints(maxWidth: 240),
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        decoration: BoxDecoration(
          color: scheme.surfaceContainerHigh,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: scheme.outlineVariant),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.image_outlined,
              size: 16,
              color: scheme.onSurfaceVariant,
            ),
            const SizedBox(width: 6),
            Flexible(
              child: Text(
                attachment.name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: AppFonts.mono(
                  context,
                  base: theme.textTheme.labelMedium,
                ),
              ),
            ),
            if (typeLabel != null) ...[
              const SizedBox(width: 6),
              Text(
                typeLabel,
                style: AppFonts.mono(
                  context,
                  base: theme.textTheme.labelSmall?.copyWith(
                    color: scheme.onSurfaceVariant,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
