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
  final List<TranscriptPage> olderPages;
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
    return olderPages[transcriptCalls++];
  }
}

void main() {
  test('refresh is silent and merges newly arriving messages', () async {
    final client = _FakeClient([
      () => _session([_message('one', 1)]),
      () => _session([_message('one', 1), _message('two', 2)]),
    ]);
    final store = SessionStore(client, _project);

    await store.load();
    await store.refresh();

    expect(store.loading, isFalse);
    expect(store.messages.map((message) => message.id), ['one', 'two']);
  });

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
            return firstTick.future;
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
}
