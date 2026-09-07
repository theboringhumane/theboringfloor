import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import 'api/gateway_client.dart';
import 'api/models.dart';
import 'settings_store.dart';
import 'time.dart';

void main() => runApp(const TheBoringFloorApp());

class TheBoringFloorApp extends StatelessWidget {
  const TheBoringFloorApp({super.key});

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'theboringfloor',
    theme: ThemeData(
      colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xff155e75)),
      useMaterial3: true,
    ),
    home: const ProjectsScreen(),
  );
}

void showGatewayError(BuildContext context, Object error) {
  final text = error is GatewayException ? error.message : error.toString();
  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text)));
}

class ProjectsScreen extends StatefulWidget {
  const ProjectsScreen({super.key});

  @override
  State<ProjectsScreen> createState() => _ProjectsScreenState();
}

class _ProjectsScreenState extends State<ProjectsScreen>
    with WidgetsBindingObserver {
  final _store = SettingsStore();
  GatewaySettings? _settings;
  List<Project> _projects = const [];
  Timer? _poller;
  bool _loading = true;
  bool _foreground = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _load();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _poller?.cancel();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _syncPolling();
  }

  GatewayClient? get _client => _settings?.configured == true
      ? GatewayClient(baseUrl: _settings!.baseUrl, token: _settings!.token)
      : null;

  Future<void> _load() async {
    final settings = await _store.load();
    if (!mounted) return;
    setState(() => _settings = settings);
    await _refresh();
    _syncPolling();
  }

  void _syncPolling() {
    _poller?.cancel();
    _poller = null;
    if (_foreground && _client != null) {
      _poller = Timer.periodic(
        const Duration(seconds: 5),
        (_) => _refresh(silent: true),
      );
    }
  }

  Future<void> _refresh({bool silent = false}) async {
    final client = _client;
    if (client == null) {
      if (mounted) setState(() => _loading = false);
      return;
    }
    try {
      final projects = await client.projects();
      if (mounted) {
        setState(() {
          _projects = projects;
          _loading = false;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() => _loading = false);
        if (!silent) showGatewayError(context, error);
      }
    }
  }

  Future<void> _openSettings() async {
    await Navigator.of(context)
        .push(MaterialPageRoute(builder: (_) => const SettingsScreen()));
    await _load();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: const Text('Projects'),
      actions: [
        IconButton(
          icon: const Icon(Icons.settings),
          tooltip: 'Settings',
          onPressed: _openSettings,
        ),
      ],
    ),
    body: _settings?.configured != true
        ? Center(
            child: FilledButton.icon(
              onPressed: _openSettings,
              icon: const Icon(Icons.settings),
              label: const Text('Configure gateway'),
            ),
          )
        : _loading
        ? const Center(child: CircularProgressIndicator())
        : RefreshIndicator(
            onRefresh: _refresh,
            child: _projects.isEmpty
                ? ListView(
                    children: const [
                      SizedBox(height: 220),
                      Center(child: Text('No projects found.')),
                    ],
                  )
                : ListView.separated(
                    padding: const EdgeInsets.all(12),
                    itemCount: _projects.length,
                    separatorBuilder: (_, _) => const SizedBox(height: 8),
                    itemBuilder: (_, index) =>
                        _ProjectCard(project: _projects[index]),
                  ),
          ),
  );
}

class _ProjectCard extends StatelessWidget {
  const _ProjectCard({required this.project});
  final Project project;

  @override
  Widget build(BuildContext context) => Card(
    child: ListTile(
      onTap: () => Navigator.of(context).push(
        MaterialPageRoute(builder: (_) => SessionScreen(project: project)),
      ),
      title: Row(
        children: [
          Expanded(child: Text(project.name)),
          _LiveBadge(live: project.live),
        ],
      ),
      subtitle: Padding(
        padding: const EdgeInsets.only(top: 7),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(project.dir, maxLines: 1, overflow: TextOverflow.ellipsis),
            const SizedBox(height: 3),
            Text(
              '${project.backend} • ${project.chatCount} chats • ${formatRelativeTime(DateTime.fromMillisecondsSinceEpoch(project.savedAt))}',
            ),
          ],
        ),
      ),
      trailing: const Icon(Icons.chevron_right),
    ),
  );
}

class _LiveBadge extends StatelessWidget {
  const _LiveBadge({required this.live});
  final bool live;
  @override
  Widget build(BuildContext context) => Chip(
    label: Text(live ? 'LIVE' : 'SAVED'),
    labelStyle: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold),
    backgroundColor: live ? Colors.green.shade100 : Colors.grey.shade200,
    visualDensity: VisualDensity.compact,
  );
}

class SessionScreen extends StatefulWidget {
  const SessionScreen({super.key, required this.project});
  final Project project;
  @override
  State<SessionScreen> createState() => _SessionScreenState();
}

class _SessionScreenState extends State<SessionScreen>
    with WidgetsBindingObserver {
  final _store = SettingsStore();
  final _composer = TextEditingController();
  final _scrollController = ScrollController();
  Timer? _poller;
  GatewaySettings? _settings;
  List<TranscriptMessage> _messages = const [];
  Busy? _busy;
  bool _loading = true;
  bool _sending = false;
  bool _foreground = true;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _load();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _poller?.cancel();
    _composer.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _syncPolling();
  }

  GatewayClient? get _client => _settings?.configured == true
      ? GatewayClient(baseUrl: _settings!.baseUrl, token: _settings!.token)
      : null;

  Future<void> _load() async {
    _settings = await _store.load();
    if (!mounted) return;
    await _refresh();
    _syncPolling();
  }

  void _syncPolling() {
    _poller?.cancel();
    if (_foreground && _client != null) {
      _poller = Timer.periodic(
        const Duration(seconds: 3),
        (_) => _refresh(silent: true),
      );
    }
  }

  Future<void> _refresh({bool silent = false}) async {
    final client = _client;
    if (client == null) return;
    try {
      final values = await Future.wait([
        client.transcript(widget.project.id),
        client.busy(widget.project.id),
      ]);
      if (mounted) {
        setState(() {
          _messages = values[0] as List<TranscriptMessage>;
          _busy = values[1] as Busy;
          _loading = false;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() => _loading = false);
        if (!silent) showGatewayError(context, error);
      }
    }
  }

  Future<bool> _confirm(String action) async =>
      await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text('$action session?'),
          content: Text('Are you sure you want to $action this session?'),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: Text(action),
            ),
          ],
        ),
      ) ??
      false;
  Future<void> _action(String action) async {
    if (!await _confirm(action)) return;
    final client = _client;
    if (client == null) return;
    try {
      if (action == 'Stop') {
        await client.stop(widget.project.id);
      } else {
        await client.startNew(widget.project.id);
      }
      if (mounted) await _refresh();
    } catch (error) {
      if (mounted) showGatewayError(context, error);
    }
  }

  Future<void> _send() async {
    final text = _composer.text.trim();
    final client = _client;
    if (text.isEmpty || client == null) return;
    setState(() => _sending = true);
    try {
      await client.message(widget.project.id, text);
      _composer.clear();
      await _refresh();
    } catch (error) {
      if (mounted) showGatewayError(context, error);
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: Text(widget.project.name),
      actions: [
        IconButton(
          onPressed: () => _action('Stop'),
          icon: const Icon(Icons.stop_circle_outlined),
          tooltip: 'Stop',
        ),
        IconButton(
          onPressed: () => _action('New'),
          icon: const Icon(Icons.add_circle_outline),
          tooltip: 'New session',
        ),
      ],
    ),
    body: Column(
      children: [
        if (_busy?.busy == true)
          LinearProgressIndicator(
            value: null,
            semanticsLabel: _busy!.thinking ? 'Thinking' : 'Busy',
          ),
        if (_busy?.busy == true)
          Padding(
            padding: const EdgeInsets.all(8),
            child: Text(
              _busy!.thinking
                  ? 'Thinking…'
                  : _busy!.delegating
                  ? 'Delegating…'
                  : 'Office is busy',
            ),
          ),
        Expanded(
          child: _loading
              ? const Center(child: CircularProgressIndicator())
              : _messages.isEmpty
              ? const Center(child: Text('No transcript messages yet.'))
              : ListView.builder(
                  controller: _scrollController,
                  padding: const EdgeInsets.all(12),
                  itemCount: _messages.length,
                  itemBuilder: (_, i) => _MessageBubble(message: _messages[i]),
                ),
        ),
        SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _composer,
                    minLines: 1,
                    maxLines: 4,
                    textInputAction: TextInputAction.send,
                    onSubmitted: (_) => _send(),
                    decoration: const InputDecoration(
                      border: OutlineInputBorder(),
                      hintText: 'Message the office',
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton.filled(
                  onPressed: _sending ? null : _send,
                  icon: _sending
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.send),
                  tooltip: 'Send',
                ),
              ],
            ),
          ),
        ),
      ],
    ),
  );
}

class _MessageBubble extends StatelessWidget {
  const _MessageBubble({required this.message});
  final TranscriptMessage message;
  @override
  Widget build(BuildContext context) {
    final user = message.from == 'user' || message.kind == 'user';
    final muted = ['think', 'wthink', 'tool', 'wtool'].contains(message.kind);
    final color = user
        ? Theme.of(context).colorScheme.primaryContainer
        : muted
        ? Colors.grey.shade200
        : Theme.of(context).colorScheme.surfaceContainerHighest;
    final label = message.kind.isEmpty
        ? message.from
        : '${message.from} · ${message.kind}';
    return Align(
      alignment: user ? Alignment.centerRight : Alignment.centerLeft,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 620),
        child: Card(
          color: color,
          child: Padding(
            padding: const EdgeInsets.all(10),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: Theme.of(context).textTheme.labelSmall),
                const SizedBox(height: 4),
                SelectableText(
                  message.text,
                  style: TextStyle(fontFamily: muted ? 'monospace' : null),
                ),
                const SizedBox(height: 5),
                Text(
                  DateFormat.jm().format(
                    DateTime.fromMillisecondsSinceEpoch(message.at),
                  ),
                  style: Theme.of(context).textTheme.labelSmall,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key});
  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final _store = SettingsStore();
  final _url = TextEditingController();
  final _token = TextEditingController();
  bool _testing = false;
  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _url.dispose();
    _token.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final settings = await _store.load();
    if (mounted) {
      _url.text = settings.baseUrl;
      _token.text = settings.token;
    }
  }

  Future<void> _save() =>
      _store.save(GatewaySettings(baseUrl: _url.text, token: _token.text));
  Future<void> _test() async {
    setState(() => _testing = true);
    try {
      await _save();
      final health = await GatewayClient(
        baseUrl: _url.text.trim(),
        token: _token.text,
      ).health();
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(jsonHealth(health))));
      }
    } catch (error) {
      if (mounted) showGatewayError(context, error);
    } finally {
      if (mounted) setState(() => _testing = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Settings')),
    body: ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const Text('Gateway', style: TextStyle(fontWeight: FontWeight.bold)),
        const SizedBox(height: 12),
        TextField(
          controller: _url,
          keyboardType: TextInputType.url,
          decoration: const InputDecoration(
            labelText: 'Gateway base URL',
            hintText: 'http://100.x.y.z:8787',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 14),
        TextField(
          controller: _token,
          obscureText: true,
          decoration: const InputDecoration(
            labelText: 'Bearer token',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 16),
        FilledButton.icon(
          onPressed: _testing ? null : _test,
          icon: const Icon(Icons.wifi_tethering),
          label: Text(_testing ? 'Testing…' : 'Test connection'),
        ),
        const SizedBox(height: 8),
        TextButton(
          onPressed: () async {
            await _save();
            if (context.mounted) Navigator.pop(context);
          },
          child: const Text('Save settings'),
        ),
      ],
    ),
  );
}

String jsonHealth(Map<String, dynamic> health) => jsonEncode(health);
