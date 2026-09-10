import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/models/floor.dart';
import 'package:theboringfloor/models/project.dart';
import 'package:theboringfloor/models/session.dart';
import 'package:theboringfloor/store/attention_store.dart';
import 'package:theboringfloor/theme.dart';
import 'package:theboringfloor/views/attention_view.dart';
import 'package:theboringfloor/views/ticket_view.dart';

import 'floor_view_test.dart' as fixture;

class InboxClient extends fixture.FloorClient {
  bool failProjects = false, failStatus = false;
  int inflight = 0, maximum = 0;
  int count = 3;
  @override
  Future<List<Project>> projects() async {
    if (failProjects) throw StateError('offline');
    return List.generate(
      count,
      (i) => Project(
        id: '$i',
        dir: '/projects/$i',
        name: ['Developer platform', 'Commerce API', 'Design system'][i % 3],
        live: true,
        backend: 'codex',
        primaryId: 'session-$i',
        port: 0,
        version: '0.7.0',
        savedAt: 0,
        chatCount: 0,
      ),
    );
  }

  @override
  Future<Status> status(String id) async {
    if (failStatus) throw StateError('offline');
    return Status(
      dir: '/projects/$id',
      backend: 'codex',
      primaryId: 'session-$id',
      planDraftLen: id == '0' ? 10 : 0,
      planApprovedLen: 0,
      chatCount: 1,
      planPending: id == '0',
      execution: ExecutionStatus(
        state: id == '0' ? 'plan' : 'working',
        summary: id == '0'
            ? 'A plan is ready for your review'
            : 'The team is working on delegated tasks',
        action: id == '0' ? 'plan' : '',
      ),
    );
  }

  @override
  Future<FloorData> floor(String id) async {
    inflight++;
    if (inflight > maximum) maximum = inflight;
    await Future<void>.delayed(const Duration(milliseconds: 1));
    inflight--;
    return FloorData(
      name: 'Developer platform',
      teams: const [FloorTeam('frontend', 'Frontend')],
      conversations: const [],
      tickets: id == '1'
          ? [
              FloorTicket({
                'id': 'TKT-42',
                'title': 'Ship team invitations',
                'status': 'review',
                'priority': 'P1',
                'owner': 'Maya',
                'team': 'frontend',
                'updated': 42,
                'description': 'Invite teammates with expiring links and clear error states.',
                'result':
                    'Invitation flow and expiry handling are ready for review.',
                'verification': 'Recorded by Maya: unit checks passed. Phone review remains.',
                'backend': 'codex',
                'session': 'session-1',
                'checklist': [
                  {
                    'text': 'Expired invitations show a useful error',
                    'done': true,
                  },
                  {'text': 'Review on a phone', 'done': false},
                ],
              }),
            ]
          : [],
    );
  }
}

void main() {
  test('inbox bounds requests, keeps stale snapshots, and distinguishes review from running', () async {
    final client = InboxClient()..count = 9;
    final store = AttentionStore(client);
    expect(await store.refresh(), isTrue);
    expect(client.maximum, lessThanOrEqualTo(4));
    expect(store.needsAttention, 2);
    expect(store.working, 8);
    client.failProjects = true;
    expect(await store.refresh(), isFalse);
    expect(store.floors.length, 9);
    expect(store.error, contains('out of date'));
    client.failProjects = false;
    client.failStatus = true;
    expect(await store.refresh(), isTrue);
    expect(
      store.floors.every((p) => p.status == null && p.problem != null),
      isTrue,
    );
    store.dispose();
  });

  setUpAll(() async {
    for (final font in {
      'Inter': 'Inter-Regular.ttf',
      'Space Grotesk': 'SpaceGrotesk-Regular.ttf',
      'Playfair Display': 'PlayfairDisplay-Regular.ttf',
      'JetBrains Mono': 'JetBrainsMono-Regular.ttf',
    }.entries) {
      final loader = FontLoader(font.key)
        ..addFont(rootBundle.load('assets/fonts/${font.value}'));
      await loader.load();
    }
    final icons = FontLoader('MaterialIcons')
      ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'));
    await icons.load();
  });

  testWidgets(
    'inbox opens ticket review and prepares a linked handoff without sending',
    (tester) async {
      tester.view.physicalSize = const Size(412, 892);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final client = InboxClient();
      final store = AttentionStore(client);
      await tester.pumpWidget(
        MaterialApp(
          debugShowCheckedModeBanner: false,
          theme: buildLightTheme(),
          home: Scaffold(body: AttentionView(store: store)),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('2 floors need you'), findsOneWidget);
      expect(find.text('Design system'), findsNothing);
      await fixture.shot(tester, 'inbox');
      await tester.scrollUntilVisible(find.text('Ship team invitations'), 250);
      await tester.tap(find.text('Ship team invitations'));
      await tester.pumpAndSettle();
      expect(find.text('Acceptance criteria'), findsOneWidget);
      expect(find.text('1 of 2 checked'), findsOneWidget);
      await fixture.shot(tester, 'ticket-review');
      await tester.scrollUntilVisible(
        find.text('Prepare in live chat'),
        250,
        scrollable: find
            .descendant(
              of: find.byType(TicketView),
              matching: find.byType(Scrollable),
            )
            .first,
      );
      await tester.tap(find.text('Prepare in live chat'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 500));
      expect(client.action?['action'], 'ticket-link');
      expect(client.action?['ticketId'], 'TKT-42');
      expect(client.action?['expectedUpdated'], 42);
      expect(
        find
            .byType(TextField)
            .evaluate()
            .any(
              (e) =>
                  (e.widget as TextField).controller?.text.contains(
                    'Acceptance criteria:',
                  ) ==
                  true,
            ),
        isTrue,
      );
      await tester.pumpWidget(const SizedBox());
      store.dispose();
    },
  );
  testWidgets(
    'inbox remains readable on a narrow dark screen and exposes refresh errors',
    (tester) async {
      tester.view.physicalSize = const Size(360, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final client = InboxClient();
      final store = AttentionStore(client);
      await tester.pumpWidget(
        MaterialApp(
          debugShowCheckedModeBanner: false,
          theme: buildDarkTheme(),
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context)
                .copyWith(textScaler: const TextScaler.linear(1.3)),
            child: child!,
          ),
          home: Scaffold(body: AttentionView(store: store)),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      await fixture.shot(tester, 'inbox-dark');
      client.failProjects = true;
      await store.refresh();
      await tester.pumpAndSettle();
      expect(find.text('Connection needs attention'), findsOneWidget);
      expect(store.floors.length, 3);
      await tester.pumpWidget(const SizedBox());
      store.dispose();
    },
  );
}
