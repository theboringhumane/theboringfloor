import 'package:flutter/material.dart';

import '../api/gateway_client.dart';
import '../models/floor.dart';

class TicketEditor extends StatefulWidget {
  const TicketEditor({
    super.key,
    this.ticket,
    required this.teams,
    required this.onSave,
    this.initialTeam = '',
  });
  final FloorTicket? ticket;
  final List<FloorTeam> teams;
  final String initialTeam;
  final Future<void> Function(Map<String, dynamic>) onSave;
  @override
  State<TicketEditor> createState() => _TicketEditorState();
}

class _TicketEditorState extends State<TicketEditor> {
  late final _title = TextEditingController(text: widget.ticket?.title);
  late final _description = TextEditingController(
    text: widget.ticket?.description,
  );
  late final _owner = TextEditingController(text: widget.ticket?.owner);
  late final _result = TextEditingController(text: widget.ticket?.result);
  late final _verification = TextEditingController(
    text: widget.ticket?.verification,
  );
  final _check = TextEditingController();
  late String _status = widget.ticket?.status ?? 'backlog';
  late String _priority = widget.ticket?.priority ?? 'P2';
  late String _team = widget.ticket?.team ?? widget.initialTeam;
  late final List<Map<String, dynamic>> _checks =
      widget.ticket?.checklist ?? [];
  bool _saving = false;
  String? _error;
  @override
  void dispose() {
    _title.dispose();
    _description.dispose();
    _owner.dispose();
    _result.dispose();
    _verification.dispose();
    _check.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (_title.text.trim().isEmpty) {
      setState(() => _error = 'Give the ticket a title.');
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.onSave({
        ...?widget.ticket?.json,
        'title': _title.text.trim(),
        'description': _description.text.trim(),
        'owner': _owner.text.trim(),
        'result': _result.text.trim(),
        'verification': _verification.text.trim(),
        'status': _status,
        'priority': _priority,
        'team': _team,
        'checklist': _checks,
      });
      if (mounted) Navigator.pop(context, true);
    } catch (e) {
      if (mounted) {
        setState(() {
          _saving = false;
          _error = e is GatewayException
              ? e.message
              : 'Could not save. Try again.';
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) => SafeArea(
    child: SingleChildScrollView(
      padding: EdgeInsets.fromLTRB(
        24,
        24,
        24,
        24 + MediaQuery.viewInsetsOf(context).bottom,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            widget.ticket == null ? 'New ticket' : widget.ticket!.id,
            style: Theme.of(context).textTheme.headlineSmall,
          ),
          const SizedBox(height: 20),
          TextField(
            controller: _title,
            enabled: !_saving,
            decoration: const InputDecoration(labelText: 'Title'),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _description,
            enabled: !_saving,
            minLines: 3,
            maxLines: 7,
            decoration: const InputDecoration(labelText: 'Description'),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: DropdownButtonFormField<String>(
                  isExpanded: true,
                  initialValue: _status,
                  decoration: const InputDecoration(labelText: 'Status'),
                  items: ticketStatuses
                      .map(
                        (s) => DropdownMenuItem(
                          value: s,
                          child: Text(statusLabel(s)),
                        ),
                      )
                      .toList(),
                  onChanged: _saving
                      ? null
                      : (s) => setState(() => _status = s!),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: DropdownButtonFormField<String>(
                  isExpanded: true,
                  initialValue: _priority,
                  decoration: const InputDecoration(labelText: 'Priority'),
                  items: ['P0', 'P1', 'P2', 'P3']
                      .map((p) => DropdownMenuItem(value: p, child: Text(p)))
                      .toList(),
                  onChanged: _saving
                      ? null
                      : (p) => setState(() => _priority = p!),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          DropdownButtonFormField<String>(
            isExpanded: true,
            initialValue: _team,
            decoration: const InputDecoration(labelText: 'Team'),
            items: [
              const DropdownMenuItem(value: '', child: Text('General')),
              ...widget.teams.map(
                (t) => DropdownMenuItem(value: t.id, child: Text(t.name)),
              ),
            ],
            onChanged: _saving ? null : (t) => setState(() => _team = t!),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _owner,
            enabled: !_saving,
            decoration: const InputDecoration(labelText: 'Owner'),
          ),
          const SizedBox(height: 20),
          Text('Checklist', style: Theme.of(context).textTheme.titleMedium),
          ..._checks.indexed.map(
            (entry) => CheckboxListTile(
              contentPadding: EdgeInsets.zero,
              value: entry.$2['done'] == true,
              title: Text(entry.$2['text'] as String),
              onChanged: _saving
                  ? null
                  : (v) => setState(
                      () => _checks[entry.$1] = {...entry.$2, 'done': v},
                    ),
            ),
          ),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _check,
                  enabled: !_saving,
                  decoration: const InputDecoration(
                    hintText: 'Add a checklist item',
                  ),
                ),
              ),
              IconButton(
                tooltip: 'Add checklist item',
                onPressed: _saving
                    ? null
                    : () {
                        if (_check.text.trim().isNotEmpty) {
                          setState(() {
                            _checks.add({
                              'text': _check.text.trim(),
                              'done': false,
                            });
                            _check.clear();
                          });
                        }
                      },
                icon: const Icon(Icons.add),
              ),
            ],
          ),
          const SizedBox(height: 20),
          TextField(
            controller: _result,
            enabled: !_saving,
            minLines: 2,
            maxLines: 5,
            decoration: const InputDecoration(
              labelText: 'Result summary',
              hintText: 'What changed? What remains?',
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _verification,
            enabled: !_saving,
            minLines: 2,
            maxLines: 5,
            decoration: const InputDecoration(
              labelText: 'Verification notes',
              hintText: 'Checks run, outcomes, and evidence to review',
            ),
          ),
          const SizedBox(height: 8),
          const Text(
            'These are recorded notes, not automatically verified test results.',
          ),
          if (_status == 'done' && _checks.any((c) => c['done'] != true))
            const Padding(
              padding: EdgeInsets.only(top: 8),
              child: Text(
                'Some acceptance criteria are still unchecked. Review them before marking this ticket done.',
              ),
            ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Text(
                _error!,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            ),
          const SizedBox(height: 20),
          FilledButton(
            onPressed: _saving ? null : _save,
            child: Text(_saving ? 'Saving…' : 'Save ticket'),
          ),
        ],
      ),
    ),
  );
}
