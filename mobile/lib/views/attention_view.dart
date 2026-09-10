import 'package:flutter/material.dart';

import '../hooks/use_polling.dart';
import '../models/project.dart';
import '../store/attention_store.dart';
import '../utils/typography.dart';
import 'floor_view.dart';

class AttentionView extends StatefulWidget {
  const AttentionView({super.key, required this.store, this.active = true});
  final AttentionStore store;
  final bool active;
  @override
  State<AttentionView> createState() => _AttentionViewState();
}

class _AttentionViewState extends State<AttentionView> {
  bool _onlyAttention = true;
  int _retry = 0;
  @override
  void initState() {
    super.initState();
    widget.store.refresh();
  }

  Future<void> _open(Project project, {int tab = 0, String? ticket}) async {
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => FloorView(
          client: widget.store.client,
          project: project,
          initialTab: tab,
          initialTicket: ticket,
        ),
      ),
    );
    if (mounted) await widget.store.refresh();
  }

  @override
  Widget build(BuildContext context) => ListenableBuilder(
    listenable: widget.store,
    builder: (context, _) {
      final store = widget.store;
      final colors = Theme.of(context).colorScheme;
      final floors = store.floors
          .where((f) => !_onlyAttention || f.needsAttention)
          .toList();
      return Polling(
        interval: const Duration(seconds: 15),
        enabled: widget.active && TickerMode.valuesOf(context).enabled,
        restartToken: _retry,
        onTick: store.refresh,
        child: SafeArea(
          child: RefreshIndicator(
            onRefresh: () async {
              setState(() => _retry++);
              await store.refresh();
            },
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(24, 24, 24, 96),
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        'Your office',
                        style: AppFonts.serif(
                          context,
                          base: Theme.of(context).textTheme.displaySmall,
                        ),
                      ),
                    ),
                    IconButton(
                      tooltip: 'Refresh inbox',
                      onPressed: store.loading
                          ? null
                          : () {
                              setState(() => _retry++);
                              store.refresh();
                            },
                      icon: const Icon(Icons.refresh),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Text(
                  'Decisions here. Progress everywhere.',
                  style: Theme.of(context).textTheme.bodyLarge
                      ?.copyWith(color: colors.onSurfaceVariant),
                ),
                const SizedBox(height: 24),
                Container(
                  padding: const EdgeInsets.all(24),
                  decoration: BoxDecoration(
                    color: colors.primaryContainer,
                    borderRadius: BorderRadius.circular(28),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Icon(
                        store.needsAttention > 0
                            ? Icons.front_hand_outlined
                            : Icons.done_all,
                        color: colors.onPrimaryContainer,
                      ),
                      const SizedBox(height: 16),
                      Text(
                        store.error != null
                            ? 'Connection needs attention'
                            : store.updatedAt == null
                            ? 'Let’s check in'
                            : store.needsAttention == 0
                            ? 'Room to focus'
                            : '${store.needsAttention} ${store.needsAttention == 1 ? 'floor needs' : 'floors need'} you',
                        style: Theme.of(context).textTheme.headlineMedium
                            ?.copyWith(color: colors.onPrimaryContainer),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        '${store.working} working · ${store.floors.length} floors',
                        style: TextStyle(color: colors.onPrimaryContainer),
                      ),
                      if (store.updatedAt != null) ...[
                        const SizedBox(height: 12),
                        Text(
                          'Checked ${TimeOfDay.fromDateTime(store.updatedAt!).format(context)}',
                          style: TextStyle(color: colors.onPrimaryContainer),
                        ),
                      ],
                    ],
                  ),
                ),
                if (store.loading)
                  const Padding(
                    padding: EdgeInsets.symmetric(vertical: 12),
                    child: LinearProgressIndicator(),
                  ),
                if (store.error != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 16),
                    child: Text(
                      store.error!,
                      style: TextStyle(color: colors.error),
                    ),
                  ),
                const SizedBox(height: 24),
                Wrap(
                  spacing: 8,
                  children: [
                    ChoiceChip(
                      label: const Text('Needs you'),
                      selected: _onlyAttention,
                      onSelected: (_) => setState(() => _onlyAttention = true),
                    ),
                    ChoiceChip(
                      label: const Text('All floors'),
                      selected: !_onlyAttention,
                      onSelected: (_) => setState(() => _onlyAttention = false),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                if (!store.loading && floors.isEmpty)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 32),
                    child: Column(
                      children: [
                        Icon(
                          Icons.spa_outlined,
                          size: 40,
                          color: colors.primary,
                        ),
                        const SizedBox(height: 16),
                        Text(
                          store.floors.isEmpty
                              ? 'Your floors will appear here'
                              : 'Nothing needs your attention',
                          style: Theme.of(context).textTheme.titleLarge,
                        ),
                        const SizedBox(height: 8),
                        Text(
                          store.floors.isEmpty
                              ? 'Connect your gateway in Settings, then add a project from the desktop office.'
                              : 'Plans, blocked tickets, and review requests appear here when they need a decision.',
                          textAlign: TextAlign.center,
                        ),
                      ],
                    ),
                  ),
                ...floors.map((pulse) => _card(context, pulse)),
              ],
            ),
          ),
        ),
      );
    },
  );

  Widget _card(BuildContext context, FloorPulse pulse) {
    final colors = Theme.of(context).colorScheme;
    final execution = pulse.status?.execution;
    final plan =
        execution?.state == 'plan' ||
        (execution == null && pulse.status?.planPending == true);
    final state = execution?.state ?? (pulse.project.live ? 'live' : 'stopped');
    final icon = switch (state) {
      'plan' => Icons.assignment_outlined,
      'permission' => Icons.lock_outline,
      'question' => Icons.help_outline,
      'working' => Icons.auto_awesome,
      'offline' => Icons.cloud_off_outlined,
      _ => Icons.domain_outlined,
    };
    return Card(
      margin: const EdgeInsets.only(bottom: 16),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Icon(icon, color: colors.primary),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    pulse.project.name,
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                ),
                IconButton(
                  tooltip: 'Open floor',
                  onPressed: () => _open(pulse.project),
                  icon: const Icon(Icons.arrow_outward),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              execution?.summary ??
                  (pulse.project.live
                      ? 'Office is online'
                      : 'Office is stopped'),
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            if (pulse.problem != null)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(
                  pulse.problem!,
                  style: TextStyle(color: colors.error),
                ),
              ),
            if (plan)
              Padding(
                padding: const EdgeInsets.only(top: 16),
                child: FilledButton.icon(
                  onPressed: () => _open(pulse.project, tab: 3),
                  icon: const Icon(Icons.fact_check_outlined),
                  label: const Text('Review plan'),
                ),
              ),
            for (final ticket in [...pulse.blocked, ...pulse.review])
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: Icon(
                  ticket.status == 'blocked'
                      ? Icons.pause_circle_outline
                      : Icons.rate_review_outlined,
                  color: ticket.status == 'blocked'
                      ? colors.error
                      : colors.primary,
                ),
                title: Text(ticket.title),
                subtitle: Text(
                  ticket.status == 'blocked'
                      ? 'Blocked · ${ticket.priority}'
                      : 'Ready for your review',
                ),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => _open(pulse.project, tab: 1, ticket: ticket.id),
              ),
          ],
        ),
      ),
    );
  }
}
