import 'package:flutter/material.dart';

import '../components/empty_state.dart';
import '../components/project_tile.dart';
import '../models/project.dart';
import '../utils/typography.dart';

class ProjectPicker extends StatefulWidget {
  const ProjectPicker({
    super.key,
    required this.projects,
    required this.onSelect,
  });
  final List<Project> projects;
  final Future<void> Function(Project) onSelect;
  @override
  State<ProjectPicker> createState() => _ProjectPickerState();
}

class _ProjectPickerState extends State<ProjectPicker> {
  String query = '';
  String? busy, error;
  Project? failedProject;
  @override
  Widget build(BuildContext context) {
    final items = widget.projects
        .where(
          (p) =>
              query.isEmpty ||
              p.name.toLowerCase().contains(query.toLowerCase()) ||
              p.dir.toLowerCase().contains(query.toLowerCase()),
        )
        .toList();
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              'Start a new session',
              style: AppFonts.heading(
                context,
                base: Theme.of(context).textTheme.headlineSmall,
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              onChanged: (v) => setState(() => query = v),
              decoration: const InputDecoration(hintText: 'Search projects'),
            ),
            if (error != null)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        error!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    ),
                    if (failedProject != null)
                      TextButton(
                        onPressed: busy == null
                            ? () => _select(failedProject!)
                            : null,
                        child: const Text('Retry'),
                      ),
                  ],
                ),
              ),
            if (busy != null)
              const Padding(
                padding: EdgeInsets.only(top: 8),
                child: Text('Starting office…'),
              ),
            SizedBox(
              height: 320,
              child: items.isEmpty
                  ? const EmptyState('No projects available.')
                  : ListView(
                      children: items
                          .map(
                            (p) => ProjectTile(
                              project: p,
                              onTap: () => busy == null ? _select(p) : null,
                              trailing: busy == p.id
                                  ? const SizedBox(
                                      width: 24,
                                      height: 24,
                                      child: CircularProgressIndicator(),
                                    )
                                  : const Icon(Icons.chevron_right),
                            ),
                          )
                          .toList(),
                    ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _select(Project p) async {
    setState(() {
      busy = p.id;
      error = null;
      failedProject = null;
    });
    try {
      await widget.onSelect(p);
      if (mounted) Navigator.pop(context);
    } catch (e) {
      if (mounted) {
        setState(() {
          error = '$e'.replaceFirst('Bad state: ', '');
          failedProject = p;
        });
      }
    }
    if (mounted) setState(() => busy = null);
  }
}
