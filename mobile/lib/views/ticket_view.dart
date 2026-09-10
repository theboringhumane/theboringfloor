import 'package:flutter/material.dart';

import '../api/gateway_client.dart';
import '../components/markdown_body.dart';
import '../models/floor.dart';
import '../models/project.dart';
import 'ticket_editor.dart';

class TicketView extends StatefulWidget {
  const TicketView({
    super.key,
    required this.client,
    required this.project,
    required this.ticket,
    required this.teams,
    required this.onPrepare,
    required this.onConversation,
  });
  final GatewayClient client;
  final Project project;
  final FloorTicket ticket;
  final List<FloorTeam> teams;
  final Future<void> Function(FloorTicket) onPrepare;
  final Future<void> Function(FloorConversation) onConversation;
  @override
  State<TicketView> createState() => _TicketViewState();
}

class _TicketViewState extends State<TicketView> {
  late FloorTicket _ticket = widget.ticket;
  bool _busy = false;
  String? _error;
  Future<void> _refresh() async {
    final floor = await widget.client.floor(widget.project.id);
    final found = floor.tickets.where((t) => t.id == _ticket.id).firstOrNull;
    if (found == null) throw StateError('This ticket is no longer available.');
    if (mounted) setState(() => _ticket = found);
  }

  Future<void> _run(Future<void> Function() action) async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await action();
      await _refresh();
    } catch (e) {
      if (mounted) {
        setState(
          () => _error = e is GatewayException
              ? e.message
              : e is StateError
              ? e.message
              : 'Could not update this ticket. Try again.',
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _edit() async {
    await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (_) => TicketEditor(
        ticket: _ticket,
        teams: widget.teams,
        onSave: (json) => widget.client.saveTicket(widget.project.id, json),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final t = _ticket;
    final colors = Theme.of(context).colorScheme;
    return Scaffold(
      appBar: AppBar(
        title: Text(t.id),
        actions: [
          IconButton(
            tooltip: 'Reload ticket',
            onPressed: _busy ? null : () => _run(() async {}),
            icon: const Icon(Icons.refresh),
          ),
          IconButton(
            tooltip: 'Edit ticket',
            onPressed: _busy ? null : () => _run(_edit),
            icon: const Icon(Icons.edit_outlined),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Wrap(
            spacing: 8,
            children: [
              Chip(label: Text(statusLabel(t.status))),
              Chip(label: Text(t.priority)),
              if (t.owner.isNotEmpty)
                Chip(
                  avatar: const Icon(Icons.person_outline, size: 16),
                  label: Text(t.owner),
                ),
            ],
          ),
          const SizedBox(height: 16),
          Text(t.title, style: Theme.of(context).textTheme.headlineMedium),
          if (t.description.isNotEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 20),
              child: SelectableText(t.description),
            ),
          const SizedBox(height: 16),
          Text(
            'Acceptance criteria',
            style: Theme.of(context).textTheme.titleLarge,
          ),
          const SizedBox(height: 12),
          if (t.checklist.isEmpty)
            const Text(
              'Add criteria so the team knows what a finished result looks like.',
            )
          else ...[
            LinearProgressIndicator(
              value: t.completedChecks / t.checklist.length,
              minHeight: 6,
              borderRadius: BorderRadius.circular(4),
            ),
            const SizedBox(height: 8),
            Text('${t.completedChecks} of ${t.checklist.length} checked'),
            for (final entry in t.checklist.indexed)
              CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                title: Text(entry.$2['text'] as String),
                value: entry.$2['done'] == true,
                onChanged: _busy
                    ? null
                    : (done) => _run(() async {
                        final checks = t.checklist;
                        checks[entry.$1] = {...entry.$2, 'done': done};
                        await widget.client.saveTicket(widget.project.id, {
                          ...t.json,
                          'checklist': checks,
                        });
                      }),
              ),
          ],
          const SizedBox(height: 24),
          Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: colors.surfaceContainerLow,
              borderRadius: BorderRadius.circular(20),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Delivery notes',
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 8),
                const Text(
                  'Recorded by the team. Review the conversation and evidence before accepting the work.',
                ),
                const SizedBox(height: 16),
                Text('Result', style: Theme.of(context).textTheme.titleSmall),
                const SizedBox(height: 8),
                if (t.result.isEmpty)
                  const Text('No result recorded yet.')
                else
                  MarkdownBody(markdown: t.result),
                const SizedBox(height: 16),
                Text(
                  'Verification',
                  style: Theme.of(context).textTheme.titleSmall,
                ),
                const SizedBox(height: 8),
                if (t.verification.isEmpty)
                  const Text('No verification recorded yet.')
                else
                  MarkdownBody(markdown: t.verification),
                const SizedBox(height: 12),
                TextButton.icon(
                  onPressed: _busy ? null : () => _run(_edit),
                  icon: const Icon(Icons.edit_note),
                  label: const Text('Edit delivery notes'),
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),
          if (t.session.isNotEmpty && t.backend.isNotEmpty)
            OutlinedButton.icon(
              onPressed: _busy
                  ? null
                  : () => _run(
                      () => widget.onConversation(
                        FloorConversation(
                          id: t.session,
                          backend: t.backend,
                          title: t.title,
                          team: t.team,
                        ),
                      ),
                    ),
              icon: const Icon(Icons.link),
              label: Text(
                'Open linked ${backendLabel(t.backend)} conversation',
              ),
            ),
          const SizedBox(height: 8),
          FilledButton.icon(
            onPressed: _busy ? null : () => _run(() => widget.onPrepare(t)),
            icon: const Icon(Icons.send_outlined),
            label: const Text('Prepare in live chat'),
          ),
          const SizedBox(height: 8),
          const Text(
            'Links this ticket to the active conversation and prepares a prompt for you to review and send. Ticket status stays unchanged.',
            textAlign: TextAlign.center,
          ),
          if (_busy)
            const Padding(
              padding: EdgeInsets.all(16),
              child: LinearProgressIndicator(),
            ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(top: 16),
              child: Text(_error!, style: TextStyle(color: colors.error)),
            ),
        ],
      ),
    );
  }
}
