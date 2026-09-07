import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../models/transcript.dart';
import 'glass.dart';
import 'markdown_body.dart';

enum MessageKind { assistant, user, tool, error }

class MessageBubble extends StatefulWidget {
  const MessageBubble({
    super.key,
    required this.message,
    required this.expanded,
    required this.onToggle,
    this.kind = MessageKind.assistant,
  });

  final TranscriptMessage message;
  final bool expanded;
  final VoidCallback onToggle;

  /// Additive override for callers that know a richer message classification.
  final MessageKind kind;

  @override
  State<MessageBubble> createState() => _MessageBubbleState();
}

class _MessageBubbleState extends State<MessageBubble> {
  static const _memberPreviewLines = 5;
  bool _memberExpanded = false;

  TranscriptMessage get message => widget.message;
  bool get _isThinking => message.kind == 'wthink';

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

  @override
  Widget build(BuildContext context) {
    if (_isThinking ||
        (message.kind == 'wtool' && widget.kind == MessageKind.assistant)) {
      return _ActivityRow(
        message: message,
        expanded: widget.expanded,
        onToggle: widget.onToggle,
      );
    }

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

    return Align(
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
                child: child,
              ),
              MessageKind.tool || MessageKind.error => Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 20,
                  vertical: 6,
                ),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 520),
                  child: child,
                ),
              ),
              MessageKind.assistant => Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 20,
                  vertical: 6,
                ),
                child: child,
              ),
            },
            IconButton(
              tooltip: 'Copy message',
              onPressed: () => _copy(context),
              icon: const Icon(Icons.copy_outlined),
            ),
          ],
        ),
      ),
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
            style: theme.textTheme.labelSmall?.copyWith(
              color: scheme.onSurfaceVariant,
              fontFamily: 'monospace',
              fontWeight: FontWeight.w700,
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

class _ActivityRow extends StatelessWidget {
  const _ActivityRow({
    required this.message,
    required this.expanded,
    required this.onToggle,
  });
  final TranscriptMessage message;
  final bool expanded;
  final VoidCallback onToggle;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final summary = message.text.trim().replaceAll(RegExp(r'\s+'), ' ');
    final isThinking = message.kind == 'wthink';
    final compact = isThinking
        ? (summary.isEmpty ? 'Thinking' : 'Thinking · $summary')
        : (summary.isEmpty ? 'Tool activity' : summary);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 2),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 5),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                IconButton(
                  key: Key('activity-${message.id}'),
                  tooltip: expanded ? 'Collapse activity' : 'Expand activity',
                  onPressed: onToggle,
                  visualDensity: VisualDensity.compact,
                  icon: Icon(
                    expanded ? Icons.expand_more : Icons.chevron_right,
                    size: 18,
                    color: scheme.onSurfaceVariant,
                  ),
                ),
                if (!expanded)
                  Expanded(
                    child: Text(
                      compact,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.bodySmall
                          ?.copyWith(color: scheme.onSurfaceVariant),
                    ),
                  ),
              ],
            ),
            if (expanded) MarkdownBody(markdown: message.text),
          ],
        ),
      ),
    );
  }
}
