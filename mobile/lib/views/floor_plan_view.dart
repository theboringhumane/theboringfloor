import 'package:flutter/material.dart';

import '../api/gateway_client.dart';
import '../components/markdown_body.dart';
import '../hooks/use_polling.dart';
import '../models/floor.dart';
import '../models/project.dart';

class FloorPlanView extends StatefulWidget {
  const FloorPlanView({super.key, required this.client, required this.project});
  final GatewayClient client;
  final Project project;
  @override
  State<FloorPlanView> createState() => _FloorPlanViewState();
}

class _FloorPlanViewState extends State<FloorPlanView> {
  final _editor = TextEditingController();
  FloorPlan? _plan;
  String _source = '';
  String? _error;
  bool _editing = false, _dirty = false, _saving = false, _loading = false;
  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _editor.dispose();
    super.dispose();
  }

  Future<bool> _load() async {
    if (_loading || _saving) return true;
    _loading = true;
    try {
      final plan = await widget.client.plan(widget.project.id);
      if (mounted) {
        setState(() {
          if (_plan == null) _error = null;
          _plan = plan;
          if (!_dirty) {
            _source = plan.draft;
            _editor.text = plan.draft;
          }
        });
      }
      return true;
    } catch (_) {
      if (mounted && _plan == null) {
        setState(() => _error = 'Start the office to review its plan.');
      }
      return false;
    } finally {
      _loading = false;
    }
  }

  Future<void> _save() async {
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.client.workspaceAction(widget.project.id, {
        'action': 'plan-update',
        'expected': _source,
        'text': _editor.text,
      });
      if (mounted) {
        setState(() {
          _dirty = false;
          _editing = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(
          () => _error = e is GatewayException
              ? e.message
              : 'Could not save the plan.',
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
    if (mounted) await _load();
  }

  Future<void> _approve() async {
    final expected = _source;
    final yes = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Approve this plan?'),
        content: const Text(
          'The assistant will start implementing the plan you just reviewed.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Keep planning'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Approve & build'),
          ),
        ],
      ),
    );
    if (yes != true || !mounted) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.client.workspaceAction(widget.project.id, {
        'action': 'plan-approve',
        'expected': expected,
      });
    } catch (e) {
      if (mounted) {
        setState(
          () => _error = e is GatewayException
              ? e.message
              : 'Could not approve the plan.',
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
    if (mounted) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final plan = _plan;
    final stale = _dirty && plan?.draft != _source;
    final approved =
        plan != null && plan.approved.isNotEmpty && plan.approved == plan.draft;
    return Polling(
      interval: const Duration(seconds: 3),
      enabled: TickerMode.valuesOf(context).enabled,
      restartToken: 0,
      onTick: _load,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    plan?.pending == true
                        ? 'Starting build…'
                        : approved
                        ? 'Approved plan'
                        : 'Plan before building',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                IconButton(
                  tooltip: 'Refresh plan',
                  onPressed: _load,
                  icon: const Icon(Icons.refresh),
                ),
                if (plan?.draft.isNotEmpty == true)
                  IconButton(
                    tooltip: 'Edit plan',
                    onPressed: _saving
                        ? null
                        : () => setState(() => _editing = !_editing),
                    icon: Icon(
                      _editing
                          ? Icons.visibility_outlined
                          : Icons.edit_outlined,
                    ),
                  ),
              ],
            ),
          ),
          if (_error != null || plan?.error.isNotEmpty == true)
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(
                _error ?? plan!.error,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            ),
          if (stale)
            MaterialBanner(
              content: const Text(
                'A newer draft arrived. Copy any edits you want to keep, then load the latest plan.',
              ),
              actions: [
                TextButton(
                  onPressed: () => setState(() {
                    _dirty = false;
                    _source = plan!.draft;
                    _editor.text = _source;
                  }),
                  child: const Text('Load latest'),
                ),
              ],
            ),
          Expanded(
            child: plan == null
                ? Center(
                    child: _error == null
                        ? const CircularProgressIndicator()
                        : TextButton(
                            onPressed: _load,
                            child: const Text('Retry'),
                          ),
                  )
                : plan.draft.isEmpty
                ? const Center(
                    child: Padding(
                      padding: EdgeInsets.all(24),
                      child: Text(
                        'Large tasks start with a plan. When the assistant presents one, review it here before building.',
                      ),
                    ),
                  )
                : _editing
                ? Padding(
                    padding: const EdgeInsets.all(16),
                    child: TextField(
                      controller: _editor,
                      enabled: !_saving,
                      expands: true,
                      maxLines: null,
                      textAlignVertical: TextAlignVertical.top,
                      onChanged: (_) => setState(() => _dirty = true),
                      decoration: const InputDecoration(
                        labelText: 'Plan draft',
                        alignLabelWithHint: true,
                      ),
                    ),
                  )
                : SingleChildScrollView(
                    padding: const EdgeInsets.all(20),
                    child: MarkdownBody(markdown: plan.draft),
                  ),
          ),
          if (plan?.draft.isNotEmpty == true)
            SafeArea(
              top: false,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: FilledButton.icon(
                  onPressed:
                      _saving || stale || plan!.pending || (approved && !_dirty)
                      ? null
                      : _dirty
                      ? _save
                      : _approve,
                  icon: Icon(
                    _dirty ? Icons.save_outlined : Icons.check_circle_outline,
                  ),
                  label: Text(
                    _saving
                        ? 'Saving…'
                        : plan!.pending
                        ? 'Starting build…'
                        : approved
                        ? 'Approved'
                        : _dirty
                        ? 'Save changes'
                        : 'Approve & build',
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
