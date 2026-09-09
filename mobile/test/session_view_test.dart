import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:theboringfloor/api/gateway_client.dart';
import 'package:theboringfloor/models/project.dart';
import 'package:theboringfloor/store/session_store.dart';
import 'package:theboringfloor/views/session_view.dart';

class _Client extends http.BaseClient {
  _Client(this.handler);
  final Future<http.StreamedResponse> Function(http.BaseRequest) handler;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) =>
      handler(request);
}

http.StreamedResponse _response(int status, Map<String, Object?> body) =>
    http.StreamedResponse(
      Stream.value(utf8.encode(jsonEncode(body))),
      status,
      headers: {'content-type': 'application/json'},
    );

const _project = Project(
  id: 'project-1',
  dir: '/workspace',
  name: 'Workspace',
  live: true,
  backend: 'opencode',
  primaryId: '',
  port: 0,
  version: '',
  savedAt: 0,
  chatCount: 0,
);

Map<String, Object?> _message(
  String id,
  String from,
  String kind,
  String text,
  int at,
) => {'id': id, 'from': from, 'kind': kind, 'text': text, 'at': at};

Map<String, Object?> _status() => {
  'dir': '/workspace',
  'backend': 'opencode',
  'primaryId': '',
  'planDraftLen': 0,
  'planApprovedLen': 0,
  'chatCount': 0,
};

Map<String, Object?> _busy(bool value) => {
  'busy': value,
  'pendingBoss': false,
  'thinking': value,
  'delegating': false,
  'questionParked': false,
};

SessionStore _store(
  Future<http.StreamedResponse> Function(http.BaseRequest) handler,
) => SessionStore(
  GatewayClient(
    baseUrl: 'http://gateway.test',
    token: 'token',
    httpClient: _Client(handler),
  ),
  _project,
);

Future<void> _pumpSession(WidgetTester tester, SessionStore store) async {
  await tester.pumpWidget(MaterialApp(home: SessionView(store: store)));
  // A working chip deliberately has a perpetual word-rotation animation, so
  // settling is not a valid way to await the initial asynchronous load.
  await tester.pump();
  await tester.pump();
}

void main() {
  testWidgets('polls a visible working session and stops after dispose', (
    tester,
  ) async {
    var transcriptRequests = 0;
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(200, _busy(true));
      }
      transcriptRequests += 1;
      return _response(200, {
        'messages': [
          _message('one', 'assistant', 'message', 'Working answer', 1),
        ],
        'hasMore': false,
      });
    });

    await _pumpSession(tester, store);
    expect(transcriptRequests, 1);

    await tester.pump(const Duration(seconds: 3));
    await tester.pump();
    expect(transcriptRequests, 2);

    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 6));
    expect(transcriptRequests, 2);
  });

  testWidgets('polls idle sessions slowly for work started elsewhere', (
    tester,
  ) async {
    var transcriptRequests = 0;
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(200, _busy(false));
      }
      transcriptRequests += 1;
      return _response(200, {'messages': [], 'hasMore': false});
    });

    await _pumpSession(tester, store);
    await tester.pump(const Duration(seconds: 20));

    expect(transcriptRequests, 2);
  });

  testWidgets('does not queue a second poll while the first is in flight', (
    tester,
  ) async {
    final pendingRefresh = Completer<http.StreamedResponse>();
    var transcriptRequests = 0;
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(200, _busy(true));
      }
      transcriptRequests += 1;
      if (transcriptRequests == 1) {
        return _response(200, {'messages': [], 'hasMore': false});
      }
      return pendingRefresh.future;
    });

    await _pumpSession(tester, store);
    await tester.pump(const Duration(seconds: 3));
    await tester.pump();
    expect(transcriptRequests, 2);

    await tester.pump(const Duration(seconds: 9));
    expect(transcriptRequests, 2);
    pendingRefresh.complete(_response(200, {'messages': [], 'hasMore': false}));
    await tester.pump();
  });

  testWidgets(
    'opens at the newest message by construction and keeps it through layout',
    (tester) async {
      final messages = List.generate(
        55,
        (index) => _message(
          'm-$index',
          index == 54 ? 'assistant' : 'user',
          index == 53 ? 'wthink' : 'message',
          index == 54
              ? 'The final answer is completely visible'
              : 'Earlier message $index',
          index,
        ),
      );
      final store = _store((request) async {
        if (request.url.path.endsWith('/status')) {
          return _response(200, _status());
        }
        if (request.url.path.endsWith('/busy')) {
          return _response(404, {'error': 'missing'});
        }
        return _response(200, {'messages': messages, 'hasMore': true});
      });

      await _pumpSession(tester, store);

      final transcript = tester.widget<ListView>(find.byType(ListView));
      expect(transcript.reverse, isTrue);
      expect(
        find.text('The final answer is completely visible'),
        findsOneWidget,
      );
      expect(find.text('Earlier message 0'), findsNothing);
      expect(find.text('1 thought'), findsNothing);

      // Markdown and other content may finish sizing after the initial frame.
      // A reversed transcript remains at offset zero without an anchoring jump.
      await tester.pump();
      await tester.pump();
      expect(
        find.text('The final answer is completely visible'),
        findsOneWidget,
      );
    },
  );

  testWidgets('a refresh keeps a history reader in place and offers latest', (
    tester,
  ) async {
    var refreshed = false;
    final messages = List.generate(
      55,
      (index) => _message(
        'message-$index',
        'user',
        'message',
        'History message $index',
        index,
      ),
    );
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(404, {'error': 'missing'});
      }
      return _response(200, {
        'messages': [
          ...messages,
          if (refreshed)
            _message(
              'latest',
              'assistant',
              'message',
              'Fresh newest answer',
              56,
            ),
        ],
        'hasMore': false,
      });
    });

    await _pumpSession(tester, store);
    await tester.drag(find.byType(ListView), const Offset(0, 10000));
    await tester.pumpAndSettle();
    expect(
      tester.widget<ListView>(find.byType(ListView)).controller!.offset,
      greaterThan(0),
    );
    expect(find.text('History message 0'), findsOneWidget);

    refreshed = true;
    await store.refresh();
    await tester.pumpAndSettle();

    // A reader remains at their history, with an explicit escape hatch.
    expect(find.text('History message 0'), findsOneWidget);
    expect(find.text('Fresh newest answer'), findsNothing);
    expect(find.byKey(const Key('jump-to-latest')), findsOneWidget);

    await tester.tap(find.byKey(const Key('jump-to-latest')));
    await tester.pumpAndSettle();

    expect(
      tester.widget<ListView>(find.byType(ListView)).controller!.offset,
      0,
    );
    expect(find.byKey(const Key('jump-to-latest')), findsNothing);
    expect(find.text('Fresh newest answer'), findsOneWidget);
  });

  testWidgets('a live arrival follows the newest end smoothly', (tester) async {
    var refreshed = false;
    final messages = List.generate(
      55,
      (index) => _message(
        'message-$index',
        'user',
        'message',
        'Message $index',
        index,
      ),
    );
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(200, _busy(false));
      }
      return _response(200, {
        'messages': [
          ...messages,
          if (refreshed)
            _message('latest', 'assistant', 'message', 'Live arrival', 56),
        ],
        'hasMore': false,
      });
    });

    await _pumpSession(tester, store);
    expect(
      tester.widget<ListView>(find.byType(ListView)).controller!.offset,
      0,
    );

    refreshed = true;
    await store.refresh();
    await tester.pump(const Duration(milliseconds: 120));
    await tester.pump(const Duration(milliseconds: 120));

    expect(
      tester.widget<ListView>(find.byType(ListView)).controller!.offset,
      0,
    );
    expect(find.text('Live arrival'), findsOneWidget);
  });

  testWidgets('rapid live arrivals coalesce scroll animations', (tester) async {
    var refreshCount = 0;
    final messages = List.generate(
      55,
      (index) => _message(
        'message-$index',
        'user',
        'message',
        'Message $index',
        index,
      ),
    );
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(200, _busy(false));
      }
      refreshCount += 1;
      return _response(200, {
        'messages': [
          ...messages,
          if (refreshCount >= 2)
            _message('first', 'assistant', 'message', 'First arrival', 56),
          if (refreshCount >= 3)
            _message('second', 'assistant', 'message', 'Second arrival', 57),
        ],
        'hasMore': false,
      });
    });

    await _pumpSession(tester, store);
    final controller = tester
        .widget<ListView>(find.byType(ListView))
        .controller!;
    controller.jumpTo(40);
    await store.refresh();
    await tester.pump();
    await store.refresh();
    await tester.pump(const Duration(milliseconds: 240));
    await tester.pumpAndSettle();

    expect(
      tester.widget<ListView>(find.byType(ListView)).controller!.offset,
      0,
    );
    expect(find.text('First arrival'), findsNothing);
    expect(find.text('Second arrival'), findsOneWidget);
  });

  testWidgets('a refresh after the transcript is disposed is safe', (
    tester,
  ) async {
    var refreshed = false;
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(200, _busy(false));
      }
      return _response(200, {
        'messages': [
          _message('one', 'assistant', 'message', 'One', 1),
          if (refreshed) _message('two', 'assistant', 'message', 'Two', 2),
        ],
        'hasMore': false,
      });
    });

    await _pumpSession(tester, store);
    await tester.pumpWidget(const SizedBox());
    refreshed = true;

    await expectLater(store.refresh(), completes);
  });

  testWidgets('member messages offer read more while short messages do not', (
    tester,
  ) async {
    const shortMember = 'A short member message.';
    final longMember = List.filled(
      8,
      'A long member message remains available in full.',
    ).join(' ');
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(404, {'error': 'missing'});
      }
      return _response(200, {
        'messages': [
          _message('short', 'user', 'message', shortMember, 1),
          _message('long', 'user', 'message', longMember, 2),
        ],
        'hasMore': false,
      });
    });

    await _pumpSession(tester, store);

    expect(find.text('Read more'), findsOneWidget);
    expect(find.byKey(Key('member-expand-$shortMember')), findsNothing);
    final memberText = tester.widget<Text>(find.text(longMember));
    expect(memberText.maxLines, 5);

    await tester.tap(find.text('Read more'));
    await tester.pump();
    expect(find.text('Show less'), findsOneWidget);
    expect(tester.widget<Text>(find.text(longMember)).maxLines, isNull);
    await tester.drag(find.byType(ListView), const Offset(0, -500));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Show less'));
    await tester.pump();
    expect(tester.widget<Text>(find.text(longMember)).maxLines, 5);
  });

  testWidgets(
    'member and assistant messages copy their full raw text on long press',
    (tester) async {
      const member = 'A full member message that is copied exactly.';
      const assistant = 'Full assistant text remains copyable.';
      final store = _store((request) async {
        if (request.url.path.endsWith('/status')) {
          return _response(200, _status());
        }
        if (request.url.path.endsWith('/busy')) {
          return _response(404, {'error': 'missing'});
        }
        return _response(200, {
          'messages': [
            _message('member', 'user', 'message', member, 1),
            _message('assistant', 'assistant', 'message', assistant, 2),
          ],
          'hasMore': false,
        });
      });
      String? clipboard;
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(SystemChannels.platform, (call) async {
            if (call.method == 'Clipboard.setData') {
              clipboard =
                  (call.arguments as Map<Object?, Object?>)['text'] as String?;
            }
            return null;
          });
      addTearDown(() {
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
            .setMockMethodCallHandler(SystemChannels.platform, null);
      });

      await _pumpSession(tester, store);
      await tester.longPress(find.byKey(const Key('message-member')));
      await tester.pump();
      expect(clipboard, member);
      expect(find.text('Copied message'), findsOneWidget);
      await tester.longPress(find.byKey(const Key('message-assistant')));
      await tester.pump();
      expect(clipboard, assistant);
    },
  );

  testWidgets('prepends older pages without losing the retained viewport', (
    tester,
  ) async {
    final requests = <Uri>[];
    final olderResponse = Completer<http.StreamedResponse>();
    final older = List.generate(
      50,
      (index) => _message('old-$index', 'user', 'message', 'Old $index', index),
    );
    final newest = List.generate(
      50,
      (index) => _message(
        'new-$index',
        index == 49 ? 'assistant' : 'user',
        'message',
        index == 49 ? 'Newest answer' : 'New $index',
        index + 50,
      ),
    );
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(404, {'error': 'missing'});
      }
      requests.add(request.url);
      return request.url.queryParameters['before'] == null
          ? _response(200, {'messages': newest, 'hasMore': true})
          : olderResponse.future;
    });

    await _pumpSession(tester, store);
    await tester.drag(find.byType(ListView), const Offset(0, 10000));
    await tester.pump();
    // Further scroll notifications while the older page is pending must not
    // issue another request for the same opaque cursor.
    await tester.drag(find.byType(ListView), const Offset(0, 10000));
    await tester.pump();
    // The reversed list's far end is the conversation's visual top.
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(
      requests.where((uri) => uri.queryParameters['before'] != null),
      hasLength(1),
    );
    expect(find.text('New 0'), findsOneWidget);
    olderResponse.complete(
      _response(200, {'messages': older, 'hasMore': false}),
    );
    await tester.pumpAndSettle();

    expect(store.messages, hasLength(100));
    expect(store.hasMore, isFalse);
    expect(requests.last.queryParameters['before'], 'new-0');
    expect(find.text('New 0'), findsOneWidget);
  });

  testWidgets('overflow keeps stop and new session actions reachable', (
    tester,
  ) async {
    final requests = <String>[];
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(404, {'error': 'missing'});
      }
      requests.add('${request.method} ${request.url.path}');
      return _response(200, {'messages': [], 'hasMore': false});
    });

    await _pumpSession(tester, store);
    await tester.tap(find.byTooltip('More'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Stop'));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('More'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('New session'));
    await tester.pumpAndSettle();

    expect(requests, contains('POST /api/v1/projects/project-1/stop'));
    expect(requests, contains('POST /api/v1/projects/project-1/new'));
  });

  test('a malformed cursor stops future paging without a retry loop', () async {
    var transcriptRequests = 0;
    final store = _store((request) async {
      if (request.url.path.endsWith('/status')) {
        return _response(200, _status());
      }
      if (request.url.path.endsWith('/busy')) {
        return _response(404, {'error': 'missing'});
      }
      transcriptRequests += 1;
      return transcriptRequests == 1
          ? _response(200, {
              'messages': [_message('oldest', 'user', 'message', 'One', 1)],
              'hasMore': true,
            })
          : _response(400, {'error': 'bad cursor'});
    });

    await store.load();
    await store.loadOlder();
    await store.loadOlder();

    expect(store.hasMore, isFalse);
    expect(transcriptRequests, 2);
  });
}
