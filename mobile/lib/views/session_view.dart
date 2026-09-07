import 'package:flutter/material.dart';

import '../components/attachment_chip.dart';
import '../components/attachment_picker.dart';
import '../components/glass.dart';
import '../components/message_bubble.dart';
import '../components/working_chip.dart';
import '../hooks/use_polling.dart';
import '../models/models.dart';
import '../store/session_store.dart';

class SessionView extends StatefulWidget {
  const SessionView({super.key, required this.store, this.attachmentPicker});
  final SessionStore store;
  final AttachmentPicker? attachmentPicker;
  @override
  State<SessionView> createState() => _SessionViewState();
}

class _SessionViewState extends State<SessionView> {
  final composer = TextEditingController();
  final scrollController = ScrollController();
  final expandedActivityIds = <String>{};
  final attachments = <Attachment>[];
  String? attachmentError;
  static const _newestEndThreshold = 80.0;
  bool _atNewestEnd = true;
  bool _hasNewerContent = false;
  bool _scrollAnimationInFlight = false;
  bool _scrollAgain = false;
  int _seenNewestMessageGeneration = 0;

  AttachmentPicker get _attachmentPicker =>
      widget.attachmentPicker ?? ImagePickerAttachmentPicker();

  @override
  void initState() {
    super.initState();
    widget.store.addListener(_changed);
    _seenNewestMessageGeneration = widget.store.newestMessageGeneration;
    scrollController.addListener(_onScroll);
    widget.store.load();
  }

  @override
  void dispose() {
    widget.store.removeListener(_changed);
    composer.dispose();
    scrollController
      ..removeListener(_onScroll)
      ..dispose();
    super.dispose();
  }

  void _changed() {
    if (!mounted) {
      return;
    }
    final newestContentArrived =
        widget.store.newestMessageGeneration != _seenNewestMessageGeneration;
    if (newestContentArrived) {
      _seenNewestMessageGeneration = widget.store.newestMessageGeneration;
      // The store notifies before this rebuild inserts the new content, so this
      // remains the member's pre-insert position.
      if (_atNewestEnd) {
        _hasNewerContent = false;
        _scrollToLatest();
      } else {
        _hasNewerContent = true;
      }
    }
    setState(() {});
  }

  void _onScroll() {
    if (!scrollController.hasClients) {
      return;
    }
    final atNewestEnd = scrollController.offset <= _newestEndThreshold;
    if (_atNewestEnd != atNewestEnd || (atNewestEnd && _hasNewerContent)) {
      setState(() {
        _atNewestEnd = atNewestEnd;
        if (atNewestEnd) {
          _hasNewerContent = false;
        }
      });
    }
    if (scrollController.hasClients &&
        scrollController.position.maxScrollExtent -
                scrollController.position.pixels <=
            48) {
      _loadOlder();
    }
  }

  Future<void> _scrollToLatest() async {
    if (!scrollController.hasClients) {
      return;
    }
    if (_scrollAnimationInFlight) {
      _scrollAgain = true;
      return;
    }

    _scrollAnimationInFlight = true;
    try {
      await scrollController.animateTo(
        0,
        duration: const Duration(milliseconds: 240),
        curve: Curves.easeOut,
      );
    } finally {
      _scrollAnimationInFlight = false;
      final scrollAgain = _scrollAgain;
      _scrollAgain = false;
      if (mounted && scrollController.hasClients && scrollAgain) {
        await _scrollToLatest();
      }
    }
  }

  void _jumpToLatest() {
    if (!scrollController.hasClients) {
      return;
    }
    setState(() => _hasNewerContent = false);
    _scrollToLatest();
  }

  Future<void> _loadOlder() async {
    final store = widget.store;
    if (store.loadingOlder || !store.hasMore || !scrollController.hasClients) {
      return;
    }
    // In a reversed list, older pages grow at the far end. The framework keeps
    // the current viewport stable, so no offset compensation is necessary.
    await store.loadOlder();
  }

  void _toggleActivity(String id) {
    setState(() {
      if (!expandedActivityIds.add(id)) {
        expandedActivityIds.remove(id);
      }
    });
  }

  Future<void> _chooseAttachmentSource() async {
    final source = await showModalBottomSheet<AttachmentPickSource>(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.photo_library_outlined),
              title: const Text('Photo library'),
              onTap: () =>
                  Navigator.pop(context, AttachmentPickSource.photoLibrary),
            ),
            ListTile(
              leading: const Icon(Icons.photo_camera_outlined),
              title: const Text('Camera'),
              onTap: () => Navigator.pop(context, AttachmentPickSource.camera),
            ),
          ],
        ),
      ),
    );
    if (source == null || !mounted) {
      return;
    }
    try {
      final picked = await _attachmentPicker.pick(source);
      if (!mounted || picked.isEmpty) {
        return;
      }
      AttachmentValidator.validate([...attachments, ...picked]);
      setState(() {
        attachments.addAll(picked);
        attachmentError = null;
      });
    } on AttachmentValidationException catch (error) {
      if (mounted) {
        setState(() => attachmentError = error.message);
      }
    }
  }

  Future<void> _showOverflowMenu() async {
    final action = await showModalBottomSheet<_SessionAction>(
      context: context,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.stop_circle_outlined),
              title: const Text('Stop'),
              onTap: () => Navigator.pop(context, _SessionAction.stop),
            ),
            ListTile(
              leading: const Icon(Icons.add_comment_outlined),
              title: const Text('New session'),
              onTap: () => Navigator.pop(context, _SessionAction.newSession),
            ),
          ],
        ),
      ),
    );
    if (action == _SessionAction.stop) {
      widget.store.client.stop(widget.store.project.id);
    }
    if (action == _SessionAction.newSession) {
      widget.store.client.startNew(widget.store.project.id);
    }
  }

  @override
  Widget build(BuildContext context) {
    final store = widget.store;
    // Keep the store and grouping algorithm chronological; reverse only this
    // presentation list so index zero is the newest transcript entry.
    final entries = groupTranscript(store.messages).reversed
        .toList(growable: false);
    return Polling(
      interval: sessionRefreshInterval,
      enabled: store.isWorking && TickerMode.valuesOf(context).enabled,
      restartToken: store.pollGeneration,
      onTick: store.refresh,
      child: Scaffold(
        body: store.loading
            ? const Center(child: CircularProgressIndicator())
            : store.error != null
            ? Center(child: Text('${store.error}'))
            : SafeArea(
                child: Stack(
                  children: [
                    Column(
                      children: [
                        Expanded(
                          child: ListView.builder(
                            controller: scrollController,
                            reverse: true,
                            padding: const EdgeInsets.only(
                              top: 64,
                              bottom: 96,
                              right: 10,
                            ),
                            itemCount:
                                entries.length + (store.loadingOlder ? 1 : 0),
                            itemBuilder: (context, index) {
                              if (store.loadingOlder &&
                                  index == entries.length) {
                                return const Padding(
                                  padding: EdgeInsets.symmetric(vertical: 8),
                                  child: Center(
                                    child: SizedBox(
                                      width: 18,
                                      height: 18,
                                      child: CircularProgressIndicator(
                                        strokeWidth: 2,
                                      ),
                                    ),
                                  ),
                                );
                              }
                              final entry = entries[index];
                              return switch (entry) {
                                MessageEntry(:final message) => MessageBubble(
                                  message: message,
                                  expanded: false,
                                  onToggle: () {},
                                ),
                                ActivityEntry(:final group) => ActivityGroupRow(
                                  group: group,
                                  expanded: expandedActivityIds.contains(
                                    group.messages.first.id,
                                  ),
                                  onToggle: () =>
                                      _toggleActivity(group.messages.first.id),
                                ),
                              };
                            },
                          ),
                        ),
                      ],
                    ),
                    Positioned(
                      top: 8,
                      left: 12,
                      child: GlassIconButton(
                        icon: Icons.chevron_left,
                        tooltip: 'Back',
                        intensity: GlassIntensity.regular,
                        onPressed: () => Navigator.maybePop(context),
                      ),
                    ),
                    Positioned(
                      top: 8,
                      right: 12,
                      child: GlassIconButton(
                        icon: Icons.more_horiz,
                        tooltip: 'More',
                        intensity: GlassIntensity.regular,
                        onPressed: _showOverflowMenu,
                      ),
                    ),
                    Positioned(
                      left: 16,
                      bottom: 92,
                      child: WorkingChip(working: store.isWorking),
                    ),
                    if (_hasNewerContent && !_atNewestEnd)
                      Positioned(
                        right: 16,
                        bottom: 148,
                        child: FilledButton.icon(
                          key: const Key('jump-to-latest'),
                          onPressed: _jumpToLatest,
                          icon: const Icon(Icons.arrow_downward),
                          label: const Text('Jump to latest'),
                        ),
                      ),
                    Positioned(
                      left: 16,
                      right: 16,
                      bottom: 12,
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          if (attachments.isNotEmpty)
                            Padding(
                              padding: const EdgeInsets.only(bottom: 8),
                              child: SizedBox(
                                height: 58,
                                child: ListView.separated(
                                  scrollDirection: Axis.horizontal,
                                  itemCount: attachments.length,
                                  separatorBuilder: (_, _) =>
                                      const SizedBox(width: 8),
                                  itemBuilder: (context, index) =>
                                      AttachmentChip(
                                        attachment: attachments[index],
                                        onRemove: widget.store.sending
                                            ? () {}
                                            : () => setState(
                                                () =>
                                                    attachments.removeAt(index),
                                              ),
                                      ),
                                ),
                              ),
                            ),
                          if (attachmentError != null ||
                              store.sendError != null)
                            Padding(
                              padding: const EdgeInsets.only(bottom: 6),
                              child: Row(
                                children: [
                                  Expanded(
                                    child: Text(
                                      attachmentError ?? store.sendError!,
                                    ),
                                  ),
                                  if (store.sendError != null)
                                    TextButton(
                                      onPressed: store.sending ? null : _send,
                                      child: const Text('Try again'),
                                    ),
                                ],
                              ),
                            ),
                          GlassPill(
                            intensity: GlassIntensity.strong,
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 4,
                            ),
                            child: Row(
                              children: [
                                IconButton(
                                  tooltip: 'Add attachment',
                                  onPressed: widget.store.sending
                                      ? null
                                      : _chooseAttachmentSource,
                                  icon: const Icon(Icons.add_circle_outline),
                                ),
                                Expanded(
                                  child: TextField(
                                    controller: composer,
                                    enabled: !widget.store.sending,
                                    decoration: const InputDecoration(
                                      border: InputBorder.none,
                                      hintText: 'Follow up…',
                                    ),
                                    onChanged: (_) => setState(() {}),
                                    onSubmitted: (_) => _send(),
                                  ),
                                ),
                                widget.store.sending
                                    ? const Padding(
                                        padding: EdgeInsets.all(12),
                                        child: SizedBox(
                                          width: 22,
                                          height: 22,
                                          child: CircularProgressIndicator(
                                            strokeWidth: 2,
                                          ),
                                        ),
                                      )
                                    : IconButton(
                                        tooltip: 'Send follow-up',
                                        onPressed: _canSend ? _send : null,
                                        icon: Icon(
                                          Icons.arrow_circle_right_rounded,
                                          color: _canSend ? Colors.blue : null,
                                        ),
                                      ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
      ),
    );
  }

  bool get _canSend =>
      composer.text.trim().isNotEmpty || attachments.isNotEmpty;

  Future<void> _send() async {
    final value = composer.text.trim();
    if (_canSend && !widget.store.sending) {
      final sent = await widget.store.send(
        value,
        attachments: List.of(attachments),
      );
      if (!mounted) {
        return;
      }
      if (sent) {
        setState(() {
          attachments.clear();
          attachmentError = null;
        });
        composer.clear();
      }
    }
  }
}

enum _SessionAction { stop, newSession }
