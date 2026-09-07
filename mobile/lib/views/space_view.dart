import 'package:flutter/material.dart';

import '../components/empty_state.dart';
import '../components/glass.dart';
import '../components/project_tile.dart';
import '../models/project.dart';
import '../store/projects_store.dart';
import '../utils/typography.dart';

class SpaceView extends StatefulWidget {
  const SpaceView({super.key, required this.store, required this.onOpen});

  final ProjectsStore store;
  final ValueChanged<Project> onOpen;

  @override
  State<SpaceView> createState() => _SpaceViewState();
}

class _SpaceViewState extends State<SpaceView> {
  static const _filters = ['All', 'Running', 'Stopped'];

  String _query = '';
  int _filter = 0;
  bool _searching = false;

  @override
  void initState() {
    super.initState();
    widget.store.addListener(_changed);
    widget.store.load();
  }

  @override
  void dispose() {
    widget.store.removeListener(_changed);
    super.dispose();
  }

  void _changed() {
    if (mounted) setState(() {});
  }

  List<Project> get _shown => widget.store.projects
      .where(
        (project) =>
            (_filter == 0 || (_filter == 1 ? project.live : !project.live)) &&
            (_query.isEmpty ||
                project.name.toLowerCase().contains(_query.toLowerCase()) ||
                project.dir.toLowerCase().contains(_query.toLowerCase())),
      )
      .toList();

  void _showFilters() {
    showModalBottomSheet<void>(
      context: context,
      builder: (context) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Show projects',
                style: AppFonts.heading(
                  context,
                  base: Theme.of(context).textTheme.titleMedium,
                ),
              ),
              const SizedBox(height: 16),
              Wrap(
                spacing: 8,
                children: _filters.asMap().entries.map((entry) {
                  return FilterChip(
                    label: Text(entry.value),
                    selected: _filter == entry.key,
                    showCheckmark: false,
                    onSelected: (_) {
                      setState(() => _filter = entry.key);
                      Navigator.pop(context);
                    },
                  );
                }).toList(),
              ),
            ],
          ),
        ),
      ),
    );
  }

  List<Widget> _section(String label, List<Project> projects) {
    if (projects.isEmpty) return const [];
    return [
      Padding(
        padding: const EdgeInsets.fromLTRB(24, 24, 24, 8),
        child: Text(
          label,
          style: AppFonts.heading(
            context,
            base: Theme.of(context).textTheme.labelMedium?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.1,
            ),
          ),
        ),
      ),
      ...projects.map(
        (project) =>
            ProjectTile(project: project, onTap: () => widget.onOpen(project)),
      ),
    ];
  }

  @override
  Widget build(BuildContext context) {
    final shown = _shown;
    final running = shown.where((project) => project.live).toList();
    final stopped = shown.where((project) => !project.live).toList();

    return SafeArea(
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(24, 16, 24, 12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Align(
                  alignment: Alignment.centerRight,
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      GlassIconButton(
                        icon: _searching ? Icons.close : Icons.search,
                        tooltip: _searching
                            ? 'Close search'
                            : 'Search projects',
                        intensity: GlassIntensity.subtle,
                        onPressed: () =>
                            setState(() => _searching = !_searching),
                      ),
                      const SizedBox(width: 8),
                      GlassIconButton(
                        icon: Icons.tune,
                        tooltip: 'Filter projects',
                        intensity: GlassIntensity.subtle,
                        onPressed: _showFilters,
                      ),
                    ],
                  ),
                ),
                Text(
                  'All Repos',
                  style: AppFonts.serif(
                    context,
                    base: Theme.of(context).textTheme.displaySmall,
                  ),
                ),
                if (_searching) ...[
                  const SizedBox(height: 16),
                  GlassSurface(
                    intensity: GlassIntensity.subtle,
                    padding: EdgeInsets.zero,
                    child: TextField(
                      autofocus: true,
                      onChanged: (value) => setState(() => _query = value),
                      decoration: const InputDecoration(
                        hintText: 'Search projects',
                        prefixIcon: Icon(Icons.search),
                        border: InputBorder.none,
                        contentPadding: EdgeInsets.symmetric(vertical: 14),
                      ),
                    ),
                  ),
                ],
              ],
            ),
          ),
          Expanded(
            child: widget.store.loading
                ? const Center(child: CircularProgressIndicator())
                : shown.isEmpty
                ? const EmptyState('No projects match your filters.')
                : RefreshIndicator(
                    onRefresh: widget.store.load,
                    child: ListView(
                      physics: const AlwaysScrollableScrollPhysics(),
                      padding: const EdgeInsets.only(
                        bottom: 24,
                        left: 8,
                        right: 8,
                      ),
                      children: [
                        ..._section('RUNNING', running),
                        ..._section('STOPPED', stopped),
                      ],
                    ),
                  ),
          ),
        ],
      ),
    );
  }
}
