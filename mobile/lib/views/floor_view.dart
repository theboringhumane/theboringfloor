import 'package:flutter/material.dart';

import '../api/gateway_client.dart';
import '../models/floor.dart';
import '../models/project.dart';
import '../models/session.dart';
import '../store/session_store.dart';
import '../hooks/use_polling.dart';
import 'session_view.dart';
import 'floor_files_view.dart';
import 'floor_plan_view.dart';
import 'ticket_editor.dart';

class FloorView extends StatefulWidget {
  const FloorView({super.key, required this.client, required this.project});
  final GatewayClient client;
  final Project project;
  @override
  State<FloorView> createState() => _FloorViewState();
}

class _FloorViewState extends State<FloorView>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs = TabController(length: 4, vsync: this);
  FloorData? _floor;
  String? _error;
  String _team = '';
  String _query = '';
  String? _seenDraft;
  bool _busy = false;
  bool _polling = false;
  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _tabs.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final floor = await widget.client.floor(widget.project.id);
      if (mounted) {
        setState(() {
          _floor = floor;
          _error = null;
        });
      }
    } catch (e) {
      if (mounted) setState(() => _error = _errorText(e));
    }
  }

  Future<bool> _pollPlan() async {
    if (_polling || _busy) return true;
    _polling = true;
    try {
      final plan = await widget.client.plan(widget.project.id);
      if (mounted &&
          plan.draft.isNotEmpty &&
          plan.draft != plan.approved &&
          _seenDraft != plan.draft) {
        _tabs.animateTo(3);
      }
      _seenDraft = plan.draft;
      return true;
    } catch (_) {
      return false;
    } finally {
      _polling = false;
    }
  }

  void _snack(Object error) {
    if (mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(_errorText(error))));
    }
  }

  Future<void> _openLive() async {
    setState(() => _busy = true);
    try {
      await _ensureLive();
      if (!mounted) return;
      await Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) =>
              SessionView(store: SessionStore(widget.client, widget.project)),
        ),
      );
      await _load();
    } catch (e) {
      _snack(e);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<Status> _ensureLive() async {
    try {
      return await widget.client.status(widget.project.id);
    } catch (_) {
      final result = await widget.client.startOffice(widget.project.id);
      if (result.outcome != StartOfficeOutcome.accepted &&
          result.outcome != StartOfficeOutcome.alreadyRunning) {
        throw StateError(result.message ?? 'Could not start this office.');
      }
      return _waitForStatus((_) => true);
    }
  }

  Future<Status> _waitForStatus(bool Function(Status) ready) async {
    final deadline = DateTime.now().add(const Duration(seconds: 20));
    while (mounted && DateTime.now().isBefore(deadline)) {
      try {
        final status = await widget.client.status(widget.project.id);
        if (ready(status)) return status;
      } catch (_) {}
      await Future<void>.delayed(const Duration(milliseconds: 400));
    }
    throw StateError(
      'The conversation is still opening. Refresh this floor in a moment.',
    );
  }

  Future<void> _launch(Map<String, dynamic> action) async {
    final old = await _ensureLive();
    await widget.client.workspaceAction(widget.project.id, {
      'action': 'conversation',
      ...action,
    });
    await _waitForStatus(
      (s) =>
          s.backend == action['backend'] &&
          s.primaryId.isNotEmpty &&
          (action['fresh'] == true
              ? (s.primaryId != old.primaryId || s.backend != old.backend)
              : s.primaryId == action['session']),
    );
  }

  Future<void> _newConversation() async {
    final floor = _floor;
    if (floor == null) return;
    final title = TextEditingController();
    String backend = backends.contains(widget.project.backend)
        ? widget.project.backend
        : 'codex';
    String team = _team;
    var saving = false;
    String? error;
    final launched = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (context) => StatefulBuilder(
        builder: (context, update) => SafeArea(
          child: SingleChildScrollView(
            padding: EdgeInsets.fromLTRB(
              24,
              24,
              24,
              24 + MediaQuery.viewInsetsOf(context).bottom,
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'New conversation',
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 8),
                Text('On ${floor.name}'),
                const SizedBox(height: 20),
                TextField(
                  controller: title,
                  enabled: !saving,
                  decoration: const InputDecoration(
                    labelText: 'Conversation title',
                    hintText: 'What are we building?',
                  ),
                ),
                const SizedBox(height: 16),
                DropdownButtonFormField<String>(
                  initialValue: backend,
                  decoration: const InputDecoration(labelText: 'Backend'),
                  items: backends
                      .map(
                        (b) => DropdownMenuItem(
                          value: b,
                          child: Text(backendLabel(b)),
                        ),
                      )
                      .toList(),
                  onChanged: saving ? null : (b) => update(() => backend = b!),
                ),
                const SizedBox(height: 16),
                DropdownButtonFormField<String>(
                  initialValue: team,
                  decoration: const InputDecoration(labelText: 'Team'),
                  items: [
                    const DropdownMenuItem(value: '', child: Text('General')),
                    ...floor.teams.map(
                      (t) => DropdownMenuItem(value: t.id, child: Text(t.name)),
                    ),
                  ],
                  onChanged: saving ? null : (t) => update(() => team = t!),
                ),
                if (error != null)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 12),
                    child: Text(
                      error!,
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                      ),
                    ),
                  ),
                const SizedBox(height: 20),
                FilledButton.icon(
                  onPressed: saving
                      ? null
                      : () async {
                          update(() {
                            saving = true;
                            error = null;
                          });
                          try {
                            await _launch({
                              'backend': backend,
                              'team': team,
                              'title': title.text.trim(),
                              'fresh': true,
                            });
                            if (context.mounted) Navigator.pop(context, true);
                          } catch (e) {
                            if (context.mounted) {
                              update(() {
                                saving = false;
                                error = _errorText(e);
                              });
                            }
                          }
                        },
                  icon: saving
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.add_comment_outlined),
                  label: Text(
                    saving ? 'Opening conversation…' : 'Start conversation',
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
    // The route's exit animation can retain its TextField for one more frame.
    await Future<void>.delayed(const Duration(milliseconds: 300));
    title.dispose();
    if (launched == true && mounted) await _openLive();
  }

  Future<void> _history(FloorConversation conversation) async {
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => Scaffold(
          appBar: AppBar(
            title: Text(conversation.title),
            actions: [
              TextButton(
                onPressed: () async {
                  if (_busy) return;
                  setState(() => _busy = true);
                  try {
                    await _launch({
                      'backend': conversation.backend,
                      'session': conversation.id,
                      'team': conversation.team,
                    });
                    if (mounted) {
                      Navigator.pop(context);
                      await _openLive();
                    }
                  } catch (e) {
                    _snack(e);
                  } finally {
                    if (mounted) setState(() => _busy = false);
                  }
                },
                child: const Text('Resume'),
              ),
            ],
          ),
          body: SessionView(
            store: SessionStore(
              widget.client,
              widget.project,
              conversation: conversation,
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _addTeam() async {
    final controller = TextEditingController();
    final name = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Add a team'),
        content: TextField(
          controller: controller,
          autofocus: true,
          decoration: const InputDecoration(labelText: 'Team name'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, controller.text.trim()),
            child: const Text('Add team'),
          ),
        ],
      ),
    );
    if (name != null && name.isNotEmpty) {
      try {
        await widget.client.addTeam(widget.project.id, name);
        await _load();
      } catch (e) {
        _snack(e);
      }
    }
  }

  Future<void> _editTicket([FloorTicket? ticket]) async {
    final floor = _floor;
    if (floor == null) return;
    final saved = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (_) => TicketEditor(
        ticket: ticket,
        teams: floor.teams,
        initialTeam: _team,
        onSave: (json) => widget.client.saveTicket(widget.project.id, json),
      ),
    );
    if (saved == true) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final floor = _floor;
    return Polling(
      interval: const Duration(seconds: 5),
      enabled: TickerMode.valuesOf(context).enabled,
      restartToken: 0,
      onTick: _pollPlan,
      child: Scaffold(
        appBar: AppBar(
          title: Text(floor?.name ?? widget.project.name),
          actions: [
            IconButton(
              tooltip: 'Refresh floor',
              onPressed: _load,
              icon: const Icon(Icons.refresh),
            ),
          ],
          bottom: TabBar(
            controller: _tabs,
            isScrollable: true,
            tabAlignment: TabAlignment.start,
            tabs: const [
              Tab(text: 'Conversations'),
              Tab(text: 'Board'),
              Tab(text: 'Files'),
              Tab(text: 'Plan'),
            ],
          ),
        ),
        body: floor == null
            ? Center(
                child: _error == null
                    ? const CircularProgressIndicator()
                    : Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Padding(
                            padding: const EdgeInsets.all(24),
                            child: Text(_error!),
                          ),
                          FilledButton(
                            onPressed: _load,
                            child: const Text('Retry'),
                          ),
                          TextButton(
                            onPressed: _busy ? null : _openLive,
                            child: const Text('Open existing chat'),
                          ),
                        ],
                      ),
              )
            : Column(
                children: [
                  if (_error != null)
                    MaterialBanner(
                      content: Text(_error!),
                      actions: [
                        TextButton(
                          onPressed: _load,
                          child: const Text('Retry'),
                        ),
                      ],
                    ),
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 12, 8, 8),
                    child: Row(
                      children: [
                        Expanded(
                          child: SingleChildScrollView(
                            scrollDirection: Axis.horizontal,
                            child: Row(
                              children: [
                                ChoiceChip(
                                  label: const Text('All teams'),
                                  selected: _team.isEmpty,
                                  onSelected: (_) => setState(() => _team = ''),
                                ),
                                const SizedBox(width: 6),
                                ...floor.teams.map(
                                  (t) => Padding(
                                    padding: const EdgeInsets.only(right: 6),
                                    child: ChoiceChip(
                                      label: Text(t.name),
                                      selected: _team == t.id,
                                      onSelected: (_) =>
                                          setState(() => _team = t.id),
                                    ),
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                        IconButton(
                          tooltip: 'Add team',
                          onPressed: _addTeam,
                          icon: const Icon(Icons.group_add_outlined),
                        ),
                      ],
                    ),
                  ),
                  Expanded(
                    child: TabBarView(
                      controller: _tabs,
                      children: [
                        _conversations(floor),
                        _board(floor),
                        FloorFilesView(
                          client: widget.client,
                          project: widget.project,
                        ),
                        FloorPlanView(
                          client: widget.client,
                          project: widget.project,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
      ),
    );
  }

  Widget _conversations(FloorData floor) {
    final rows = floor.conversations
        .where(
          (c) =>
              (_team.isEmpty || c.team == _team) &&
              (_query.isEmpty ||
                  c.title.toLowerCase().contains(_query.toLowerCase())),
        )
        .toList();
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: _busy ? null : _openLive,
                  icon: const Icon(Icons.forum_outlined),
                  label: const Text('Open live chat'),
                ),
              ),
              const SizedBox(width: 8),
              OutlinedButton.icon(
                onPressed: _busy ? null : _newConversation,
                icon: const Icon(Icons.add),
                label: const Text('New'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          TextField(
            onChanged: (q) => setState(() => _query = q),
            decoration: const InputDecoration(
              hintText: 'Find a conversation',
              prefixIcon: Icon(Icons.search),
            ),
          ),
          const SizedBox(height: 16),
          if (rows.isEmpty)
            const Padding(
              padding: EdgeInsets.all(32),
              child: Text(
                'No conversations here yet. Start one with the backend and team you need.',
              ),
            ),
          ...rows.map(
            (c) => Card(
              margin: const EdgeInsets.only(bottom: 8),
              child: ListTile(
                leading: const Icon(Icons.chat_bubble_outline),
                title: Text(
                  c.title,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Text(
                  '${backendLabel(c.backend)} · ${c.messages} messages',
                ),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => _history(c),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _board(FloorData floor) => Column(
    children: [
      Padding(
        padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
        child: Row(
          children: [
            Text(
              '${floor.tickets.where((t) => _team.isEmpty || t.team == _team).length} tickets',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const Spacer(),
            FilledButton.icon(
              onPressed: () => _editTicket(),
              icon: const Icon(Icons.add),
              label: const Text('Ticket'),
            ),
          ],
        ),
      ),
      Expanded(
        child: ListView(
          scrollDirection: Axis.horizontal,
          padding: const EdgeInsets.fromLTRB(16, 4, 16, 16),
          children: ticketStatuses.map((status) {
            final tickets = floor.tickets
                .where(
                  (t) =>
                      t.status == status && (_team.isEmpty || t.team == _team),
                )
                .toList();
            return Container(
              width: 280,
              margin: const EdgeInsets.only(right: 12),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.surfaceContainerLow,
                borderRadius: BorderRadius.circular(20),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    '${statusLabel(status)}  ${tickets.length}',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  const SizedBox(height: 12),
                  Expanded(
                    child: ListView(
                      children: [
                        if (tickets.isEmpty)
                          const Padding(
                            padding: EdgeInsets.all(16),
                            child: Text('No tickets'),
                          ),
                        ...tickets.map(
                          (t) => Card(
                            margin: const EdgeInsets.only(bottom: 10),
                            child: ListTile(
                              onTap: () => _editTicket(t),
                              title: Text(t.title),
                              subtitle: Text(
                                '${t.id} · ${t.priority}${t.owner.isEmpty ? '' : ' · ${t.owner}'}',
                              ),
                              trailing: const Icon(
                                Icons.edit_outlined,
                                size: 18,
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            );
          }).toList(),
        ),
      ),
    ],
  );
}

String _errorText(Object error) => error is GatewayException
    ? (error.statusCode == 404
          ? 'Update the gateway and office to use floor workspaces.'
          : error.message)
    : error is StateError
    ? error.message
    : 'Could not connect. Check the gateway and try again.';
