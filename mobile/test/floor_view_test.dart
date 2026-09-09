import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/api/gateway_client.dart';
import 'package:theboringfloor/models/floor.dart';
import 'package:theboringfloor/models/project.dart';
import 'package:theboringfloor/models/session.dart';
import 'package:theboringfloor/models/transcript.dart';
import 'package:theboringfloor/theme.dart';
import 'package:theboringfloor/store/session_store.dart';
import 'package:theboringfloor/views/session_view.dart';
import 'package:theboringfloor/views/floor_view.dart';
import 'package:theboringfloor/views/floor_plan_view.dart';

const project = Project(
  id: 'p',
  dir: '/projects/platform',
  name: 'Developer platform',
  live: true,
  backend: 'opencode',
  primaryId: 'old',
  port: 0,
  version: '0.6.0',
  savedAt: 0,
  chatCount: 4,
);

class FloorClient extends GatewayClient {
  FloorClient() : super(baseUrl: 'http://unused', token: 'test');
  Map<String, dynamic>? ticket;
  Map<String, dynamic>? action;
  String active = 'old';
  String backend = 'opencode';
  FloorPlan currentPlan = const FloorPlan();
  @override
  Future<FloorData> floor(String id) async => FloorData(
    name: 'Developer platform',
    teams: const [
      FloorTeam('ui', 'UI'),
      FloorTeam('frontend', 'Frontend'),
      FloorTeam('backend', 'Backend'),
    ],
    tickets: [
      FloorTicket({
        'id': 'TKT-21',
        'title': 'Ship project navigation',
        'status': 'in-progress',
        'priority': 'P1',
        'team': 'frontend',
        'owner': 'maya',
      }),
    ],
    conversations: const [
      FloorConversation(
        id: 'old',
        backend: 'codex',
        title: 'Project file explorer',
        team: 'frontend',
        messages: 12,
      ),
      FloorConversation(
        id: 'api',
        backend: 'claudecode',
        title: 'API authentication',
        team: 'backend',
        messages: 8,
      ),
    ],
  );
  @override
  Future<void> saveTicket(String id, Map<String, dynamic> value) async {
    ticket = value;
  }

  @override
  Future<void> addTeam(String id, String name) async {}
  @override
  Future<FloorPlan> plan(String id) async => currentPlan;
  @override
  Future<void> workspaceAction(String id, Map<String, dynamic> value) async {
    action = value;
    if (value['action'] == 'conversation') {
      active = 'new';
      backend = value['backend'] as String;
    }
    if (value['action'] == 'plan-approve') {
      currentPlan = FloorPlan(
        draft: currentPlan.draft,
        approved: currentPlan.draft,
      );
    }
  }

  @override
  Future<Status> status(String id) async => Status(
    dir: project.dir,
    backend: backend,
    primaryId: active,
    planDraftLen: 0,
    planApprovedLen: 0,
    chatCount: 4,
    planPending:
        currentPlan.draft.isNotEmpty &&
        currentPlan.draft != currentPlan.approved,
    planRevision: currentPlan.draft,
  );
  @override
  Future<SessionData> session(String id) async => SessionData(
    status: await status(id),
    messages: const [
      TranscriptMessage(
        id: 'u',
        from: 'user',
        kind: 'user',
        text: 'Build the project explorer.',
        at: 1,
      ),
      TranscriptMessage(
        id: 'progress',
        from: 'boss',
        kind: 'boss',
        text: 'I will inspect the files.',
        at: 2,
      ),
      TranscriptMessage(
        id: 'tool',
        from: 'boss',
        kind: 'tool',
        text: 'bash · cat secret.log',
        at: 3,
      ),
      TranscriptMessage(
        id: 'a',
        from: 'boss',
        kind: 'boss',
        text: 'The explorer is ready. Open folders, preview source, and attach a file to your next conversation.',
        at: 4,
      ),
    ],
  );
  @override
  Future<ProjectFilePage> files(String id, String path) async =>
      path == 'src/main.dart'
      ? const ProjectFilePage(
          path: 'src/main.dart',
          directory: false,
          entries: [],
          content: 'void main() {\n  runApp(Office());\n}',
        )
      : ProjectFilePage(
          path: path,
          directory: true,
          content: '',
          entries: path == ''
              ? const [
                  ProjectFile(path: 'src', name: 'src', directory: true),
                  ProjectFile(
                    path: 'README.md',
                    name: 'README.md',
                    directory: false,
                  ),
                ]
              : const [
                  ProjectFile(
                    path: 'src/main.dart',
                    name: 'main.dart',
                    directory: false,
                  ),
                ],
        );
}

Future<void> pumpFloor(WidgetTester tester, FloorClient client) async {
  await tester.binding.setSurfaceSize(const Size(412, 892));
  await tester.pumpWidget(
    MaterialApp(
      debugShowCheckedModeBanner: false,
      theme: buildLightTheme(),
      home: FloorView(client: client, project: project),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> shot(WidgetTester tester, String name) async {
  if (!const bool.fromEnvironment('FLOOR_SHOTS')) return;
  await expectLater(
    find.byType(MaterialApp),
    matchesGoldenFile(Uri.file('/tmp/floor-mobile-shots/$name.png')),
  );
}

void main() {
  setUpAll(() async {
    if (!const bool.fromEnvironment('FLOOR_SHOTS')) return;
    final icons = FontLoader('MaterialIcons')
      ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'));
    await icons.load();
    for (final entry in {
      'Inter': 'Inter-Regular.ttf',
      'Space Grotesk': 'SpaceGrotesk-Regular.ttf',
      'Playfair Display': 'PlayfairDisplay-Regular.ttf',
      'JetBrains Mono': 'JetBrainsMono-Regular.ttf',
    }.entries) {
      final loader = FontLoader(entry.key)
        ..addFont(rootBundle.load('assets/fonts/${entry.value}'));
      await loader.load();
    }
  });
  testWidgets('floor offers teams, conversations, board, and ticket editing', (
    tester,
  ) async {
    final client = FloorClient();
    await pumpFloor(tester, client);
    expect(find.text('Frontend'), findsOneWidget);
    expect(find.text('Project file explorer'), findsOneWidget);
    await shot(tester, 'conversations');
    await tester.tap(find.text('Board'));
    await tester.pumpAndSettle();
    expect(find.text('Ship project navigation'), findsOneWidget);
    await shot(tester, 'board');
    await tester.tap(find.text('Ticket'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.widgetWithText(TextField, 'Title'),
      'Build Android file browser',
    );
    await tester.ensureVisible(find.text('Save ticket'));
    await tester.tap(find.text('Save ticket'));
    await tester.pumpAndSettle();
    expect(client.ticket?['title'], 'Build Android file browser');
    expect(client.ticket?['status'], 'backlog');
  });
  testWidgets('file tree expands and previews source without executing it', (
    tester,
  ) async {
    await pumpFloor(tester, FloorClient());
    await tester.tap(find.text('Files'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('src'));
    await tester.pumpAndSettle();
    expect(find.text('main.dart'), findsOneWidget);
    await shot(tester, 'files');
    await tester.tap(find.text('main.dart'));
    await tester.pumpAndSettle();
    expect(find.textContaining('runApp(Office())'), findsOneWidget);
    await shot(tester, 'file-preview');
  });
  testWidgets(
    'new conversation chooses its backend and opens a clean transcript',
    (tester) async {
      final client = FloorClient();
      await pumpFloor(tester, client);
      await tester.tap(find.text('New'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('OpenCode'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Codex').last);
      await tester.pumpAndSettle();
      await shot(tester, 'new-conversation');
      await tester.tap(find.text('Start conversation'));
      await tester.pumpAndSettle();
      await tester.pump(const Duration(milliseconds: 400));
      await tester.pumpAndSettle();
      expect(client.action?['backend'], 'codex');
      expect(client.action?['fresh'], true);
      expect(find.textContaining('The explorer is ready.'), findsOneWidget);
      expect(find.textContaining('secret.log'), findsNothing);
      expect(find.text('I will inspect the files.'), findsNothing);
      await shot(tester, 'transcript');
    },
  );
  testWidgets('chat automatically presents a new plan and respects dismissal', (
    tester,
  ) async {
    final client = FloorClient()
      ..currentPlan = const FloorPlan(
        draft: '# Build safely\n\n1. Review the API.\n2. Verify behavior.',
      );
    await tester.pumpWidget(
      MaterialApp(
        theme: buildLightTheme(),
        home: SessionView(store: SessionStore(client, project)),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Plan before building'), findsOneWidget);
    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.text('Plan before building'), findsNothing);
    expect(find.text('Review plan'), findsOneWidget);
  });
  testWidgets(
    'live plan updates preserve edits and approval uses the reviewed draft',
    (tester) async {
      final client = FloorClient()
        ..currentPlan = const FloorPlan(
          draft: '# Project file explorer\n\n1. Add a file tree.\n2. Preview source.\n3. Verify project boundaries.',
        );
      await tester.binding.setSurfaceSize(const Size(412, 892));
      await tester.pumpWidget(
        MaterialApp(
          debugShowCheckedModeBanner: false,
          theme: buildLightTheme(),
          home: Scaffold(
            appBar: AppBar(title: const Text('Review plan')),
            body: FloorPlanView(client: client, project: project),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await shot(tester, 'plan');
      await tester.tap(find.byTooltip('Edit plan'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), '# My unsaved edits');
      client.currentPlan = const FloorPlan(draft: '# Revised server plan');
      await tester.pump(const Duration(seconds: 4));
      await tester.pump();
      expect(find.text('# My unsaved edits'), findsOneWidget);
      expect(find.text('Load latest'), findsOneWidget);
      await tester.tap(find.text('Load latest'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Approve & build'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Approve & build').last);
      await tester.pumpAndSettle();
      expect(client.action?['expected'], '# Revised server plan');
      expect(find.text('Approved'), findsOneWidget);
    },
  );
}
