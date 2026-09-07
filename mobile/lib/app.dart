import 'dart:async';

import 'package:dynamic_color/dynamic_color.dart';
import 'package:flutter/material.dart';

import 'api/gateway_client.dart';
import 'components/app_nav_bar.dart';
import 'models/project.dart';
import 'models/session.dart';
import 'store/projects_store.dart';
import 'store/session_store.dart';
import 'store/settings_store.dart';
import 'store/terminal_store.dart';
import 'theme.dart';
import 'views/project_picker.dart';
import 'views/session_view.dart';
import 'views/settings_view.dart';
import 'views/space_view.dart';
import 'views/terminal_view.dart';

class TheBoringFloorApp extends StatelessWidget {
  const TheBoringFloorApp({super.key, required this.settings});
  final SettingsStore settings;
  @override
  Widget build(BuildContext context) => DynamicColorBuilder(
    builder: (l, d) => MaterialApp(
      title: 'theboringfloor',
      theme: buildLightTheme(l),
      darkTheme: buildDarkTheme(d),
      themeMode: ThemeMode.system,
      home: AppShell(settings: settings),
    ),
  );
}

class AppShell extends StatefulWidget {
  const AppShell({
    super.key,
    required this.settings,
    this.client,
    this.readinessPollInterval = defaultReadinessPollInterval,
    this.readinessTimeout = defaultReadinessTimeout,
  });

  /// Gives a newly launched office a moment to write its discovery record.
  static const defaultReadinessPollInterval = Duration(milliseconds: 250);

  /// Bounds startup waiting so a launch that never becomes ready is honest.
  static const defaultReadinessTimeout = Duration(seconds: 10);

  final SettingsStore settings;
  final GatewayClient? client;
  final Duration readinessPollInterval;
  final Duration readinessTimeout;
  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  late GatewayClient client;
  late ProjectsStore projects;
  late TerminalStore terminal;
  final Completer<void> _disposed = Completer<void>();
  Timer? _pollTimer;
  Project? _pendingSession;
  int index = 0;
  @override
  void initState() {
    super.initState();
    client = widget.client ?? widget.settings.client();
    projects = ProjectsStore(client);
    terminal = TerminalStore(client);
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    if (!_disposed.isCompleted) {
      _disposed.complete();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    body: IndexedStack(
      index: index,
      children: [
        SpaceView(store: projects, onOpen: _open),
        TerminalView(store: terminal, cwd: ''),
        SettingsView(store: widget.settings),
      ],
    ),
    bottomNavigationBar: AppNavBar(
      index: index,
      onChanged: (v) => setState(() => index = v),
    ),
    floatingActionButton: FloatingActionButton(
      tooltip: 'New session',
      onPressed: _pick,
      child: const Icon(Icons.add),
    ),
  );

  void _open(Project p) => Navigator.of(context).push(
    MaterialPageRoute(
      builder: (_) => SessionView(store: SessionStore(client, p)),
    ),
  );

  Future<void> _pick() async {
    if (projects.projects.isEmpty && !projects.loading) {
      await projects.load();
    }
    if (!mounted) {
      return;
    }
    _pendingSession = null;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (_) =>
          ProjectPicker(projects: projects.projects, onSelect: _newSession),
    );
    final project = _pendingSession;
    _pendingSession = null;
    if (mounted && project != null) {
      _open(project);
    }
  }

  Future<void> _newSession(Project p) async {
    if (!p.live) {
      final result = await client.startOffice(p.id);
      switch (result.outcome) {
        case StartOfficeOutcome.accepted:
          final ready = await _waitForReadiness(p.id);
          if (!mounted) {
            return;
          }
          if (!ready) {
            throw StateError(
              'The office did not become ready in time. Try again.',
            );
          }
        case StartOfficeOutcome.alreadyRunning:
          break;
        case StartOfficeOutcome.projectNotFound:
          throw StateError('Project not found on the gateway.');
        case StartOfficeOutcome.launchFailed:
          throw StateError('Office could not be started: ${result.message}');
      }
    }
    await client.startNew(p.id);
    if (mounted) {
      _pendingSession = p;
    }
  }

  Future<bool> _waitForReadiness(String projectId) async {
    final deadline = DateTime.now().add(widget.readinessTimeout);
    while (mounted && DateTime.now().isBefore(deadline)) {
      try {
        await client.status(projectId);
        return mounted;
      } catch (_) {
        // An office is ready only after its status endpoint responds.
      }
      final delay = Completer<void>();
      _pollTimer = Timer(widget.readinessPollInterval, delay.complete);
      await Future.any<void>([delay.future, _disposed.future]);
      _pollTimer?.cancel();
      _pollTimer = null;
    }
    return false;
  }
}
