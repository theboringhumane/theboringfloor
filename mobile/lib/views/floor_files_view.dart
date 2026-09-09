import 'package:flutter/material.dart';

import '../api/gateway_client.dart';
import '../models/floor.dart';
import '../models/project.dart';
import '../store/session_store.dart';
import 'session_view.dart';

class FloorFilesView extends StatefulWidget {
  const FloorFilesView({
    super.key,
    required this.client,
    required this.project,
  });
  final GatewayClient client;
  final Project project;
  @override
  State<FloorFilesView> createState() => _FloorFilesViewState();
}

class _FloorFilesViewState extends State<FloorFilesView> {
  final Map<String, ProjectFilePage> _folders = {};
  final Set<String> _expanded = {''};
  final Set<String> _loading = {};
  String? _error;
  @override
  void initState() {
    super.initState();
    _load('');
  }

  Future<void> _load(String path) async {
    if (_loading.contains(path)) return;
    setState(() => _loading.add(path));
    try {
      final page = await widget.client.files(widget.project.id, path);
      if (mounted) {
        setState(() {
          _folders[path] = page;
          _error = null;
        });
      }
    } catch (_) {
      if (mounted) {
        setState(() => _error = 'Could not load files. Refresh to try again.');
      }
    } finally {
      if (mounted) setState(() => _loading.remove(path));
    }
  }

  Future<void> _select(ProjectFile file) async {
    if (file.directory) {
      if (_expanded.contains(file.path)) {
        setState(() => _expanded.remove(file.path));
      } else {
        setState(() => _expanded.add(file.path));
        if (!_folders.containsKey(file.path)) await _load(file.path);
      }
    } else {
      await Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => FilePreviewView(
            client: widget.client,
            project: widget.project,
            file: file,
          ),
        ),
      );
    }
  }

  List<Widget> _rows(String path, int depth) {
    final page = _folders[path];
    if (page == null) return [];
    return [
      for (final file in page.entries) ...[
        Padding(
          padding: EdgeInsets.only(left: depth * 16.0),
          child: ListTile(
            key: ValueKey(file.path),
            dense: true,
            leading: Icon(
              file.directory
                  ? (_expanded.contains(file.path)
                        ? Icons.folder_open
                        : Icons.folder_outlined)
                  : Icons.description_outlined,
            ),
            title: Text(
              file.name,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: _loading.contains(file.path)
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : file.directory
                ? Icon(
                    _expanded.contains(file.path)
                        ? Icons.expand_less
                        : Icons.expand_more,
                  )
                : null,
            onTap: () => _select(file),
          ),
        ),
        if (file.directory && _expanded.contains(file.path))
          ..._rows(file.path, depth + 1),
      ],
      if (page.truncated)
        const ListTile(title: Text('Showing the first 1,000 entries.')),
    ];
  }

  @override
  Widget build(BuildContext context) => Column(
    children: [
      ListTile(
        title: const Text('Project files'),
        subtitle: Text(
          widget.project.dir,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        trailing: IconButton(
          tooltip: 'Refresh files',
          onPressed: () {
            _folders.clear();
            _expanded
              ..clear()
              ..add('');
            _load('');
          },
          icon: const Icon(Icons.refresh),
        ),
      ),
      if (_error != null)
        Padding(padding: const EdgeInsets.all(16), child: Text(_error!)),
      Expanded(
        child: _loading.contains('')
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.symmetric(horizontal: 8),
                children: [
                  ..._rows('', 0),
                  if (_folders['']?.entries.isEmpty == true)
                    const ListTile(
                      title: Text('This project has no files yet.'),
                    ),
                ],
              ),
      ),
    ],
  );
}

class FilePreviewView extends StatefulWidget {
  const FilePreviewView({
    super.key,
    required this.client,
    required this.project,
    required this.file,
  });
  final GatewayClient client;
  final Project project;
  final ProjectFile file;
  @override
  State<FilePreviewView> createState() => _FilePreviewViewState();
}

class _FilePreviewViewState extends State<FilePreviewView> {
  late Future<ProjectFilePage> _page = widget.client.files(
    widget.project.id,
    widget.file.path,
  );
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: Text(widget.file.name),
      actions: [
        IconButton(
          tooltip: 'Ask about this file',
          icon: const Icon(Icons.add_comment_outlined),
          onPressed: () => Navigator.of(context).push(
            MaterialPageRoute<void>(
              builder: (_) => SessionView(
                store: SessionStore(widget.client, widget.project),
                initialDraft: 'Review the project file ${widget.file.path}.',
              ),
            ),
          ),
        ),
      ],
    ),
    body: FutureBuilder<ProjectFilePage>(
      future: _page,
      builder: (context, snapshot) {
        if (snapshot.hasError) {
          return Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Text('Preview unavailable for this file.'),
                TextButton(
                  onPressed: () => setState(
                    () => _page = widget.client.files(
                      widget.project.id,
                      widget.file.path,
                    ),
                  ),
                  child: const Text('Retry'),
                ),
              ],
            ),
          );
        }
        final page = snapshot.data;
        if (page == null) {
          return const Center(child: CircularProgressIndicator());
        }
        if (page.binary) {
          return const Center(
            child: Text('Binary file · text preview unavailable'),
          );
        }
        final lines = page.content.split('\n');
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.all(16),
              child: Text(
                '${widget.file.path} · Read only${page.truncated ? ' · First 64 KiB' : ''}',
              ),
            ),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(16),
                child: SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: SelectableText(
                    lines.indexed
                        .map(
                          (line) =>
                              '${(line.$1 + 1).toString().padLeft(4)}  ${line.$2}',
                        )
                        .join('\n'),
                    style: const TextStyle(
                      fontFamily: 'JetBrains Mono',
                      fontSize: 12,
                      height: 1.6,
                    ),
                  ),
                ),
              ),
            ),
          ],
        );
      },
    ),
  );
}
