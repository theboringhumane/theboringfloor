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
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('opens at the newest message without hiding its full text', (
    tester,
  ) async {
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

    expect(find.text('The final answer is completely visible'), findsOneWidget);
    expect(find.text('Earlier message 0'), findsNothing);
    expect(find.text('Thinking · Earlier message 53'), findsOneWidget);
  });

  testWidgets(
    'keeps member and every assistant message readable while activity toggles',
    (tester) async {
      const member = 'Can you review this change?';
      const olderAssistant = 'Older assistant answer with all of its detail.';
      const newerAssistant = 'Newer assistant answer with all of its detail.';
      const thinking = 'I should inspect the test coverage before responding.';
      const tool = 'bash · gofmt -l internal/backend';
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
            _message('older', 'assistant', 'message', olderAssistant, 2),
            _message('thinking', 'worker', 'wthink', thinking, 3),
            _message('tool', 'worker', 'wtool', tool, 4),
            _message('newer', 'boss', 'message', newerAssistant, 5),
          ],
          'hasMore': false,
        });
      });

      await _pumpSession(tester, store);

      expect(find.text(member), findsOneWidget);
      expect(find.text(olderAssistant), findsOneWidget);
      expect(find.text(newerAssistant), findsOneWidget);
      expect(find.text('Thinking · $thinking'), findsOneWidget);
      expect(find.text(tool), findsOneWidget);

      await tester.tap(find.byKey(const Key('activity-thinking')));
      await tester.pump();
      expect(find.text(thinking), findsOneWidget);
      await tester.tap(find.byKey(const Key('activity-thinking')));
      await tester.pump();
      expect(find.text('Thinking · $thinking'), findsOneWidget);

      await tester.tap(find.byKey(const Key('activity-tool')));
      await tester.pump();
      expect(find.text(tool), findsOneWidget);
      await tester.tap(find.byKey(const Key('activity-tool')));
      await tester.pump();
      expect(find.text(tool), findsOneWidget);

      await tester.tap(find.byKey(const Key('message-member')));
      await tester.tap(find.byKey(const Key('message-older')));
      await tester.pump();
      expect(find.text(member), findsOneWidget);
      expect(find.text(olderAssistant), findsOneWidget);
    },
  );

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
    'messages keep selectable Markdown and offer explicit full copy',
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
      // Markdown selection owns the long press; whole-message copy is explicit.
      await tester.longPress(find.byKey(const Key('message-member')));
      await tester.pump();
      expect(clipboard, isNull);
      expect(find.byType(SelectableRegion), findsNWidgets(2));
      await tester.tap(find.byTooltip('Copy message').at(0));
      await tester.pump();
      expect(clipboard, member);
      expect(find.text('Copied message'), findsOneWidget);
      await tester.tap(find.byTooltip('Copy message').at(1));
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
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
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
