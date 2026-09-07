import 'dart:async';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../models/activity_group.dart';
import '../models/transcript.dart';
import '../utils/typography.dart';
import 'glass.dart';
import 'activity_bubble.dart';
import 'markdown_body.dart';
import 'transcript_attachment_chip.dart';

enum MessageKind { assistant, user, tool, error }

/// How far a finger may drift during a press-and-hold before it counts as a
/// scroll rather than a copy gesture. Matches Flutter's own touch slop.
const double _copySlop = kTouchSlop;

class MessageBubble extends StatefulWidget {
  const MessageBubble({
    super.key,
    required this.message,
    required this.expanded,
    required this.onToggle,
    this.kind = MessageKind.assistant,
    this.attachments,
  });

  final TranscriptMessage message;
  final bool expanded;
  final VoidCallback onToggle;

  /// Additive override for callers that know a richer message classification.
  final MessageKind kind;

  /// Optional caller override while transcript rows migrate to attachment data.
  final List<TranscriptAttachment>? attachments;

  @override
  State<MessageBubble> createState() => _MessageBubbleState();
}

class _MessageBubbleState extends State<MessageBubble> {
  static const _memberPreviewLines = 5;
  bool _memberExpanded = false;
  Timer? _copyTimer;
  Offset? _copyOrigin;

  TranscriptMessage get message => widget.message;
  MessageKind get _kind {
    if (widget.kind != MessageKind.assistant) return widget.kind;
    if (message.kind == 'wtool') return MessageKind.tool;
    if (message.from == 'user') return MessageKind.user;
    return MessageKind.assistant;
  }

  Future<void> _copy(BuildContext context) async {
    await Clipboard.setData(ClipboardData(text: message.text));
    if (!context.mounted) return;
    ScaffoldMessenger.of(context)
        .showSnackBar(const SnackBar(content: Text('Copied message')));
  }

  void _startCopyTimer(BuildContext context) {
    _copyTimer?.cancel();
    _copyTimer = Timer(kLongPressTimeout, () => _copy(context));
  }

  void _cancelCopyTimer() {
    _copyTimer?.cancel();
    _copyTimer = null;
    _copyOrigin = null;
  }

  @override
  void dispose() {
    _cancelCopyTimer();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final kind = _kind;
    final child = switch (kind) {
      MessageKind.user => _MemberMessage(
        text: message.text,
        expanded: _memberExpanded,
        onToggle: () => setState(() => _memberExpanded = !_memberExpanded),
      ),
      MessageKind.assistant => MarkdownBody(markdown: message.text),
      MessageKind.tool => _ToolMessage(text: message.text),
      MessageKind.error => _ErrorMessage(text: message.text),
    };
    final attachments = widget.attachments ?? message.attachments;
    final messageContent = attachments.isEmpty
        ? child
        : Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: [
                  for (final attachment in attachments)
                    TranscriptAttachmentChip(attachment: attachment),
                ],
              ),
              const SizedBox(height: 8),
              child,
            ],
          );

    final messageRow = Align(
      alignment: kind == MessageKind.user
          ? Alignment.centerRight
          : Alignment.centerLeft,
      child: Semantics(
        key: Key('message-${message.id}'),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: kind == MessageKind.user
              ? CrossAxisAlignment.end
              : CrossAxisAlignment.start,
          children: [
            switch (kind) {
              MessageKind.user => GlassSurface(
                padding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 12,
                ),
                borderRadius: BorderRadius.circular(22),
                intensity: GlassIntensity.regular,
                child: messageContent,
              ),
              MessageKind.tool || MessageKind.error => Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 20,
                  vertical: 6,
                ),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 520),
                  child: messageContent,
                ),
              ),
              MessageKind.assistant => Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 20,
                  vertical: 6,
                ),
                child: messageContent,
              ),
            },
          ],
        ),
      ),
    );

    if (kind != MessageKind.user && kind != MessageKind.assistant) {
      return messageRow;
    }
    return Listener(
      onPointerDown: (event) {
        _copyOrigin = event.position;
        _startCopyTimer(context);
      },
      // A finger that travels is a scroll, not a long press. Without this the
      // copy fires mid-scroll whenever a drag outlasts kLongPressTimeout.
      onPointerMove: (event) {
        final origin = _copyOrigin;
        if (origin == null) return;
        if ((event.position - origin).distance > _copySlop) _cancelCopyTimer();
      },
      onPointerUp: (_) => _cancelCopyTimer(),
      onPointerCancel: (_) => _cancelCopyTimer(),
      child: messageRow,
    );
  }
}

class _MemberMessage extends StatelessWidget {
  const _MemberMessage({
    required this.text,
    required this.expanded,
    required this.onToggle,
  });

  final String text;
  final bool expanded;
  final VoidCallback onToggle;

  @override
  Widget build(BuildContext context) => ConstrainedBox(
    constraints: const BoxConstraints(maxWidth: 340),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (expanded || !_hasMoreThanPreview(text))
          MarkdownBody(markdown: text)
        else
          Text(
            text,
            maxLines: _MessageBubbleState._memberPreviewLines,
            overflow: TextOverflow.ellipsis,
            style: Theme.of(context).textTheme.bodyLarge,
          ),
        if (_hasMoreThanPreview(text))
          TextButton(
            key: Key('member-expand-$text'),
            onPressed: onToggle,
            style: TextButton.styleFrom(
              padding: const EdgeInsets.only(top: 4),
              minimumSize: Size.zero,
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            ),
            child: Text(expanded ? 'Show less' : 'Read more'),
          ),
      ],
    ),
  );

  bool _hasMoreThanPreview(String value) =>
      value.length > 220 ||
      value.split('\n').length > _MessageBubbleState._memberPreviewLines;
}

class _ToolMessage extends StatelessWidget {
  const _ToolMessage({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Container(
      key: const Key('tool-message'),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHigh,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: scheme.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'tool',
            style: AppFonts.heading(
              context,
              base: theme.textTheme.labelSmall?.copyWith(
                color: scheme.onSurfaceVariant,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          const SizedBox(height: 6),
          MarkdownBody(markdown: text, variant: MarkdownBodyVariant.compact),
        ],
      ),
    );
  }
}

class _ErrorMessage extends StatelessWidget {
  const _ErrorMessage({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      key: const Key('error-message'),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: scheme.errorContainer,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: scheme.error),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.error_outline,
            color: scheme.error,
            semanticLabel: 'Error',
          ),
          const SizedBox(width: 8),
          Expanded(
            child: MarkdownBody(
              markdown: text,
              variant: MarkdownBodyVariant.error,
            ),
          ),
        ],
      ),
    );
  }
}

/// A quiet disclosure row for a consecutive run of agent work.
class ActivityGroupRow extends StatelessWidget {
  const ActivityGroupRow({
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
  Widget build(BuildContext context) => ActivityBubble(
    group: group,
    expanded: expanded,
    onToggle: onToggle,
    running: running,
    officeWorking: officeWorking,
  );
}
