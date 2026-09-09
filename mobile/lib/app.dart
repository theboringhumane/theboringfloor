import 'dart:async';

import 'package:dynamic_color/dynamic_color.dart';
import 'package:flutter/material.dart';

import 'api/gateway_client.dart';
import 'components/app_nav_bar.dart';
import 'models/project.dart';
import 'models/session.dart';
import 'store/projects_store.dart';
import 'store/settings_store.dart';
import 'store/terminal_store.dart';
import 'theme.dart';
import 'views/project_picker.dart';
import 'views/floor_view.dart';
import 'views/session_view.dart';
import 'store/session_store.dart';
import 'views/settings_view.dart';
import 'views/splash_view.dart';
import 'views/space_view.dart';
import 'views/terminal_view.dart';

class AppBootstrap {
  const AppBootstrap({required this.settings, required this.projects});

  final SettingsStore settings;
  final ProjectsStore projects;

  static Future<AppBootstrap> load() async {
    final settings = await SettingsStore.load();
    final projects = ProjectsStore(settings.client());
    // A first run has no gateway to contact; preserve access to Settings.
    if (!settings.settings.configured) {
      return AppBootstrap(settings: settings, projects: projects);
    }
    await projects.load();
    if (projects.error case final error?) {
      throw error;
    }
    return AppBootstrap(settings: settings, projects: projects);
  }
}

class TheBoringFloorApp extends StatefulWidget {
  const TheBoringFloorApp({
    super.key,
    this.bootstrap = AppBootstrap.load,
    this.bootstrapTimeout = defaultBootstrapTimeout,
  });

  /// Bounds startup work so a failed gateway cannot leave a permanent spinner.
  static const defaultBootstrapTimeout = Duration(seconds: 15);

  final Future<AppBootstrap> Function() bootstrap;
  final Duration bootstrapTimeout;

  @override
  State<TheBoringFloorApp> createState() => _TheBoringFloorAppState();
}

class _TheBoringFloorAppState extends State<TheBoringFloorApp> {
  AppBootstrap? _bootstrap;
  String? _errorMessage;
  int _attempt = 0;

  @override
  void initState() {
    super.initState();
    _startBootstrap();
  }

  Future<void> _startBootstrap() async {
    final attempt = ++_attempt;
    setState(() {
      _bootstrap = null;
      _errorMessage = null;
    });
    try {
      final result = await widget.bootstrap().timeout(widget.bootstrapTimeout);
      if (!mounted || attempt != _attempt) {
        return;
      }
      setState(() => _bootstrap = result);
    } on TimeoutException {
      if (!mounted || attempt != _attempt) {
        return;
      }
      setState(
        () => _errorMessage = 'The app took too long to start. Try again.',
      );
    } catch (_) {
      if (!mounted || attempt != _attempt) {
        return;
      }
      setState(
        () => _errorMessage = 'We could not finish starting the app. Check your connection and try again.',
      );
    }
  }

  @override
  Widget build(BuildContext context) => DynamicColorBuilder(
    builder: (l, d) {
      final bootstrap = _bootstrap;
      return MaterialApp(
        title: 'theboringfloor',
        theme: buildLightTheme(l),
        darkTheme: buildDarkTheme(d),
        themeMode: ThemeMode.system,
        home: bootstrap == null
            ? SplashView(errorMessage: _errorMessage, onRetry: _startBootstrap)
            : AppShell(
                settings: bootstrap.settings,
                projects: bootstrap.projects,
              ),
      );
    },
  );
}

class AppShell extends StatefulWidget {
  const AppShell({
    super.key,
    required this.settings,
    this.client,
    this.projects,
    this.readinessPollInterval = defaultReadinessPollInterval,
    this.readinessTimeout = defaultReadinessTimeout,
  });

  /// Gives a newly launched office a moment to write its discovery record.
  static const defaultReadinessPollInterval = Duration(milliseconds: 250);

  /// Bounds startup waiting so a launch that never becomes ready is honest.
  static const defaultReadinessTimeout = Duration(seconds: 10);

  final SettingsStore settings;
  final GatewayClient? client;
  final ProjectsStore? projects;
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
    client =
        widget.client ?? widget.projects?.client ?? widget.settings.client();
    projects = widget.projects ?? ProjectsStore(client);
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
      builder: (_) => FloorView(client: client, project: p),
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
      Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => SessionView(store: SessionStore(client, project)),
        ),
      );
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
