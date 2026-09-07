import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../models/transcript.dart';

class MessageBubble extends StatelessWidget {
  const MessageBubble({
    super.key,
    required this.message,
    required this.expanded,
    required this.onToggle,
  });
  final TranscriptMessage message;
  final bool expanded;
  final VoidCallback onToggle;

  bool get _isThinking => message.kind == 'wthink';
  bool get _isTool => message.kind == 'wtool';

  String get _summary {
    if (_isThinking) {
      return message.text.trim().isEmpty
          ? 'Thought'
          : 'Thought · ${message.text.trim()}';
    }
    if (_isTool) {
      final line = message.text.trim();
      return line.startsWith('{') || line.startsWith('[')
          ? 'Tool activity'
          : (line.isEmpty ? 'Tool activity' : line);
    }
    final line = message.text.trim().replaceAll(RegExp(r'\s+'), ' ');
    return line.isEmpty ? 'Message' : line;
  }

  void _copy(BuildContext context) {
    Clipboard.setData(ClipboardData(text: message.text));
    ScaffoldMessenger.of(context)
        .showSnackBar(const SnackBar(content: Text('Copied message')));
  }

  @override
  Widget build(BuildContext context) {
    final mine = message.from == 'user';
    final colors = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    final content = expanded
        ? Text(message.text, style: text.bodyMedium)
        : Row(
            children: [
              Expanded(
                child: Text(
                  _summary,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: text.bodyMedium?.copyWith(
                    color: colors.onSurfaceVariant,
                  ),
                ),
              ),
              Icon(
                Icons.chevron_right,
                size: 18,
                color: colors.onSurfaceVariant,
              ),
            ],
          );
    return Align(
      alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
      child: GestureDetector(
        key: Key('message-${message.id}'),
        onTap: onToggle,
        onLongPress: () => _copy(context),
        child: Container(
          margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: mine ? colors.surfaceContainerHighest : null,
            borderRadius: mine ? BorderRadius.circular(14) : null,
          ),
          child: content,
        ),
      ),
    );
  }
}
