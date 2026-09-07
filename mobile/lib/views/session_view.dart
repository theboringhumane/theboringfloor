import 'package:flutter/material.dart';

import '../components/message_bubble.dart';
import '../store/session_store.dart';

class SessionView extends StatefulWidget {
  const SessionView({super.key, required this.store});
  final SessionStore store;
  @override
  State<SessionView> createState() => _SessionViewState();
}

class _SessionViewState extends State<SessionView> {
  final composer = TextEditingController();
  final scrollController = ScrollController();
  final expandedMessageIds = <String>{};
  bool initiallyAnchored = false;
  @override
  void initState() {
    super.initState();
    widget.store.addListener(_changed);
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
    if (!mounted) return;
    setState(() {});
    if (!initiallyAnchored && widget.store.data != null) {
      initiallyAnchored = true;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted && scrollController.hasClients) {
          scrollController.jumpTo(scrollController.position.maxScrollExtent);
        }
      });
    }
  }

  void _onScroll() {
    if (scrollController.hasClients && scrollController.position.pixels <= 48) {
      _loadOlderKeepingPosition();
    }
  }

  Future<void> _loadOlderKeepingPosition() async {
    final s = widget.store;
    if (s.loadingOlder || !s.hasMore || !scrollController.hasClients) return;
    final position = scrollController.position;
    final beforeOffset = position.pixels;
    final beforeExtent = position.maxScrollExtent;
    await s.loadOlder();
    if (!mounted || !scrollController.hasClients) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !scrollController.hasClients) return;
      final next = scrollController.position;
      final delta = next.maxScrollExtent - beforeExtent;
      next.jumpTo((beforeOffset + delta).clamp(0.0, next.maxScrollExtent));
    });
  }

  bool _isFinalAssistant(int index) {
    final messages = widget.store.messages;
    if (index != messages.length - 1) return false;
    final from = messages[index].from;
    return from == 'assistant' || from == 'boss';
  }

  void _toggle(String id) {
    setState(() {
      if (!expandedMessageIds.add(id)) expandedMessageIds.remove(id);
    });
  }

  @override
  Widget build(BuildContext context) {
    final s = widget.store;
    return Scaffold(
      appBar: AppBar(
        title: Text(s.project.name),
        actions: [
          IconButton(
            tooltip: 'Stop',
            onPressed: () => s.client.stop(s.project.id),
            icon: const Icon(Icons.stop_circle_outlined),
          ),
          IconButton(
            tooltip: 'New session',
            onPressed: () => s.client.startNew(s.project.id),
            icon: const Icon(Icons.add_comment_outlined),
          ),
        ],
      ),
      body: s.loading
          ? const Center(child: CircularProgressIndicator())
          : s.error != null
          ? Center(child: Text('${s.error}'))
          : Column(
              children: [
                Expanded(
                  child: ListView.builder(
                    controller: scrollController,
                    reverse: false,
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    itemCount: s.messages.length + (s.loadingOlder ? 1 : 0),
                    itemBuilder: (context, index) {
                      if (s.loadingOlder && index == 0) {
                        return const Padding(
                          padding: EdgeInsets.symmetric(vertical: 8),
                          child: Center(
                            child: SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            ),
                          ),
                        );
                      }
                      final messageIndex = index - (s.loadingOlder ? 1 : 0);
                      final message = s.messages[messageIndex];
                      return MessageBubble(
                        message: message,
                        expanded:
                            _isFinalAssistant(messageIndex) ||
                            expandedMessageIds.contains(message.id),
                        onToggle: () => _toggle(message.id),
                      );
                    },
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.all(12),
                  child: Row(
                    children: [
                      Expanded(
                        child: TextField(
                          controller: composer,
                          decoration: const InputDecoration(
                            hintText: 'Send follow-up',
                          ),
                          onSubmitted: (_) => _send(),
                        ),
                      ),
                      IconButton(
                        onPressed: _send,
                        icon: const Icon(Icons.arrow_upward),
                      ),
                    ],
                  ),
                ),
              ],
            ),
    );
  }

  void _send() {
    final value = composer.text.trim();
    if (value.isNotEmpty) {
      composer.clear();
      widget.store.send(value);
    }
  }
}
