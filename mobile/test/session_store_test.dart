import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/api/gateway_client.dart';
import 'package:theboringfloor/hooks/use_polling.dart';
import 'package:theboringfloor/models/project.dart';
import 'package:theboringfloor/models/session.dart';
import 'package:theboringfloor/models/transcript.dart';
import 'package:theboringfloor/store/session_store.dart';

const _project = Project(
  id: 'project',
  dir: '/project',
  name: 'Project',
  live: true,
  backend: 'opencode',
  primaryId: '',
  port: 0,
  version: '',
  savedAt: 0,
  chatCount: 0,
);

const _status = Status(
  dir: '/project',
  backend: 'opencode',
  primaryId: '',
  planDraftLen: 0,
  planApprovedLen: 0,
  chatCount: 0,
);

TranscriptMessage _message(String id, int at, {String text = 'Message'}) =>
    TranscriptMessage(
      id: id,
      from: 'assistant',
      kind: 'message',
      text: text,
      at: at,
    );

SessionData _session(
  List<TranscriptMessage> messages, {
  bool hasMore = false,
  Status status = _status,
}) => SessionData(status: status, messages: messages, hasMore: hasMore);

class _FakeClient extends GatewayClient {
  _FakeClient(this.sessions, {this.olderPages = const []})
    : super(baseUrl: 'http://unused.test', token: 'token');

  final List<FutureOr<SessionData> Function()> sessions;
  final List<FutureOr<TranscriptPage>> olderPages;
  int sessionCalls = 0;
  int transcriptCalls = 0;

  @override
  Future<SessionData> session(String id) async {
    final index = sessionCalls++;
    return sessions[index < sessions.length ? index : sessions.length - 1]();
  }

  @override
  Future<TranscriptPage> transcript(
    String id, {
    int limit = transcriptPageSize,
    String? before,
  }) async {
    return await olderPages[transcriptCalls++];
  }
}

TranscriptMessage _activity(String id, int at, String kind) =>
    TranscriptMessage(
      id: id,
      from: 'worker',
      kind: kind,
      text: '$kind $id',
      at: at,
    );

void main() {
  test(
    'refresh silently merges three arrivals without duplicate ids',
    () async {
      final client = _FakeClient([
        () => _session([_message('one', 1)]),
        () => _session([
          _message('one', 1),
          _message('two', 2),
          _message('three', 3),
          _message('four', 4),
          _message('two', 2),
        ]),
      ]);
      final store = SessionStore(client, _project);

      await store.load();
      await store.refresh();

      expect(store.loading, isFalse);
      expect(store.messages.map((message) => message.id), [
        'one',
        'two',
        'three',
        'four',
      ]);
      expect(store.newestMessageGeneration, 1);
    },
  );

  test('refresh preserves older pages and their hasMore state', () async {
    final client = _FakeClient(
      [
        () => _session([
          _message('new-1', 3),
          _message('new-2', 4),
        ], hasMore: true),
        () => _session([
          _message('new-1', 3),
          _message('new-2', 4),
          _message('new-3', 5),
        ]),
      ],
      olderPages: [
        TranscriptPage(
          messages: [_message('old-1', 1), _message('old-2', 2)],
          hasMore: true,
        ),
      ],
    );
    final store = SessionStore(client, _project);

    await store.load();
    await store.loadOlder();
    await store.refresh();

    expect(store.messages.map((message) => message.id), [
      'old-1',
      'old-2',
      'new-1',
      'new-2',
      'new-3',
    ]);
    expect(store.hasMore, isTrue);
  });

  test('a no-op refresh does not notify listeners', () async {
    final client = _FakeClient([
      () => _session([_message('one', 1)]),
    ]);
    final store = SessionStore(client, _project);
    await store.load();
    var notifications = 0;
    store.addListener(() => notifications += 1);

    await store.refresh();

    expect(notifications, 0);
  });

  test('keeps a variable-size older page ordered and deduplicated', () async {
    final activity = List.generate(
      501,
      (index) => _activity(
        'activity-$index',
        index + 2,
        index.isEven ? 'wthink' : 'wtool',
      ),
    );
    final client = _FakeClient(
      [
        () => _session([_message('new-user', 600)], hasMore: true),
      ],
      olderPages: [
        TranscriptPage(
          messages: [
            _message('old-user-1', 1),
            ...activity,
            _message('old-user-2', 503),
            _message('new-user', 600),
          ],
          hasMore: true,
        ),
      ],
    );
    final store = SessionStore(client, _project);

    await store.load();
    await store.loadOlder();

    expect(store.messages, hasLength(504));
    expect(store.messages.map((message) => message.id).toSet(), hasLength(504));
    expect(
      store.messages.map((message) => message.at),
      orderedEquals([1, ...List.generate(501, (index) => index + 2), 503, 600]),
    );
    expect(store.hasMore, isTrue);
  });

  test('an empty page with hasMore false ends pagination', () async {
    final client = _FakeClient(
      [
        () => _session([_message('newest', 2)], hasMore: true),
      ],
      olderPages: [const TranscriptPage(messages: [], hasMore: false)],
    );
    final store = SessionStore(client, _project);

    await store.load();
    await store.loadOlder();
    await store.loadOlder();

    expect(store.messages.map((message) => message.id), ['newest']);
    expect(store.hasMore, isFalse);
    expect(store.loadingOlder, isFalse);
    expect(client.transcriptCalls, 1);
    expect(store.newestMessageGeneration, 0);
  });

  test(
    'walks two older pages without duplicate ids or chronology gaps',
    () async {
      final client = _FakeClient(
        [
          () => _session([_message('new', 5)], hasMore: true),
        ],
        olderPages: [
          TranscriptPage(
            messages: [_message('middle-1', 3), _message('middle-2', 4)],
            hasMore: true,
          ),
          TranscriptPage(
            messages: [_message('old-1', 1), _message('middle-1', 3)],
            hasMore: false,
          ),
        ],
      );
      final store = SessionStore(client, _project);

      await store.load();
      await store.loadOlder();
      await store.loadOlder();

      expect(store.messages.map((message) => message.id), [
        'old-1',
        'middle-1',
        'middle-2',
        'new',
      ]);
      expect(store.messages.map((message) => message.at), [1, 3, 4, 5]);
      expect(store.hasMore, isFalse);
    },
  );

  test('drops concurrent older-page requests', () async {
    final pending = Completer<TranscriptPage>();
    final client = _FakeClient(
      [
        () => _session([_message('newest', 2)], hasMore: true),
      ],
      olderPages: [pending.future],
    );
    final store = SessionStore(client, _project);
    await store.load();

    final first = store.loadOlder();
    final second = store.loadOlder();
    expect(client.transcriptCalls, 1);
    expect(store.loadingOlder, isTrue);

    pending.complete(
      TranscriptPage(messages: [_message('older', 1)], hasMore: false),
    );
    await Future.wait([first, second]);

    expect(client.transcriptCalls, 1);
    expect(store.loadingOlder, isFalse);
    expect(store.messages.map((message) => message.id), ['older', 'newest']);
  });

  test('overlapping refreshes are dropped', () async {
    final pending = Completer<SessionData>();
    final client = _FakeClient([
      () => _session([_message('one', 1)]),
      () => pending.future,
    ]);
    final store = SessionStore(client, _project);
    await store.load();

    final first = store.refresh();
    final second = store.refresh();
    expect(client.sessionCalls, 2);
    pending.complete(_session([_message('one', 1), _message('two', 2)]));
    await Future.wait([first, second]);

    expect(client.sessionCalls, 2);
    expect(store.messages.last.id, 'two');
  });

  test('refresh does not race an in-flight older-page request', () async {
    final older = Completer<TranscriptPage>();
    final client = _FakeClient(
      [
        () => _session([_message('newest', 2)], hasMore: true),
        () => _session([_message('newest', 2), _message('fresh', 3)]),
      ],
      olderPages: [older.future],
    );
    final store = SessionStore(client, _project);
    await store.load();

    final loadingOlder = store.loadOlder();
    final refreshed = await store.refresh();

    expect(refreshed, isTrue);
    expect(client.sessionCalls, 1);
    older.complete(
      TranscriptPage(messages: [_message('old', 1)], hasMore: false),
    );
    await loadingOlder;
    expect(store.messages.map((message) => message.id), ['old', 'newest']);
  });

  test(
    'a failed refresh preserves state and later refreshes recover',
    () async {
      final client = _FakeClient([
        () => _session([_message('one', 1)]),
        () => Future<SessionData>.error(StateError('offline')),
        () => _session([_message('one', 1), _message('two', 2)]),
      ]);
      final store = SessionStore(client, _project);
      await store.load();

      await store.refresh();
      expect(store.messages.map((message) => message.id), ['one']);
      expect(store.error, isNull);
      await store.refresh();

      expect(store.messages.map((message) => message.id), ['one', 'two']);
    },
  );

  testWidgets('Polling drops overlapping ticks and stops after dispose', (
    tester,
  ) async {
    final firstTick = Completer<void>();
    var ticks = 0;
    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: Polling(
          interval: const Duration(seconds: 1),
          onTick: () {
            ticks += 1;
            return firstTick.future.then((_) => true);
          },
          child: const SizedBox(),
        ),
      ),
    );

    await tester.pump(const Duration(seconds: 3));
    expect(ticks, 1);
    firstTick.complete();
    await tester.pump();
    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 3));

    expect(ticks, 1);
  });

  testWidgets('Polling pauses in the background and stops after failures', (
    tester,
  ) async {
    var ticks = 0;
    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: Polling(
          interval: const Duration(seconds: 1),
          onTick: () async {
            ticks += 1;
            return false;
          },
          child: const SizedBox(),
        ),
      ),
    );

    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
    await tester.pump(const Duration(seconds: 10));
    expect(ticks, 0);

    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pump(const Duration(seconds: 1));
    await tester.pump();
    await tester.pump(const Duration(seconds: 2));
    await tester.pump();
    await tester.pump(const Duration(seconds: 4));
    await tester.pump();
    await tester.pump(const Duration(seconds: 20));

    expect(ticks, 3);
  });
}
