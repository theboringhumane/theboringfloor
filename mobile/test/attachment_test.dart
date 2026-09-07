import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:theboringfloor/api/gateway_client.dart';
import 'package:theboringfloor/components/attachment_picker.dart';
import 'package:theboringfloor/models/attachment.dart';
import 'package:theboringfloor/models/project.dart';
import 'package:theboringfloor/store/session_store.dart';
import 'package:theboringfloor/views/session_view.dart';

class _CapturingClient extends http.BaseClient {
  _CapturingClient(this.handler);
  final Future<http.Response> Function(http.BaseRequest) handler;
  String? body;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    body = request is http.Request ? request.body : null;
    final response = await handler(request);
    return http.StreamedResponse(
      Stream.value(response.bodyBytes),
      response.statusCode,
    );
  }
}

class _Picker implements AttachmentPicker {
  _Picker(this.attachments);
  final List<Attachment> attachments;
  @override
  Future<List<Attachment>> pick(AttachmentPickSource source) async =>
      attachments;
}

const _project = Project(
  id: 'project',
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

Attachment _png(String name, [int bytes = 3]) =>
    Attachment(name: name, mimeType: 'image/png', bytes: Uint8List(bytes));

void main() {
  test('Attachment.toJson uses standard base64 contract', () {
    expect(_png('dot.png').toJson(), {
      'name': 'dot.png',
      'mimeType': 'image/png',
      'data': 'AAAA',
    });
  });

  test('message without attachments retains the legacy JSON body', () async {
    final httpClient = _CapturingClient((_) async => http.Response('', 200));
    final client = GatewayClient(
      baseUrl: 'http://test',
      token: 'token',
      httpClient: httpClient,
    );
    await client.message('project', 'hello');
    expect(httpClient.body, '{"text":"hello"}');
  });

  test('message sends attachment contract body', () async {
    final httpClient = _CapturingClient((_) async => http.Response('', 200));
    final client = GatewayClient(
      baseUrl: 'http://test',
      token: 'token',
      httpClient: httpClient,
    );
    await client.message(
      'project',
      'hello',
      attachments: [_png('one.png'), _png('two.png')],
    );
    expect(jsonDecode(httpClient.body!), {
      'text': 'hello',
      'attachments': [
        {'name': 'one.png', 'mimeType': 'image/png', 'data': 'AAAA'},
        {'name': 'two.png', 'mimeType': 'image/png', 'data': 'AAAA'},
      ],
    });
  });

  test('attachment validation rejects count, MIME, and oversize images', () {
    expect(
      () => AttachmentValidator.validate(
        List.generate(5, (index) => _png('$index.png')),
      ),
      throwsA(isA<AttachmentValidationException>()),
    );
    expect(
      () => AttachmentValidator.validate([
        Attachment(
          name: 'file.pdf',
          mimeType: 'application/pdf',
          bytes: Uint8List(1),
        ),
      ]),
      throwsA(isA<AttachmentValidationException>()),
    );
    expect(
      () => AttachmentValidator.validate([
        _png('large.png', maxAttachmentBytes + 1),
      ]),
      throwsA(isA<AttachmentValidationException>()),
    );
  });

  testWidgets(
    'attachments enable empty composer and failed send preserves them',
    (tester) async {
      final sent = Completer<http.Response>();
      final httpClient = _CapturingClient((request) async {
        if (request.url.path.endsWith('/status')) {
          return http.Response(jsonEncode(_status()), 200);
        }
        if (request.url.path.endsWith('/busy')) {
          return http.Response(jsonEncode({'error': 'missing'}), 404);
        }
        if (request.url.path.endsWith('/transcript')) {
          return http.Response(
            jsonEncode({'messages': [], 'hasMore': false}),
            200,
          );
        }
        return sent.future;
      });
      final store = SessionStore(
        GatewayClient(
          baseUrl: 'http://test',
          token: 'token',
          httpClient: httpClient,
        ),
        _project,
      );
      await tester.pumpWidget(
        MaterialApp(
          home: SessionView(
            store: store,
            attachmentPicker: _Picker([_png('one.png')]),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('Add attachment'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Photo library'));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('attachment-chip-one.png')), findsOneWidget);
      final sendButton = find.ancestor(
        of: find.byIcon(Icons.send_outlined),
        matching: find.byType(IconButton),
      );
      expect(tester.widget<IconButton>(sendButton).onPressed, isNotNull);
      await tester.tap(find.byTooltip('Send follow-up'));
      await tester.pump();
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      sent.complete(
        http.Response(jsonEncode({'error': 'attachment rejected'}), 400),
      );
      await tester.pumpAndSettle();
      expect(find.text('attachment rejected'), findsOneWidget);
      expect(find.text('Try again'), findsOneWidget);
      expect(find.byKey(const Key('attachment-chip-one.png')), findsOneWidget);
    },
  );

  testWidgets('413 response renders the friendly image-size error', (
    tester,
  ) async {
    final httpClient = _CapturingClient((request) async {
      if (request.url.path.endsWith('/status')) {
        return http.Response(jsonEncode(_status()), 200);
      }
      if (request.url.path.endsWith('/busy')) {
        return http.Response(jsonEncode({'error': 'missing'}), 404);
      }
      if (request.url.path.endsWith('/transcript')) {
        return http.Response(
          jsonEncode({'messages': [], 'hasMore': false}),
          200,
        );
      }
      return http.Response(jsonEncode({'error': 'attachments too large'}), 413);
    });
    final store = SessionStore(
      GatewayClient(
        baseUrl: 'http://test',
        token: 'token',
        httpClient: httpClient,
      ),
      _project,
    );
    await tester.pumpWidget(
      MaterialApp(
        home: SessionView(
          store: store,
          attachmentPicker: _Picker([_png('one.png')]),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Add attachment'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Photo library'));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Send follow-up'));
    await tester.pumpAndSettle();
    expect(find.text('Image too large to send.'), findsOneWidget);
  });
}

Map<String, Object?> _status() => {
  'dir': '/workspace',
  'backend': 'opencode',
  'primaryId': '',
  'planDraftLen': 0,
  'planApprovedLen': 0,
  'chatCount': 0,
};
