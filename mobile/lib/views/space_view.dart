import 'package:flutter/material.dart';

import '../components/empty_state.dart';
import '../components/project_tile.dart';
import '../models/project.dart';
import '../store/projects_store.dart';

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
        (p) =>
            (_filter == 0 || (_filter == 1 ? p.live : !p.live)) &&
            (_query.isEmpty ||
                p.name.toLowerCase().contains(_query.toLowerCase()) ||
                p.dir.toLowerCase().contains(_query.toLowerCase())),
      )
      .toList();

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return SafeArea(
      child: RefreshIndicator(
        onRefresh: widget.store.load,
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 24, 24, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Space',
                    style: Theme.of(context).textTheme.headlineLarge,
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    onChanged: (value) => setState(() => _query = value),
                    decoration: InputDecoration(
                      hintText: 'Search projects',
                      prefixIcon: const Icon(Icons.search),
                      filled: true,
                      fillColor: colors.surfaceContainerLow,
                      border: const OutlineInputBorder(
                        borderRadius: BorderRadius.all(Radius.circular(28)),
                        borderSide: BorderSide.none,
                      ),
                      enabledBorder: const OutlineInputBorder(
                        borderRadius: BorderRadius.all(Radius.circular(28)),
                        borderSide: BorderSide.none,
                      ),
                      focusedBorder: OutlineInputBorder(
                        borderRadius: const BorderRadius.all(
                          Radius.circular(28),
                        ),
                        borderSide: BorderSide(color: colors.primary, width: 2),
                      ),
                    ),
                  ),
                  const SizedBox(height: 12),
                  Wrap(
                    spacing: 8,
                    children: _filters
                        .asMap()
                        .entries
                        .map(
                          (e) => FilterChip(
                            label: Text(e.value),
                            selected: _filter == e.key,
                            showCheckmark: false,
                            selectedColor: colors.secondaryContainer,
                            backgroundColor: colors.surface,
                            side: BorderSide(color: colors.outlineVariant),
                            onSelected: (_) => setState(() => _filter = e.key),
                          ),
                        )
                        .toList(),
                  ),
                ],
              ),
            ),
            Expanded(
              child: widget.store.loading
                  ? const Center(child: CircularProgressIndicator())
                  : _shown.isEmpty
                  ? const EmptyState('No projects match your filters.')
                  : ListView.builder(
                      physics: const AlwaysScrollableScrollPhysics(),
                      padding: const EdgeInsets.only(bottom: 16),
                      itemCount: _shown.length,
                      itemBuilder: (_, index) => ProjectTile(
                        project: _shown[index],
                        onTap: () => widget.onOpen(_shown[index]),
                      ),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}
